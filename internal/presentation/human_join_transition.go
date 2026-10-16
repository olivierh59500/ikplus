package presentation

import "github.com/olivierh59500/ikplus/internal/game"

import "github.com/olivierh59500/ikplus/internal/pilot"

// humanJoinTransition retains the scene while a human start restores the arena,
// samples fighter positions and publishes ready. These are ordinary Go phases.
type humanJoinTransition struct {
	players, elapsed, restoreUpdates, readyUpdate int
	owners                                        [2]bool
	picture                                       titleSequence
	clock                                         *highScoreSequence
}

func (a *App) beginHumanJoin(players int, mode byte) {
	p := a.ambient.planes
	j := &humanJoinTransition{players: players, readyUpdate: 4, picture: titleSequence{effects: titleEffects{planes: p, paletteMode: paletteArena}}}
	if players == 2 && a.match.Ambient().Mode != 4 {
		// The shorter two-player setup fits the earlier publication only
		// while the fish handler is not using the preparation budget.
		j.readyUpdate = 3
	}
	j.picture.effects.hardware = a.titleColors
	if mode == 8 {
		j.restoreUpdates, j.readyUpdate = 4, 7
		p.state.BlackScreen = 1
		j.picture.effects.paletteMode = paletteTitle
		for i := range j.picture.effects.hardware {
			j.picture.effects.hardware[i] = p.artwork.Title.Black[i]
		}
	}
	p.state.Humans[0], p.state.Humans[1], p.state.JoinPending = 1, byte(players-1), 1
	a.CancelInput()
	a.players, a.view, a.returnView, a.paused, a.scoreRecorded = players, matchView, matchView, false, false
	a.eliminatedScores = [2]*score{}
	a.scrollKey, a.ambient.history = 0, 0
	a.pilot = pilot.New(0)
	a.presentation = &presentationCycle{rgba: make([]byte, 320*200*4), humanJoin: j}
	if j.restoreUpdates == 0 {
		j.owners = a.match.BeginSourceJoin(&p.state)
	}
	a.attachPresentation()
	j.clock = newPresentationClock(p)
	a.match.EndPresentationUpdate(true)
}

func (a *App) advanceHumanJoin() {
	c := a.presentation
	j := c.humanJoin
	p := a.ambient.planes
	if p.state.BlackScreen != 0 {
		p.state.Frame++
		if p.state.Frame == 0 {
			p.state.FrameWraps++
		}
		p.publishDisplay()
		a.match.BeginPresentationUpdate()
	} else {
		j.clock.vbl()
		j.picture.effects.paletteMode = paletteArena
		if a.match.RestartRequested() {
			a.warmRestart()
			return
		}
		if players := a.match.JoinRequested(); players != 0 {
			a.sourceJoin(players)
			return
		}
		if p.state.Frozen != 0 {
			a.match.EndPresentationUpdate(true)
			return
		}
	}
	j.elapsed++
	if j.restoreUpdates != 0 && j.elapsed == j.restoreUpdates {
		for _, bank := range p.screens() {
			copy(bank.pixels[:32000], p.artwork.Background[:32000])
		}
		j.owners = a.match.BeginSourceJoin(&p.state)
		a.attachPresentation()
		j.clock = newPresentationClock(p)
	}
	phase := j.elapsed - j.restoreUpdates
	if phase == 1 {
		a.paintArenaResetPictures(a.match.DisplaySpeed())
	}
	if phase == 2 {
		a.match.PrepareSourceJoin(&p.state, j.players, j.owners)
		a.paintArenaResetHeader(hudState{Round: 1, Seconds: 30, Mode: 1, Speed: a.match.DisplaySpeed(), Humans: (1 << j.players) - 1, German: p.state.German != 0})
	}
	if phase == 3 {
		p.state.Mode = 1
	}
	if j.elapsed == j.readyUpdate {
		a.match.FinishSourceJoin(&p.state, j.players)
		a.ambient.AmbientState = a.match.Ambient()
		a.ambient.lastFrame = a.match.Frame
		a.ambient.videoBank = 0
		if p.state.Displayed == game.PictureSecond {
			a.ambient.videoBank = 1
		}
		a.handoffPixels = append(a.handoffPixels[:0], j.picture.pixels()...)
		a.presentation, p.presentation = nil, nil
		a.match.PrepareJoinedArena()
	}
	a.match.EndPresentationUpdate(true)
}
