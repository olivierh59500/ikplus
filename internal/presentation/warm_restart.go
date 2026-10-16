package presentation

import "github.com/olivierh59500/ikplus/internal/game"

import "github.com/olivierh59500/ikplus/internal/audio"

func (a *App) attachPresentation() {
	p := a.ambient.planes
	p.stableDisplay = a.match.StableDisplayEnabled()
	p.publishScoreFrame()
	a.match.AttachPresentation(&p.state, func() { p.paintSpeed(a.match.DisplaySpeed()) })
	p.presentation = &presentationIO{
		pacedScores: a.match.PALFrameClocks() == audio.Supplied313.FrameClocks(),
		randomWord:  a.match.PresentationRandomWord, beginUpdate: a.match.BeginPresentationUpdate,
		beforeWait: a.match.PresentationDisplayWait, mainWait: a.match.SetPresentationWait,
		restartRequested: a.match.RestartRequested,
	}
}

// warmRestart follows the nonreturning $17A0 path after the keyboard poll.
// The existing match keeps its audio clock, mutable scenery and ranked table.
func (a *App) warmRestart() {
	p := a.ambient.planes
	if a.presentation == nil {
		a.materializeArenaPictures()
		a.match.CopyPresentationState(&p.state)
	}
	// $77C6 does not change these pointers. $24AC changes them only after
	// its initial two-VBL wait, so a native arena publication must retain them.
	lists := [2]game.RestorationCursor{p.state.EraseStart, p.state.EraseCursor}
	for i := range p.state.TitlePalette {
		p.state.TitlePalette[i] = p.artwork.Title.Black[i]
	}
	for index := range a.titleColors {
		a.titleColors[index] = p.artwork.Title.Black[index]
	}
	p.state.SoundStep, p.state.Fast = 0, 0
	clear(p.state.Humans[0:3])
	p.state.Speed, p.state.Graphics, p.state.Referee = 2, 0, 0
	p.state.GroundShade = 0x333
	p.state.HintCount = 0
	p.state.TitleLanguage = 0
	p.state.German = 0
	p.state.FontOutline = 0xffff
	p.state.Displayed = game.PictureFirst
	p.state.FreezeLocation = game.WaitArenaSetup
	a.CancelInput()
	// Game owns $77C6/$623C, including exactly three arena RNG calls with
	// the old source VBL. It then enables music and resets the source VBL.
	a.match.WarmRestart(&p.state)
	a.paintRestartPictures()
	a.handoffPixels = nil
	p.state.Frame, p.state.Elapsed, p.state.BufferPhase, p.state.Mode = 0, 0, 4, 8
	p.state.Filter = 0
	p.state.TitleLanguage = 0
	p.state.StartupReserved = 0
	p.state.Background, p.state.Drawing, p.state.Displayed = game.PictureFirst, game.PictureFirst, game.PictureFirst
	p.state.FreezeLocation = game.WaitArenaSetup
	p.state.EraseStart, p.state.EraseCursor = lists[0], lists[1]
	p.state.CollisionRead, p.state.CollisionReadMirror, p.state.CollisionWrite, p.state.CollisionWriteMirror = game.CoverageSecond, game.CoverageSecondMirrored, game.CoverageFirst, game.CoverageFirstMirrored
	p.state.CollisionEnabled = 0
	p.state.SecondaryCollisionEnabled = 0
	// Replace the abandoned program directly. Normal score completion has
	// already called $623C; this warm path did so inside Game.WarmRestart.
	a.presentation = &presentationCycle{rgba: make([]byte, 320*200*4)}
	a.attachPresentation()
	a.constructNativeTitle()
	a.presentation.title.effects.paletteMode = paletteArena // $19C4/$F24 precedes the first title IRQ.
	a.players, a.selection, a.scrollKey = 1, 0, 0
	a.view, a.returnView, a.paused, a.scoreRecorded = titleView, titleView, false, false
	a.eliminatedScores = [2]*score{}
	a.ambient.history = 0
	a.ambient.AmbientState = presentationAmbient(p)
	a.ambient.lastFrame, a.ambient.videoBank = a.match.Frame, 0
	// The source abandoned its interrupted program before any remaining
	// Copper/main work. Finish the existing journal once with the new mode.
	a.match.EndPresentationUpdate(true)
}

// paintRestartPictures mirrors $ADAE and $77C6's header drawing. Unrelated
// pixels above row117 survive until $24AC clears its screens after two VBLs.
func (a *App) paintRestartPictures() {
	a.paintArenaResetPictures(2)
}

func (a *App) paintArenaResetPictures(speed int) {
	p := a.ambient.planes
	h := newPresentationClock(p)
	for _, bank := range p.screens() {
		h.restoreBottom(bank)
	}
	for id := 0; id < 2; id++ {
		h.hand(id, false)
	}
	for _, bank := range p.screens() {
		p.spriteAt(bank, 144, 10, 5, p.artwork.Sprites[p.artwork.SpeedLabel[0]], false, false, false)
		p.spriteAt(bank, 152, 10, 5, p.artwork.Sprites[p.artwork.SpeedLabel[1]], false, false, false)
	}
	p.paintSpeed(speed)
	for _, draw := range hudLayout(hudState{Round: 1, Seconds: 30, Speed: speed}) {
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

// constructNativeTitle enters $24AC with an already prepared arena. The
// ordinary score dispatcher performs its separate $623C before calling it.
func (a *App) constructNativeTitle() {
	p := a.ambient.planes
	seed := uint64(p.state.RandomCounter)<<48 | uint64(p.state.RandomMultiplier)<<32 | uint64(p.state.RandomAddition)<<16 | uint64(p.state.RandomXor)
	a.presentation.flow = nil
	a.presentation.title = newTitleSequenceFromPlanes(p, seed, p.state.Frame)
	a.presentation.title.paced = a.match.PALFrameClocks() == audio.Supplied313.FrameClocks()
	a.presentation.title.effects.hardware = a.titleColors
	a.view = titleView
}
