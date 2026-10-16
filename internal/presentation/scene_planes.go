package presentation

import (
	"encoding/binary"
	"image"
	"math"
	"math/bits"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/graphics"
)

// originalScene composes the recovered scene in packed planes. Shadows can
// then modify earlier artwork with the original bitwise OR, and all artwork
// receives the actual scanline palette rather than a fixed atlas palette.
type originalScene struct {
	image         *ebiten.Image
	rgba          []byte
	indices       []byte
	captionOutput []byte
	captionPixels *image.RGBA
	planes        *sceneryPlanes
	hintBase      []byte
	hintMask      []byte
}

func (s *originalScene) initialize() {
	if s.rgba != nil {
		return
	}
	s.rgba = make([]byte, 320*200*4)
	s.indices, s.captionOutput = make([]byte, 64000), make([]byte, 64000)
	s.captionPixels = image.NewRGBA(image.Rect(0, 0, 320, 200))
}

func (s *originalScene) frame(id, frame, rawX int, mirrored bool) {
	s.frameAt(&s.planes.working, id, frame, rawX, mirrored, nil, true)
}

// frameAt also supplies the original one-plane collision coverage used as
// the title's reduced fighter picture source, without running contacts.
func (s *originalScene) frameAt(bank *packedBitmap, id, frame, rawX int, mirrored bool, mask *packedBitmap, shadow bool) {
	s.frameRowsAt(bank, id, frame, rawX, mirrored, mask, 0, -1)
	if shadow {
		s.frameShadowAt(bank, frame, rawX, mirrored)
	}
}

func (s *originalScene) frameRowsAt(bank *packedBitmap, id, frame, rawX int, mirrored bool, mask *packedBitmap, first, last int) {
	if frame < 0 || frame >= 88 {
		return
	}
	pose := &s.planes.artwork.Fighters[frame]
	rawX &= 255
	shift := (rawX & 7) * 2
	base := 96*160 - 4 + (rawX&248)/4
	ordinal := 0
	for _, run := range pose.Runs {
		offset := run.Y*160 + run.X/8
		if mirrored {
			offset = offset/160*160 - offset%160 + 8
		}
		for row, data := range run.Rows {
			ordinal++
			if ordinal <= first || last >= 0 && ordinal > last {
				continue
			}
			var authored [3]uint32
			for plane := range authored {
				word := data[plane]
				if mirrored {
					word = bits.Reverse16(word)
				}
				authored[plane] = bits.RotateLeft32(uint32(word)<<16, -shift)
			}
			opacity := authored[0] | authored[1] | authored[2]
			if mask != nil {
				offset := base + offset + row*160
				at := offset/160*40 + offset%160
				mask.putLong(at, mask.long(at)|opacity)
			}
			ink := [4]uint32{0, authored[0], authored[1], authored[2]}
			if id == 1 {
				ink[0] = authored[1]
			} else if id == 2 {
				ink[0], ink[2] = authored[1], 0
			}
			for plane, value := range ink {
				at := base + offset + row*160 + plane*40
				bank.putLong(at, bank.long(at)&^opacity|value)
			}
		}
	}
}

func (s *originalScene) frameShadowAt(bank *packedBitmap, frame, rawX int, mirrored bool) {
	if frame < 0 || frame >= 88 {
		return
	}
	pose := &s.planes.artwork.Fighters[frame]
	rawX &= 255
	shift := (rawX & 7) * 2
	base := 96*160 - 4 + (rawX&248)/4
	// $1CF6 mirrors five shadow words per row, reversing their bit order.
	for row := 0; row < 8; row++ {
		for column := 0; column < 5; column++ {
			from := column
			if mirrored {
				from = 4 - column
			}
			word := pose.Shadow[row][from]
			if mirrored {
				word = bits.Reverse16(word)
			}
			rotated := bits.RotateLeft32(uint32(word), -shift)
			at := base + 0x3c00 + row*160 + column*2
			bank.putWord(at, bank.word(at)|uint16(rotated))
			bank.putWord(at+2, bank.word(at+2)|uint16(rotated>>16))
		}
	}
}

func (s *originalScene) caption(state game.CaptionState, text bool) {
	if !state.Visible {
		return
	}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			index := byte(0)
			for plane := 0; plane < 4; plane++ {
				index |= (s.planes.working.pixels[y*160+plane*40+x/8] >> uint(7-x%8) & 1) << plane
			}
			s.indices[y*320+x] = index
		}
	}
	clear(s.captionPixels.Pix)
	paintOriginalCaptionParts(s.captionPixels, state, s.indices, s.captionOutput, text)
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			if s.captionPixels.Pix[(y*320+x)*4+3] == 0 {
				continue
			}
			mask, index := byte(128>>uint(x&7)), s.captionOutput[y*320+x]
			for plane := 0; plane < 4; plane++ {
				at := y*160 + plane*40 + x/8
				s.planes.working.pixels[at] &^= mask
				if index&(1<<plane) != 0 {
					s.planes.working.pixels[at] |= mask
				}
			}
		}
	}
}

func (s *originalScene) fighter(f game.Fighter, picture game.PreparedPicture) {
	if len(picture.BallActorErases) == 0 || f.ID != picture.Active {
		s.frame(f.ID, f.SpriteFrame, int(f.X/2), f.Facing < 0)
	} else {
		first := 0
		for _, erase := range picture.BallActorErases {
			s.frameRowsAt(&s.planes.working, f.ID, f.SpriteFrame, int(f.X/2), f.Facing < 0, nil, first, erase.Rows)
			ball := erase.Ball
			s.planes.paintSpriteAt(&s.planes.working, int(ball.X)+24, int(ball.Y)-6, 11, s.ballSprite(ball), false, true, false, true)
			first = erase.Rows
		}
		s.frameRowsAt(&s.planes.working, f.ID, f.SpriteFrame, int(f.X/2), f.Facing < 0, nil, first, -1)
		s.frameShadowAt(&s.planes.working, f.SpriteFrame, int(f.X/2), f.Facing < 0)
	}
	s.fighterEffects(f)
	if picture.Mode == 5 && picture.ShieldVisible && f.ID == picture.Active {
		s.shield(f, f.ShieldAlternate)
	}
	if picture.Mode == 7 && f.ID == picture.Active {
		s.kickMarker(f)
	}
}

func (s *originalScene) kickMarker(f game.Fighter) {
	if x, y, active := game.KickMarker(f); active {
		s.planes.spriteAt(&s.planes.working, x+32, y, 1, s.planes.artwork.Sprites[s.planes.artwork.KickMarker], false, true, false)
	}
}

func (s *originalScene) fighterEffects(f game.Fighter) {
	s.visitFighterEffects(f, func(effect game.FighterEffect) {
		s.planes.spriteAt(&s.planes.working, effect.X, effect.Y, len(effect.Sprite.Rows), effect.Sprite, false, false, false)
	})
}

// The same queued sprite rectangles are used when $AE58 restores these effects.
func (s *originalScene) visitFighterEffects(f game.Fighter, visit func(game.FighterEffect)) {
	game.VisitFighterEffects(f, visit)
}

func (s *originalScene) shield(f game.Fighter, alternate bool) {
	pose := 0
	if f.ShieldType == 2 {
		pose = 1
	} else if f.ShieldType == 1 {
		pose = 2
		if alternate {
			pose = 3
		}
	}
	cells := s.planes.artwork.Shields[pose]
	source := s.planes.artwork.Sprites[cells[0]]
	s.planes.spriteAt(&s.planes.working, int(f.ShieldX)+32, int(f.ShieldY), 23, source, false, true, false)
	if f.ShieldType == 1 {
		s.planes.spriteAt(&s.planes.working, int(f.ShieldX)+48, int(f.ShieldY), 23, s.planes.artwork.Sprites[cells[1]], false, true, false)
	}
}

func (s *originalScene) bomb(bomb game.Bomb) {
	if bomb.Fragment {
		source := s.fragmentSprite(bomb)
		s.planes.spriteAt(&s.planes.working, int(bomb.X)+24, int(bomb.Y)-6, 9, source, false, true, true)
	} else if bomb.Stage == 5 {
		if bomb.BlastFrame >= 73 && bomb.BlastFrame <= 81 {
			rawX := int(math.Floor((bomb.X+24)/2)) - 8
			s.frame(2, bomb.BlastFrame, rawX, false)
		}
	} else {
		s.planes.spriteAt(&s.planes.working, int(bomb.X)+24, int(bomb.Y)-6, 10, s.planes.artwork.Sprites[s.planes.artwork.BombBody], false, true, true)
		source := s.bombFuseSprite(bomb)
		s.planes.spriteAt(&s.planes.working, int(bomb.X)+24, int(bomb.Y)-14, 13, source, false, true, false)
	}
}

func (s *originalScene) ballSprite(ball game.Ball) graphics.Sprite {
	data := s.planes.artwork
	return data.Sprites[data.Balls[min(7, max(0, ball.Color))][byte(ball.Frame)]]
}

func (s *originalScene) fragmentSprite(bomb game.Bomb) graphics.Sprite {
	data := s.planes.artwork
	return data.Sprites[data.Fragments[bomb.Variant&3][bomb.Frame&7]]
}

func (s *originalScene) bombFuseSprite(bomb game.Bomb) graphics.Sprite {
	data := s.planes.artwork
	return data.Sprites[data.BombFuses[min(35, max(0, bomb.StageTicks))]]
}

func (s *originalScene) render(g *game.Game, ambient *ambientState, hud hudState) {
	s.compose(g, ambient, hud)
	if s.image == nil {
		s.image = ebiten.NewImage(320, 200)
	}
	s.image.WritePixels(s.rgba)
}

func (s *originalScene) compose(g *game.Game, ambient *ambientState, hud hudState) {
	s.composePicture(g, ambient, hud, g.DisplayPicture(ambient.videoBank))
}

// Preparation comparisons can inspect a completed native picture before its
// display handoff. Normal rendering always supplies the published picture.
func (s *originalScene) composePicture(g *game.Game, ambient *ambientState, hud hudState, picture game.PreparedPicture) {
	s.initialize()
	s.planes = ambient.planes
	if cleared := g.SceneClearFrame(); cleared != 0 && cleared == g.Frame && cleared != ambient.lastSceneClear {
		for _, bank := range s.planes.screens() {
			copy(bank.pixels[117*160:32000], s.planes.artwork.Background[117*160:32000])
		}
		ambient.lastSceneClear = cleared
	}
	copy(s.planes.working.pixels[:32000], s.planes.pictures[ambient.videoBank].pixels[:32000])
	for row := 192; row < 200; row++ {
		clear(s.planes.working.pixels[row*160 : row*160+40])
	}
	bonus := picture.Mode == 5 || picture.Mode == 7
	if picture.HintActive {
		newPresentationClock(s.planes).restoreBottom(&s.planes.working)
	} else if picture.HintBacking != nil {
		s.hintBacking(picture)
	}
	// The interrupt draws particles before the main thread reuses a bank.
	// Between such preparations, particles are drawn over its retained actors.
	particlesFirst := picture.Valid && picture.Frame == g.Frame
	if particlesFirst {
		s.particles(g, ambient.videoBank)
		s.erasePreviousPicture(picture.Previous)
		// $AE4A clears the ground after this interrupt's particle shadows.
		for row := 192; row < 200; row++ {
			clear(s.planes.working.pixels[row*160 : row*160+40])
		}
	}
	if bonus {
		s.caption(picture.Caption, picture.CaptionText)
	}
	// $6492 draws blue first, then the last white/red scorer last.
	order := [3]int{2, int(picture.Priority&1) ^ 1, int(picture.Priority & 1)}
	if picture.Mode == 6 {
		order = [3]int{2, 1, 0}
	}
	for _, id := range order {
		fighter := picture.Fighters[id]
		if picture.Valid && (!bonus || id == picture.Active) {
			s.fighter(fighter, picture)
			if picture.Mode == 6 {
				fighter.X += float64([3]int{0x1e, 0x44, 0x68}[id] * 2)
				s.fighter(fighter, picture)
			}
		}
	}
	for _, blast := range picture.Blasts {
		if blast.Active {
			s.bomb(blast)
		}
	}
	if picture.EraseCaption && picture.CaptionText {
		s.eraseCaption(picture.EraseAnchor)
	}
	if bonus && (picture.Mode == 7 || picture.Caption.Visible) {
		s.projectShadows()
	}
	if !bonus {
		s.caption(picture.Caption, picture.CaptionText)
		if picture.HintActive && picture.HintDraw && s.planes.artwork.HintColors[picture.Hint.Index&31] == 15 {
			s.hint(picture)
			s.projectShadows()
		} else {
			s.projectShadows()
			if picture.HintDraw {
				s.hint(picture)
			}
		}
	}
	if picture.EraseCaption && !picture.CaptionText {
		// $82FA also modifies the retained bank. Its old referee/sparkles
		// already exist when the speech strip and masked tail are restored.
		s.eraseCaption(picture.EraseAnchor)
	}
	if !particlesFirst {
		s.particles(g, ambient.videoBank)
	}
	s.hud(hud)
	colors := g.DisplayBeltColors()
	for y := 0; y < 200; y++ {
		palette := ambient.palettes[y]
		for x := 0; x < 320; x++ {
			index := byte(0)
			for plane := 0; plane < 4; plane++ {
				index |= (s.planes.working.pixels[y*160+plane*40+x/8] >> uint(7-x%8) & 1) << plane
			}
			color := palette[index]
			if y >= 77 && y < 115 && index == 10 {
				color = g.PublishedGraphicsColor()
			}
			if y >= 192 && index == 15 {
				// $18D8 selects $333, copied to paletteMode COLOR15 by $EC4.
				color = g.PublishedGroundShade()
			}
			if y >= 115 {
				switch index {
				case 1:
					if y < 192 {
						color = g.EffectColor()
					}
				case 12:
					color = colors[0]
				case 13:
					color = colors[1]
				case 9:
					color = colors[2]
				}
			}
			at := (y*320 + x) * 4
			s.rgba[at], s.rgba[at+1], s.rgba[at+2], s.rgba[at+3] = byte(color>>8&15)*17, byte(color>>4&15)*17, byte(color&15)*17, 255
		}
	}
}

// $AE58 restores every queued RLE run as a four-byte rectangle, including
// one extra row. It can erase parts of particles drawn by the preceding IRQ.
func (s *originalScene) erasePreviousPicture(previous game.PictureEraseState) {
	if !previous.Valid {
		return
	}
	for id, fighter := range previous.Fighters {
		if previous.Active >= 0 && id != previous.Active {
			continue
		}
		s.eraseFrame(fighter.SpriteFrame, int(fighter.X/2), fighter.Facing < 0)
		if previous.Mode == 6 {
			s.eraseFrame(fighter.SpriteFrame, int(fighter.X/2)+[3]int{0x1e, 0x44, 0x68}[id], fighter.Facing < 0)
		}
		s.visitFighterEffects(fighter, func(effect game.FighterEffect) {
			s.eraseSpriteRectangle(effect.X, effect.Y, len(effect.Sprite.Rows), false)
		})
		if previous.Mode == 7 && id == previous.Active {
			if x, y, active := game.KickMarker(fighter); active {
				s.eraseSpriteRectangle(x+32, y, 1, true)
			}
		}
	}
	if previous.Caption.Visible {
		s.eraseFrame(int(previous.Caption.RefereeFrame), int(previous.Caption.RefereeX), false)
		if previous.CaptionText && previous.Caption.Previous < 40 {
			s.eraseSpriteRectangle(int(previous.Caption.Anchor<<2+19), 138, 7, false)
		}
		for slot, draw := range previous.Caption.Draw {
			if draw {
				x := previous.Caption.Anchor<<2 + 14 + previous.Caption.X[slot]
				s.eraseSpriteRectangle(int(x), int(previous.Caption.Y[slot])+140, 5, false)
			}
		}
	}
	if previous.Mode == 5 && previous.ShieldVisible && previous.Active >= 0 {
		fighter := previous.Fighters[previous.Active]
		s.eraseSpriteRectangle(int(fighter.ShieldX)+32, int(fighter.ShieldY), 23, true)
		if fighter.ShieldType == 1 {
			s.eraseSpriteRectangle(int(fighter.ShieldX)+48, int(fighter.ShieldY), 23, true)
		}
	}
	for _, blast := range previous.Blasts {
		if blast.Active && blast.BlastFrame >= 73 && blast.BlastFrame <= 81 {
			rawX := int(math.Floor((blast.X+24)/2)) - 8
			s.eraseFrame(blast.BlastFrame, rawX, false)
		}
	}
}

func (s *originalScene) eraseFrame(frame, rawX int, mirrored bool) {
	if frame < 0 || frame >= 88 {
		return
	}
	base := 0x3bfc + (rawX&248)/4
	for _, run := range s.planes.artwork.Fighters[frame].Runs {
		offset := run.Y*160 + run.X/8
		if mirrored {
			offset = offset/160*160 - offset%160 + 8
		}
		s.planes.restore(savedDrawing{&s.planes.working, base + offset}, len(run.Rows)+1)
	}
}

func (s *originalScene) eraseSpriteRectangle(x, y, height int, pixelX bool) {
	if pixelX {
		x = (x & 511) >> 1
	} else {
		x &= 255
	}
	if x&248 < 8 || x&248 >= 176 || y < 0 || y >= 200 {
		return
	}
	if y+height >= 201 {
		height = 200 - y
	}
	address := ((y*160 + (x&248)/4) & 0x7ffe) - 4
	s.planes.restore(savedDrawing{&s.planes.working, address}, height+1)
}

func (s *originalScene) eraseCaption(anchor byte) {
	for row := 117; row < 138; row++ {
		copy(s.planes.working.pixels[row*160:(row+1)*160], s.planes.artwork.Background[row*160:(row+1)*160])
	}
	x := int(byte(anchor<<2 + 19))
	s.planes.paintSpriteAt(&s.planes.working, x, 138, 7, s.planes.artwork.Sprites[s.planes.artwork.CaptionTail], false, false, false, true)
}

func (s *originalScene) particles(g *game.Game, bank int) {
	for _, operation := range g.BallPictureOperations(bank) {
		ball := operation.Ball
		if operation.Kind == game.BallGroundDraw {
			s.projectileShadow(ball.X, ball.Y, 187)
			continue
		}
		source := s.ballSprite(ball)
		s.planes.paintSpriteAt(&s.planes.working, int(ball.X)+24, int(ball.Y)-6, 11, source, false, true, false, true)
	}
	if g.BallPictureVisible(bank) && (g.Phase == game.BallBonus || g.Phase == game.BonusEnd && len(g.Bonus.Balls) != 0) {
		for slot := len(g.Bonus.Balls) - 1; slot >= 0; slot-- {
			ball := g.Bonus.Balls[slot]
			if ball.Active {
				source := s.ballSprite(ball)
				// $C2D2 passes eleven rows; the atlas inspection cell has twelve.
				s.planes.spriteAt(&s.planes.working, int(ball.X)+24, int(ball.Y)-6, 11, source, false, true, false)
			}
		}
	}
	erases := g.BombPictureOperations(bank)
	var current [10]*game.BombPictureOperation
	for start := 0; start < len(erases) && erases[start].Frame < g.Frame; {
		end := start + 1
		for end < len(erases) && erases[end].Frame == erases[start].Frame {
			end++
		}
		// The old black coverage already exists when an IRQ starts. Replaying
		// it at a later slot's restore would erase an earlier slot's new ink.
		for _, operation := range erases[start:end] {
			if operation.Kind == game.BombMaskRestore && operation.PaintedAfterActors {
				s.bombMask(operation.Bomb)
			}
		}
		for _, operation := range erases[start:end] {
			s.bombPictureOperation(operation)
		}
		start = end
	}
	irqErases := g.CurrentBombOperations()
	for index := range irqErases {
		erase := &irqErases[index]
		if erase.Bank == bank && erase.Kind == game.BombMaskRestore {
			current[erase.Slot] = erase
			if erase.PaintedAfterActors {
				s.bombMask(erase.Bomb)
			}
		}
	}
	for slot, bomb := range g.Bonus.Bombs {
		if current[slot] != nil {
			s.eraseBombPicture(current[slot].Bomb)
		}
		if bomb.Active && bomb.PictureVisible() && bomb.Stage != 5 {
			s.bomb(bomb)
		}
		for _, operation := range irqErases {
			if operation.Bank == bank && operation.Slot == slot && operation.Kind != game.BombMaskRestore {
				s.bombPictureOperation(operation)
			}
		}
	}
}

func (s *originalScene) bombMask(bomb game.Bomb) {
	source, height := s.planes.artwork.Sprites[s.planes.artwork.BombBody], 10
	if bomb.Fragment {
		source, height = s.fragmentSprite(bomb), 9
	}
	s.planes.spriteAt(&s.planes.working, int(bomb.X)+24, int(bomb.Y)-6, height, source, false, true, true)
}

func (s *originalScene) bombPictureOperation(operation game.BombPictureOperation) {
	switch operation.Kind {
	case game.BombMaskRestore:
		s.eraseBombPicture(operation.Bomb)
	case game.BombGroundRestore:
		s.eraseBombShadow(operation.Bomb)
	case game.BombGroundDraw:
		s.projectileShadow(operation.Bomb.X, operation.Bomb.Y, 188)
	}
}

// $D15A and $CE46 call $1C076 to restore the preceding black mask with OR.
func (s *originalScene) eraseBombPicture(bomb game.Bomb) {
	x, y := int(bomb.X)+24, int(bomb.Y)-6
	if bomb.Fragment {
		source := s.fragmentSprite(bomb)
		s.planes.paintSpriteAt(&s.planes.working, x, y, 9, source, false, true, true, true)
		return
	}
	s.planes.paintSpriteAt(&s.planes.working, x, y, 10, s.planes.artwork.Sprites[s.planes.artwork.BombBody], false, true, true, true)
	source := s.bombFuseSprite(bomb)
	s.planes.paintSpriteAt(&s.planes.working, x, y-8, 13, source, false, true, false, true)
}

func (s *originalScene) eraseBombShadow(bomb game.Bomb) {
	base, _, _, visible := bombShadowPlacement(bomb.X, bomb.Y, 188)
	if visible {
		for row := 0; row < 8; row++ {
			at := base + row*160
			s.planes.working.putLong(at, binary.BigEndian.Uint32(s.planes.artwork.Background[at&0x7ffe:]))
		}
	}
}

// Projectile ground ink uses authored cells selected by height above the floor.
// Its placement follows whole horizontal positions, independently of the sprite.
func (s *originalScene) projectileShadow(x, y float64, floor int) {
	placement, visible := graphics.ProjectileGroundShadow(x, y, floor)
	if !visible {
		return
	}
	base := 192*160 + placement.Left/8
	for row := 0; row < 8; row++ {
		shape := uint32(s.planes.artwork.ProjectileShadows[placement.Shape][row]) << 16
		shape = bits.RotateLeft32(shape, -placement.Shift) & s.planes.artwork.EdgeMasks[placement.Edge]
		at := base + row*160
		s.planes.working.putLong(at, s.planes.working.long(at)|shape)
	}
}

func bombShadowPlacement(x, y float64, floor int) (base, mask, height int, visible bool) {
	placement, visible := graphics.ProjectileGroundShadow(x, y, floor)
	if !visible {
		return 0, 0, 0, false
	}
	return 192*160 + placement.Left/8, placement.Edge, placement.Shape, true
}

// $B00A projects the ground's first plane with fixed outward word shifts.
// Left ranges seed X from the low bit; right ranges begin with arithmetic
// shift-right and therefore seed it from the first word's sign bit.
func (s *originalScene) projectShadows() {
	s.projectShadowsAt(&s.planes.working)
}

func (s *originalScene) projectShadowsAt(bank *packedBitmap) {
	for _, group := range shadowProjection {
		first := bank.word(group.first)
		carry, step := first>>15, 2
		if group.left {
			carry, step = first&1, -2
		}
		for at := group.first; ; at += step {
			word := bank.word(at)
			next := word & 1
			value := word>>1 | carry<<15
			if group.left {
				next, value = word>>15, word<<1|carry
			}
			bank.putWord(at, value)
			carry = next
			if at == group.last {
				break
			}
		}
	}
}

// hud uses the original eight-row font and converted plane cells. The same
// independently verified layout is shared with the existing HUD references.
func (s *originalScene) hud(state hudState) {
	clear(s.planes.working.pixels[:16*160])
	// $6CEE leaves COLOR15 behind the masked active hand. COLOR0 and
	// COLOR15 both display black here, but their packed bits must survive.
	for id := 0; id < 2; id++ {
		data := s.planes.artwork
		s.planes.spriteAt(&s.planes.working, data.HandX[id], 0, 15, data.Sprites[data.Hands[0][id]], false, false, false)
	}
	for _, draw := range hudLayout(state) {
		if draw.Image != "" {
			cells, width, height := s.planes.artwork.SpeedLabel, 32, 5
			switch {
			case draw.Image == "hud-speed-label":
			case strings.HasPrefix(draw.Image, "hud-speed-"):
				cells = s.planes.artwork.Speeds[int(draw.Image[len(draw.Image)-1]-'0')]
			default:
				cells[0] = s.planes.artwork.Hands[1][int(draw.Image[len(draw.Image)-1]-'0')]
				width, height = 16, 15
			}
			for x := 0; x < width; x += 16 {
				s.planes.spriteAt(&s.planes.working, draw.X+x+32, draw.Y, height, s.planes.artwork.Sprites[cells[x/16]], false, true, false)
			}
			continue
		}
		for row := 0; row < 8; row++ {
			mask := s.planes.artwork.HUDFont[draw.Glyph][row]
			at := (draw.Y+row)*160 + draw.X/8
			for plane := 0; plane < 4; plane++ {
				if draw.Color&(1<<plane) != 0 {
					s.planes.working.pixels[at+plane*40] = mask
				} else {
					s.planes.working.pixels[at+plane*40] = 0
				}
			}
		}
	}
}
