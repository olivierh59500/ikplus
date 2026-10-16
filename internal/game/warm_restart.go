package game

// RestartRequested reports a request to replace the current scene with the title.
func (g *Game) RestartRequested() bool { return g.restartRequested }

// SetBootMarker supplies the loader-owned $64C word after native cold setup.
// Controlled routine fixtures may start before that setup with marker zero.
func (g *Game) SetBootMarker(marker uint32) {
	g.sourceBootMarker = marker
	if g.presentationState != nil {
		g.presentationState.BootMarker = marker
	}
}

// WarmRestart resets match controllers while preserving the running audio,
// random stream, scenery and high-score history.
func (g *Game) WarmRestart(state *PresentationState) {
	holder := state
	if g.presentationState != nil {
		g.DetachPresentation()
	}
	g.presentationState = state
	g.sourceFrameOffset = holder.Frame - byte(g.Frame)
	g.speedKeyDelay = holder.KeyDelay
	g.musicDisabled, g.effectsDisabled = holder.Music == 0, holder.Effects == 0
	if g.audioEngine != nil {
		g.audioEngine.MusicEnabled, g.audioEngine.EffectsEnabled = !g.musicDisabled, !g.effectsDisabled
	}
	g.pausePoint = WaitArenaSetup
	g.hint = HintState{}
	g.hintPosition, g.hintOffset, g.hintLine, g.hintGlyphCounter, g.hintColor = 0, 0, 0, 0, 0
	g.hintOutline = 0xffff
	g.waitPoint = 0
	g.speedValue, g.displaySpeedValue = 3, 3
	g.sourceFast = 0
	g.graphicsIndex, g.refereeVariant, g.groundShade = 0, 0, 0x333
	g.germanText, g.displayGermanText, g.filterLatch, g.ledFilterEnabled = false, false, false, false
	g.graphicsPublished = holder.GraphicsColor
	g.groundPublished = holder.VisibleGroundShade
	if g.audioEngine != nil {
		g.audioEngine.ResetArena()
	} else if g.audioControl != nil {
		g.audioControl.ClearArenaMemory()
	}
	holder.ResetArena()
	g.random &= 0x0000ffffffffffff
	g.Paused = false
	g.beginDemoArena(false)
	g.soundCooldown = [2]byte{}
	if g.audioEngine != nil {
		g.audioEngine.Restart()
	}
	g.musicDisabled = false
	g.setMasterVolume(15)
	// $19C4 resets $608 after the three $623C RNG calls, while the native
	// monotonic frame and completed audio clock remain continuous.
	g.sourceFrameOffset = -byte(g.Frame)
	holder.Frame = 0
	g.CopyPresentationState(state)
	// $623C leaves both descending owner/actor indices at -1 and publishes
	// its no-human strategy defaults before the title program takes ownership.
	holder.OwnerCursor, holder.ActorCursor, holder.LastActor = 255, 255, 255
	holder.Defence, holder.Activity = 1, 1
	holder.Mode = 8
	g.SetBootMarker(0x01010202)
	g.restartRequested = false
}
