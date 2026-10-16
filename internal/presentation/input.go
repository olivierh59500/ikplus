package presentation

import (
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/ikplus/internal/game"
)

type touchRole uint8

const (
	ignoredTouch touchRole = iota
	stickTouch
	fireTouch
	musicTouch
	effectsTouch
	volumeDownTouch
	volumeUpTouch
)

// Input ownership is fixed when a finger first touches the screen, so dragging
// the joystick across the fire control cannot steal another finger's action.
type controls struct {
	roles          map[ebiten.TouchID]touchRole
	stickID        ebiten.TouchID
	stickActive    bool
	blocked        bool
	x, y           float64
	fire           bool
	audioPanel     bool
	keys           []byte
	mouseDown      bool
	mouseRole      touchRole
	sidebarBlocked bool
}

func (a *App) CancelInput() {
	a.releaseAudioContacts()
	a.controls = controls{roles: make(map[ebiten.TouchID]touchRole), blocked: true, audioPanel: a.audioOptions}
}

func audioTouchKey(role touchRole) byte {
	switch role {
	case musicTouch:
		return 0x3d
	case effectsTouch:
		return 0x3e
	case volumeDownTouch:
		return 0x0c
	case volumeUpTouch:
		return 0x4e
	}
	return 0
}

func audioTouchRole(x, y int) touchRole {
	if x <= 404 || x >= 476 {
		return ignoredTouch
	}
	switch {
	case y >= 10 && y <= 42:
		return musicTouch
	case y >= 50 && y <= 82:
		return effectsTouch
	case y >= 142 && y <= 174 && x < 440:
		return volumeDownTouch
	case y >= 142 && y <= 174:
		return volumeUpTouch
	}
	return ignoredTouch
}

func (a *App) releaseAudioContacts() {
	for _, id := range orderedContactIDs(a.controls.roles) {
		role := a.controls.roles[id]
		if key := audioTouchKey(role); key != 0 {
			if a.match != nil {
				a.match.QueueSoundKey(key | 0x80)
			}
			a.controls.roles[id] = ignoredTouch
		}
	}
	if key := audioTouchKey(a.controls.mouseRole); key != 0 {
		if a.match != nil {
			a.match.QueueSoundKey(key | 0x80)
		}
		a.controls.mouseRole = ignoredTouch
	}
}

func orderedContactIDs(roles map[ebiten.TouchID]touchRole) []ebiten.TouchID {
	ids := make([]ebiten.TouchID, 0, len(roles))
	for id := range roles {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (a *App) setAudioOptions(open bool) {
	a.releaseAudioContacts()
	a.audioOptions, a.controls.audioPanel = open, open
	a.controls.sidebarBlocked = true
	a.controls.fire = false
	if a.controls.mouseRole == fireTouch {
		a.controls.mouseRole = ignoredTouch
	}
	for id, role := range a.controls.roles {
		if role == fireTouch {
			a.controls.roles[id] = ignoredTouch
		}
	}
}

func pressed(keys ...ebiten.Key) bool {
	for _, key := range keys {
		if ebiten.IsKeyPressed(key) {
			return true
		}
	}
	return false
}

func just(keys ...ebiten.Key) bool {
	for _, key := range keys {
		if inpututil.IsKeyJustPressed(key) {
			return true
		}
	}
	return false
}

func axis(negative, positive bool) int {
	if negative == positive {
		return 0
	}
	if negative {
		return -1
	}
	return 1
}

// joystickStartPlayers follows $69D2/$6A2E for ready keyboard polling in the
// no-human arena, score or title display. White fire has priority over red.
func joystickStartPlayers(frame byte, input [2]game.Input) int {
	if frame < 25 {
		return 0
	}
	if input[0].Fire {
		return 1
	}
	if input[1].Fire {
		return 2
	}
	return 0
}

func (a *App) readInput() [2]game.Input {
	inputs := [2]game.Input{
		{X: axis(pressed(ebiten.KeyLeft), pressed(ebiten.KeyRight)), Y: axis(pressed(ebiten.KeyUp), pressed(ebiten.KeyDown)), Fire: pressed(ebiten.KeySpace, ebiten.KeyControlRight, ebiten.KeyAltRight)},
		{X: axis(pressed(ebiten.KeyA), pressed(ebiten.KeyD)), Y: axis(pressed(ebiten.KeyW), pressed(ebiten.KeyS)), Fire: pressed(ebiten.KeyShiftLeft)},
	}
	ids := ebiten.AppendGamepadIDs(nil)
	for i, id := range ids {
		if i >= 2 {
			break
		}
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		gx, gy := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal), ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		left := gx < -0.35 || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft)
		right := gx > 0.35 || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight)
		up := gy < -0.35 || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop)
		down := gy > 0.35 || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom)
		if left || right {
			inputs[i].X = axis(left, right)
		}
		if up || down {
			inputs[i].Y = axis(up, down)
		}
		inputs[i].Fire = inputs[i].Fire || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom)
		if !a.match.SourceFrozen() && inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			a.togglePause()
		}
	}
	if !a.options.Mobile {
		return inputs
	}
	a.readTouches()
	return a.addTouchInput(inputs)
}

// addTouchInput routes the shared touchscreen controls to the original name
// controller's selected slot, including when both human flags remain set.
func (a *App) addTouchInput(inputs [2]game.Input) [2]game.Input {
	player := 0
	if a.enteringName() {
		player = int(a.ambient.planes.state.CandidateOwner) & 1
	} else if !a.match.Fighters[0].Human && a.match.Fighters[1].Human {
		player = 1
	}
	if a.controls.x != 0 || a.controls.y != 0 {
		inputs[player].X, inputs[player].Y = quantize(a.controls.x), quantize(a.controls.y)
	}
	inputs[player].Fire = inputs[player].Fire || a.controls.fire
	return inputs
}

func quantize(value float64) int {
	if value < -0.32 {
		return -1
	}
	if value > 0.32 {
		return 1
	}
	return 0
}

type contact struct {
	id   ebiten.TouchID
	x, y int
}

func (a *App) readTouches() {
	ids := ebiten.AppendTouchIDs(nil)
	contacts := make([]contact, 0, len(ids))
	for _, id := range ids {
		x, y := ebiten.TouchPosition(id)
		contacts = append(contacts, contact{id: id, x: x, y: y})
	}
	mouseX, mouseY := ebiten.CursorPosition()
	a.controls.update(contacts, ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft), mouseX, mouseY)
	for _, key := range a.controls.keys {
		a.match.QueueSoundKey(key)
	}
}

// update handles snapshots without platform calls so simultaneous ownership and
// lifecycle cancellation can be verified independently of Android's view.
func (c *controls) update(contacts []contact, mouseDown bool, mouseX, mouseY int) {
	c.x, c.y, c.fire = 0, 0, false
	c.keys = c.keys[:0]
	if c.sidebarBlocked {
		rightDown := mouseDown && mouseX > 400
		for _, touch := range contacts {
			rightDown = rightDown || touch.x > 400
		}
		if !rightDown {
			c.sidebarBlocked = false
		}
	}
	if c.roles == nil {
		c.roles = make(map[ebiten.TouchID]touchRole)
	}
	if c.blocked {
		if len(contacts) == 0 && !mouseDown {
			c.blocked = false
		}
		return
	}
	active := make(map[ebiten.TouchID]bool, len(contacts))
	for _, touch := range contacts {
		active[touch.id] = true
	}
	// Remove released ownership before assigning a new contact in this tick.
	for _, id := range orderedContactIDs(c.roles) {
		role := c.roles[id]
		if !active[id] {
			if key := audioTouchKey(role); key != 0 {
				c.keys = append(c.keys, key|0x80)
			}
			delete(c.roles, id)
			if role == stickTouch {
				c.stickActive = false
			}
		}
	}
	for _, touch := range contacts {
		id, x, y := touch.id, touch.x, touch.y
		role, exists := c.roles[id]
		if !exists {
			role = ignoredTouch
			if x >= 0 && x < 80 && y > 112 && y < 256 && !c.stickActive {
				role = stickTouch
				c.stickID = id
				c.stickActive = true
			}
			if c.sidebarBlocked && x > 400 {
				role = ignoredTouch
			} else if c.audioPanel && x > 400 {
				role = audioTouchRole(x, y)
				if key := audioTouchKey(role); key != 0 {
					c.keys = append(c.keys, key)
				}
			} else if x > 400 && x < 480 && y > 142 && y < 256 {
				role = fireTouch
			}
			c.roles[id] = role
		}
		switch role {
		case stickTouch:
			c.x, c.y = stickPosition(x, y)
		case fireTouch:
			if !c.audioPanel {
				c.fire = true
			}
		}
	}
	if !mouseDown && c.mouseDown {
		if key := audioTouchKey(c.mouseRole); key != 0 {
			c.keys = append(c.keys, key|0x80)
		}
		c.mouseRole = ignoredTouch
	}
	if mouseDown && !c.mouseDown {
		c.mouseRole = ignoredTouch
		if mouseX >= 0 && mouseX < 80 && mouseY > 112 && mouseY < 256 {
			c.mouseRole = stickTouch
		} else if !c.sidebarBlocked && c.audioPanel {
			c.mouseRole = audioTouchRole(mouseX, mouseY)
		} else if !c.sidebarBlocked && !c.audioPanel && mouseX > 400 && mouseX < 480 && mouseY > 142 && mouseY < 256 {
			c.mouseRole = fireTouch
		}
		if key := audioTouchKey(c.mouseRole); key != 0 {
			c.keys = append(c.keys, key)
		}
	}
	c.mouseDown = mouseDown
	if mouseDown {
		x, y := mouseX, mouseY
		if c.mouseRole == stickTouch {
			c.x, c.y = stickPosition(x, y)
		}
		if c.mouseRole == fireTouch && !c.audioPanel && !c.sidebarBlocked {
			c.fire = true
		}
	}
}

func stickPosition(x, y int) (float64, float64) {
	dx, dy := float64(x-40)/32, float64(y-192)/32
	distance := math.Hypot(dx, dy)
	if distance > 1 {
		dx /= distance
		dy /= distance
	}
	return dx, dy
}

// pointerPress returns one newly pressed contact; menus never use held touches.
func pointerPress() (int, int, bool) {
	ids := inpututil.AppendJustPressedTouchIDs(nil)
	if len(ids) > 0 {
		x, y := ebiten.TouchPosition(ids[0])
		return x, y, true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return x, y, true
	}
	return 0, 0, false
}
