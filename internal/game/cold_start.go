package game

// ColdStart prepares the first title and its initial arena poses.
// Three random words select fighter positions before the first publication.
func (g *Game) ColdStart(state *PresentationState) {
	holder := state
	if g.Frame != 0 || g.demonstrating {
		panic("cold startup requires an unstarted game")
	}
	g.presentationState = state
	g.sourceBootMarker = 0x01010202
	g.pausePoint = WaitArenaSetup
	g.speedValue, g.displaySpeedValue = 3, 3
	g.hintOutline = 0xffff
	g.graphicsPublished = holder.GraphicsColor
	g.groundPublished = holder.VisibleGroundShade
	if g.audioEngine != nil {
		g.audioEngine.ResetArena()
		g.audioEngine.MusicEnabled, g.audioEngine.EffectsEnabled = true, true
	}
	holder.ResetArena()
	g.random &= 0x0000ffffffffffff // Begin a new scene random counter.
	g.beginDemoArena(false)
	if g.audioEngine != nil {
		g.audioEngine.Restart()
	}
	g.setMasterVolume(15)
	g.CopyPresentationState(state)
	holder.Frame, holder.Mode = 0, 8
	holder.OwnerCursor, holder.ActorCursor, holder.LastActor = 255, 255, 255
	holder.Defence, holder.Activity = 1, 1
}
