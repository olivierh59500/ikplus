package presentation

import (
	"github.com/olivierh59500/ikplus/internal/audio"
	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/pilot"
)

// requestHumanStart maps platform menu selection to the original F1/F2 poll.
// The source IRQ owns its deadline, abandoned program and arena reset.
func (a *App) requestHumanStart(players int) {
	a.CancelInput()
	a.match.QueueSourceKey(0x3b + byte(players-1))
}

// sourceJoin replaces the interrupted program after the real keyboard/joystick
// poll, including platform menu requests mapped to original F1/F2.
func (a *App) sourceJoin(players int) {
	p := a.ambient.planes
	if a.presentation == nil {
		a.materializeArenaPictures()
		a.match.CopyPresentationState(&p.state)
	}
	mode := p.state.Mode
	video := p.state.Displayed
	if a.match.PALFrameClocks() == audio.Supplied313.FrameClocks() {
		a.beginHumanJoin(players, mode)
		return
	}
	palette, paletteMode := a.titleColors, paletteArena
	if mode == 8 {
		// The IRQ's 1060 branch publishes the black title list before its
		// fabricated RTE reaches AD4 and restores both complete backgrounds.
		paletteMode = paletteTitle
		for index := range palette {
			palette[index] = p.artwork.Title.Black[index]
		}
		p.state.BlackScreen = 1
		for _, bank := range p.screens() {
			copy(bank.pixels[:32000], p.artwork.Background[:32000])
		}
	}
	p.state.Humans[0], p.state.Humans[1], p.state.JoinPending = 1, byte(players-1), 1
	a.CancelInput()
	a.match.SourceJoin(&p.state, players)
	a.paintArenaResetHeader(hudState{Round: 1, Seconds: 30, Mode: 1, Speed: a.match.DisplaySpeed(), Humans: (1 << players) - 1, German: p.state.German != 0})
	// The old title/score/name program must not publish or resume afterward.
	a.presentation, p.presentation = nil, nil
	a.players, a.view, a.returnView, a.paused, a.scoreRecorded = players, matchView, matchView, false, false
	a.eliminatedScores = [2]*score{}
	a.scrollKey, a.ambient.history = 0, 0
	a.pilot = pilot.New(0)
	a.ambient.AmbientState = a.match.Ambient()
	a.ambient.lastFrame = a.match.Frame
	a.match.PrepareJoinedArena()
	_, drawing := a.match.SceneryBuffers()
	a.ambient.videoBank = drawing
	base := &p.pictures[drawing]
	retained := append([]byte(nil), base.pixels[:32000]...)
	a.scene.compose(a.match, &a.ambient, a.currentHUDState())
	copy(base.pixels[:32000], p.working.pixels[:32000])
	if video == game.PictureSecond {
		a.ambient.videoBank = 1
	} else {
		a.ambient.videoBank = 0
	}
	p.state.Displayed = video
	frame := titleSequence{effects: titleEffects{planes: p, hardware: palette, paletteMode: paletteMode}}
	a.handoffPixels = append(a.handoffPixels[:0], frame.pixels()...)
	// Arena composition prepares actors in its working bank on every draw.
	// Only the completed join picture needs this materialized actor snapshot.
	copy(base.pixels[:32000], retained)
	a.match.EndPresentationUpdate(true)
}

func (a *App) paintArenaResetHeader(state hudState) {
	p := a.ambient.planes
	h := newPresentationClock(p)
	for _, bank := range p.screens() {
		h.restoreBottom(bank)
	}
	for id := 0; id < 2; id++ {
		if state.Humans&(1<<id) == 0 {
			h.hand(id, false)
		}
	}
	for _, bank := range p.screens() {
		p.spriteAt(bank, 144, 10, 5, p.artwork.Sprites[p.artwork.SpeedLabel[0]], false, false, false)
		p.spriteAt(bank, 152, 10, 5, p.artwork.Sprites[p.artwork.SpeedLabel[1]], false, false, false)
	}
	p.paintSpeed(state.Speed)
	for _, draw := range hudLayout(state) {
		if draw.Image != "" {
			continue
		}
		for row := 0; row < 8; row++ {
			mask := p.artwork.HUDFont[draw.Glyph][row]
			for plane := 0; plane < 4; plane++ {
				value := byte(0)
				if draw.Color&(1<<plane) != 0 {
					value = mask
				}
				for _, bank := range p.screens() {
					bank.pixels[(draw.Y+row)*160+draw.X/8+plane*40] = value
				}
			}
		}
	}
	p.state.ClockGlyph = 6
}
