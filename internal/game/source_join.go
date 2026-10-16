package game

// SourceJoin starts a human tournament through the native preparation phases.
// The existing audio, random stream and retained scenes keep their owners.
func (g *Game) SourceJoin(state *PresentationState, players int) {
	owners := g.BeginSourceJoin(state)
	g.PrepareSourceJoin(state, players, owners)
	g.FinishSourceJoin(state, players)
}

// BeginSourceJoin clears the arena before its later preparation phases. The
// running music, frame clock and retained scenery stay with the same match.
func (g *Game) BeginSourceJoin(state *PresentationState) [2]bool {
	holder := state
	if g.presentationState != nil {
		g.DetachPresentation()
	}
	g.presentationState = state
	g.readHintState(state)
	g.sourceFrameOffset = -byte(g.Frame)
	g.sourceBootMarker = holder.BootMarker
	g.pausePoint = holder.FreezeLocation
	g.waitPoint = 0
	g.musicDisabled, g.effectsDisabled = holder.Music == 0, holder.Effects == 0
	g.setMasterVolume(holder.Volume)
	fast := holder.Fast
	g.sourceFast = fast
	if g.audioEngine != nil {
		g.audioEngine.MusicEnabled, g.audioEngine.EffectsEnabled = !g.musicDisabled, !g.effectsDisabled
		g.audioEngine.Fast, g.audioEngine.Substep = fast != 0, 0
	}
	g.graphicsPublished = holder.GraphicsColor
	g.groundPublished = holder.VisibleGroundShade
	var previousOwners [2]bool
	if holder.Mode != 8 {
		for id := range previousOwners {
			// Presentation can blink these hands while arena logic is
			// suspended. The current $6C8C counter retains its painted bit.
			previousOwners[id] = holder.OwnerFlash[id]&4 == 0
		}
	}
	g.irqPictureBank = 0
	if holder.Displayed == PictureSecond {
		g.irqPictureBank = 1
	}
	holder.Frame, holder.KeyDelay, holder.Speed, holder.Mode = 0, 0, 2, 0
	g.speedKeyDelay, g.speedValue, g.displaySpeedValue = 0, 3, 3
	if fast != 0 {
		g.displaySpeedValue = 2
	}
	g.resetArenaAudio()
	holder.ResetArena()
	g.random &= 0x0000ffffffffffff
	g.publishPresentationRandom()
	g.sourceJoinPlayers = 0
	return previousOwners
}

// PrepareSourceJoin samples fighter jitter on the preparation update. Retain
// randomness consumed by scenery after the earlier clear, and its counters.
func (g *Game) PrepareSourceJoin(state *PresentationState, players int, previousOwners [2]bool) {
	holder := state
	publishedBelts := [3]uint16{holder.VisibleWhiteBelt, holder.VisibleRedBelt, holder.VisibleBlueBelt}
	publishedEffect := holder.VisibleEffectColor
	video := holder.Displayed
	repeat, elapsed, wraps := holder.InputRepeat, holder.Elapsed, holder.FrameWraps
	// Scenery has continued during preparation; preserve its live water phase.
	g.DetachPresentation()
	g.presentationState = state
	g.startTournament(players, g.random, g.Frame, true)
	g.inputRepeatCounter, g.displayElapsed, g.sourceVBLWraps = repeat, elapsed, wraps
	for id := 0; id < players; id++ {
		// $623C redraws only absent owners. A present human retains the
		// hand already painted until the next fourth-VBL blink update.
		g.ownerIndicators[id] = previousOwners[id]
	}
	// EC4 published these colors before the join command. The new belt and
	// effect colors become visible at the next VBL, as in ordinary arena work.
	g.beltColorsVisible, g.effectColorVisible = publishedBelts, publishedEffect
	g.CopyPresentationState(state)
	holder.Displayed = video
	holder.Mode = 0
	holder.HumanMask = byte((1 << players) - 1)
	holder.OwnerCursor, holder.ActorCursor, holder.LastActor = 255, 255, 255
	// The initial arena controllers run only after ready publication.
}

// FinishSourceJoin publishes ready after preparation, before the next pose is
// calculated. The atomic reference adapter also calls this without a delay.
func (g *Game) FinishSourceJoin(state *PresentationState, players int) {
	holder := state
	holder.Mode = 1
	copy(g.ownerFlash[:], holder.OwnerFlash[0:2])
	if holder.Frame != 0 && holder.Frame&3 == 0 {
		for id := 0; id < players; id++ {
			g.ownerIndicators[id] = g.ownerFlash[id]&4 == 0
		}
	}
	g.irqPictureBank = 0
	if holder.Displayed == PictureSecond {
		g.irqPictureBank = 1
	}
	g.DetachPresentation()
	g.completeDisplayWait()
	g.rememberFightBoundary()
	g.CopyPresentationState(state)
	holder.Mode = 1
	holder.HumanMask = byte((1 << players) - 1)
	holder.BlackScreen = 0
	holder.OwnerCursor, holder.ActorCursor, holder.LastActor = 255, 255, 255
	g.sourceJoinPlayers = 0
	g.retainPresentationState(state)
	g.presentationState, g.presentationSpeedChanged = nil, nil
	g.requestSound(15, false)
}

// PrepareJoinedArena continues B38/1E4A to the new tournament's first wait.
// It consumes no VBL and prevents the next Tick from preparing that picture twice.
func (g *Game) PrepareJoinedArena() {
	g.prepareFightPicture(g.joystick)
}
