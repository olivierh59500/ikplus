package presentation

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/olivierh59500/ikplus/internal/game"
	"golang.org/x/image/font/basicfont"
)

var (
	ink   = color.RGBA{205, 205, 205, 255}
	white = color.RGBA{240, 240, 240, 255}
	red   = color.RGBA{238, 51, 85, 255}
	blue  = color.RGBA{68, 68, 238, 255}
	gold  = color.RGBA{238, 204, 68, 255}
	panel = color.RGBA{0, 0, 34, 235}
)

const sceneY = 28

type animation struct {
	Action int   `json:"action"`
	Frames []int `json:"frames"`
}

func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	a.drawScene(screen)
	if a.options.Mobile {
		a.drawTouch(screen)
	}
	switch a.view {
	case titleView:
		a.drawTitle(screen)
	case helpView:
		a.drawHelp(screen)
	case scoresView:
		a.drawScores(screen)
	case matchView:
		if a.paused {
			a.drawPause(screen)
		} else if a.match.Phase == game.PauseDemo {
			a.labelCenter(screen, "FIRE/PAUSE RESUME   P DEMO SOUND", 247, ink)
		}
	}
	if a.saveError != nil {
		a.label(screen, "SAVE/CAPTURE ERROR", a.offset()+8, 250, red)
	}
	a.capture(screen)
	if a.recorder != nil && a.ticks > a.recorder.frames {
		a.recorder.write(screen)
	}
}

func (a *App) drawScene(screen *ebiten.Image) {
	pixels := a.scenePixels()
	var target **ebiten.Image
	if a.presentation != nil {
		target = &a.presentation.image
	} else {
		target = &a.scene.image
	}
	if *target == nil {
		*target = ebiten.NewImage(320, 200)
	}
	(*target).WritePixels(pixels)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(a.offset()), sceneY)
	screen.DrawImage(*target, op)
	if a.presentation != nil || len(a.handoffPixels) != 0 {
		return
	}
	g := a.match
	a.drawHUD(screen)
	if a.view == matchView {
		switch g.Phase {
		case game.GameOver:
			if !g.Caption().Visible {
				a.banner(screen, "GAME OVER", sceneY+88)
			}
			a.labelCenter(screen, "PRESS FIRE OR TAP TO RETURN", 247, ink)
		}
	}
}

// scenePixels is the shared arena/title compositor. Offline exports retain
// these exact pixels before GPU upload, scaling and application overlays.
func (a *App) scenePixels() []byte {
	if a.presentation != nil {
		return a.presentation.pixels(a.ambient.planes)
	}
	if len(a.handoffPixels) != 0 {
		return a.handoffPixels
	}
	a.scene.compose(a.match, &a.ambient, a.currentHUDState())
	return a.scene.rgba
}

func (a *App) drawHUD(screen *ebiten.Image) {
	g := a.match
	if a.view == matchView && !a.paused && g.Phase != game.PauseDemo {
		if g.Phase == game.BallBonus || g.Phase == game.BombBonus {
			label := g.Message
			if !g.BonusPreparing() {
				label = fmt.Sprintf("%s  %d/%d", label, g.Bonus.Number, g.Bonus.Total)
			}
			a.labelCenter(screen, label, 247, ink)
		} else {
			a.labelCenter(screen, "P PAUSE   H MOVES   M SOUND", 247, ink)
		}
	}
}

// shieldImageName follows the source pointer branches at $B9AE. Type 2 selects
// the adjacent right-facing shield, while type 1 is two neutral shield boards.
func shieldImageName(shieldType int, alternate bool) string {
	return game.ShieldAssetName(shieldType, alternate)
}

// bombSpritePosition preserves the separate source origins in $D0B8: the
// black body sits eight pixels below the thirteen-row fuse/highlight layer.
func bombSpritePosition(x, y float64) (left, bodyTop, fuseTop float64) {
	return x - 8, y - 6, y - 14
}

func (a *App) drawTitle(screen *ebiten.Image) {
	if a.options.Mobile {
		a.labelCenter(screen, "INTERNATIONAL KARATE +  DEMO", 18, gold)
	} else {
		a.label(screen, "IK+ DEMO", 8, 18, gold)
	}
	for _, button := range titleActions(a.options.Mobile) {
		c := color.Color(ink)
		if button.Action <= titleTwoPlayers && int(button.Action) == a.selection {
			c = gold
		}
		r := button.Bounds
		a.button(screen, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), button.Label, c)
	}
}

func (a *App) drawPause(screen *ebiten.Image) {
	o := a.offset()
	vector.DrawFilledRect(screen, float32(o+32), 67, 256, 143, panel, false)
	a.labelCenter(screen, "PAUSED", 93, gold)
	a.button(screen, o+40, 114, 240, 35, "RESUME  [P / ENTER]", ink)
	a.button(screen, o+40, 160, 240, 32, "RETURN TO TITLE", ink)
}

func (a *App) drawHelp(screen *ebiten.Image) {
	o := a.offset()
	vector.DrawFilledRect(screen, float32(o+4), 24, 312, 231, panel, false)
	a.labelCenter(screen, "JOYSTICK MOVES", 42, gold)
	lines := []string{"ARROWS + SPACE: WHITE", "W A S D + LEFT SHIFT: RED", "DIRECTIONS RELATIVE TO FACING", "        NORMAL      + FIRE", "UP      JUMP        FLYING KICK", "UP/FWD  FACE PUNCH  HEADBUTT", "FORWARD WALK        STOMACH KICK", "DN/FWD  SHIN KICK   FACE KICK", "DOWN    FOOTSWEEP   REV. SWEEP", "DN/BACK LOW PUNCH   BACKSTEP KICK", "BACK    RETREAT     BACKFLIP", "UP/BACK REV. PUNCH  DOUBLE KICK", "RELEASE TO CHANGE / REPEAT MOVE", "RETREAT TO GUARD CLOSE ATTACKS"}
	for i, line := range lines {
		a.label(screen, line, o+10, 58+i*12, ink)
	}
	a.labelCenter(screen, "BACK / ENTER / TAP TO RETURN", 250, gold)
}

func (a *App) drawScores(screen *ebiten.Image) {
	a.labelCenter(screen, "HIGH SCORES", 18, gold)
	a.labelCenter(screen, "UP/DOWN SCROLL  FIRE START  BACK", 247, ink)
}

func (a *App) drawTouch(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, 80, 256, color.RGBA{6, 9, 24, 255}, false)
	vector.DrawFilledRect(screen, 400, 0, 80, 256, color.RGBA{6, 9, 24, 255}, false)
	if a.audioOptions {
		a.drawAudioOptions(screen)
	} else {
		if a.view == matchView && a.presentation == nil {
			label := "PAUSE"
			if a.match.Phase == game.PauseDemo || a.paused {
				label = "RESUME"
			}
			a.button(screen, 405, 10, 70, 32, label, ink)
		}
		if a.view != titleView {
			a.button(screen, 405, 50, 70, 32, "MENU", ink)
		}
		label := "SOUND"
		if a.muted {
			label = "MUTED"
		}
		a.button(screen, 405, 90, 70, 32, label, ink)
	}
	a.label(screen, "8-WAY", 21, 145, ink)
	vector.StrokeCircle(screen, 40, 192, 34, 1.5, ink, true)
	vector.StrokeLine(screen, 6, 192, 74, 192, 1, color.RGBA{60, 66, 80, 255}, false)
	vector.StrokeLine(screen, 40, 158, 40, 226, 1, color.RGBA{60, 66, 80, 255}, false)
	vector.DrawFilledCircle(screen, 40+float32(a.controls.x*23), 192+float32(a.controls.y*23), 9, gold, true)
	if !a.audioOptions {
		vector.StrokeCircle(screen, 440, 192, 29, 2, ink, true)
		if a.controls.fire {
			vector.DrawFilledCircle(screen, 440, 192, 25, color.RGBA{80, 48, 35, 255}, true)
		}
		fireLabel := "FIRE"
		if a.view == titleView || a.view == scoresView {
			fireLabel = "START"
		} else if a.enteringName() {
			fireLabel = "CONFIRM"
		}
		a.label(screen, fireLabel, 440-len(fireLabel)*7/2, 196, gold)
	}
	a.label(screen, "IK+", 29, 36, gold)
	player := "WHITE"
	if !a.match.Fighters[0].Human && a.match.Fighters[1].Human {
		player = "RED"
	}
	if a.view == scoresView {
		player = "SCROLL"
	} else if a.enteringName() {
		player = "WHITE"
		if a.ambient.planes.state.CandidateOwner == 1 {
			player = "RED"
		}
	}
	if a.view != titleView {
		a.label(screen, player, 40-len(player)*7/2, 58, ink)
	}
}

func (a *App) drawAudioOptions(screen *ebiten.Image) {
	music, effects, mute := "MUSIC OFF", "FX OFF", "MUTE OFF"
	if a.match.MusicEnabled() {
		music = "MUSIC ON"
	}
	if a.match.EffectsEnabled() {
		effects = "FX ON"
	}
	if a.muted {
		mute = "MUTE ON"
	}
	a.button(screen, 405, 10, 70, 32, music, ink)
	a.button(screen, 405, 50, 70, 32, effects, ink)
	a.button(screen, 405, 90, 70, 32, mute, ink)
	a.label(screen, fmt.Sprintf("VOL %02d", a.match.MasterVolume()), 419, 135, gold)
	a.button(screen, 405, 142, 33, 32, "-", ink)
	a.button(screen, 442, 142, 33, 32, "+", ink)
	a.button(screen, 405, 214, 70, 32, "BACK", ink)
}

func (a *App) label(screen *ebiten.Image, label string, x, y int, c color.Color) {
	text.Draw(screen, label, basicfont.Face7x13, x, y, c)
}

func (a *App) labelCenter(screen *ebiten.Image, label string, y int, c color.Color) {
	a.label(screen, label, a.offset()+(320-len(label)*7)/2, y, c)
}
func (a *App) button(screen *ebiten.Image, x, y, w, h int, label string, c color.Color) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), color.RGBA{20, 24, 43, 255}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, c, false)
	a.label(screen, label, x+(w-len(label)*7)/2, y+(h+10)/2, c)
}
func (a *App) banner(screen *ebiten.Image, label string, y int) {
	w := min(308, len(label)*7+24)
	vector.DrawFilledRect(screen, float32(a.offset()+(320-w)/2), float32(y-17), float32(w), 25, ink, false)
	a.labelCenter(screen, label, y, color.RGBA{0, 0, 68, 255})
}
