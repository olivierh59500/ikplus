package presentation

import (
	"github.com/olivierh59500/ikplus/internal/audio"
	"github.com/olivierh59500/ikplus/internal/game"
)

// coldStartupRequested preserves explicit direct-game verification inputs;
// ordinary desktop/mobile launches and attract captures begin at the title.
func coldStartupRequested(options Options, replay *inputReplay) bool {
	if replay != nil {
		return replay.ColdStart
	}
	return !options.Autoplay && options.StartBonus == 0 && !options.ScoreMenu &&
		(options.SmokeTicks == 0 || options.Attract)
}

// coldStart translates the first $17EA entry after native asset conversion.
// It does not run the arena demo before $24AC or reuse a warm reset's RNG.
func (a *App) coldStart() {
	p := a.ambient.planes
	p.state.BootMarker = 0x01010202
	// $18958/$18A1E installs the water callback and sparkle arrays. These
	// live bytes also remain observable through the original indexed RNG.
	copy(p.state.ReflectionPhases[0:21], p.artwork.InitialWater[:])
	for i := range p.state.ReflectionLeft {
		p.state.ReflectionLeft[i] = 31
	}
	p.glitter = [8]uint16{0, 0, 0, 0, 0, 0, 32, 32}
	clear(p.state.Humans[0:3])
	p.state.Speed, p.state.Music, p.state.Effects = 2, 1, 1
	p.state.Graphics, p.state.Referee = 0, 0
	p.state.GroundShade, p.state.HintCount, p.state.TitleLanguage, p.state.German, p.state.FontOutline = 0x333, 0, 0, 0, 0xffff
	p.state.Displayed = game.PictureFirst
	p.state.FreezeLocation = game.WaitArenaSetup
	a.match.ColdStart(&p.state)
	a.paintArenaResetPictures(2)
	p.state.Frame, p.state.Elapsed, p.state.BufferPhase, p.state.Mode = 0, 0, 4, 8
	p.state.Filter, p.state.TitleLanguage, p.state.StartupReserved = 0, 0, 0
	p.state.CollisionEnabled, p.state.SecondaryCollisionEnabled = 0, 0
	p.state.Background, p.state.Drawing, p.state.Displayed = game.PictureFirst, game.PictureFirst, game.PictureFirst
	p.state.EraseStart, p.state.EraseCursor = game.RestorationCursor{}, game.RestorationCursor{}
	p.state.FreezeLocation = game.WaitArenaSetup
	p.state.CollisionRead, p.state.CollisionReadMirror, p.state.CollisionWrite, p.state.CollisionWriteMirror = game.CoverageSecond, game.CoverageSecondMirrored, game.CoverageFirst, game.CoverageFirstMirrored
	for index := range a.titleColors {
		a.titleColors[index] = p.artwork.Title.Black[index]
	}
	a.presentation = &presentationCycle{rgba: make([]byte, 320*200*4)}
	a.attachPresentation()
	if a.match.PALFrameClocks() == audio.Supplied313.FrameClocks() {
		a.initializeColdPresentation()
	}
	a.constructNativeTitle()
	if a.match.PALFrameClocks() == audio.Controlled312.FrameClocks() {
		a.presentation.title.effects.paletteMode = paletteArena
	} else {
		a.presentation.title.effects.paletteMode = paletteTitle
	}
	a.view, a.returnView = titleView, titleView
	a.ambient.AmbientState = presentationAmbient(p)
	a.ambient.lastFrame, a.ambient.videoBank = a.match.Frame, 0
}

// initializeColdPresentation publishes the first observed title state before
// its animation sequence starts. Music, counters and palette advance together
// as an ordinary native update; there is no CPU startup context to simulate.
func (a *App) initializeColdPresentation() {
	p := a.ambient.planes
	p.state.Frame++
	if p.state.Frame == 0 {
		p.state.FrameWraps++
	}
	if p.state.MessageTimer != 0 {
		p.state.MessageTimer--
	}
	a.match.BeginColdPresentationUpdate(func() {
		p.publishDisplay()
		a.match.SetSourceJoystick([2]game.Input{})
	})
	effects := titleEffects{planes: p, random: a.match.PresentationRandomWord, paletteMode: paletteTitle}
	effects.tick(p.state.Frame)
	a.titleColors = effects.hardware
	a.match.EndColdPresentationUpdate()
}
