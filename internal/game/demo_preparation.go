package game

// BeginDemoPreparation stops arena effects and clears the original RNG counter.
// Fighter jitter is a later phase so it samples the proper presentation update.
func (g *Game) BeginDemoPreparation() {
	g.resetArenaAudio()
	g.random &= 0x0000ffffffffffff
	g.publishPresentationRandom()
}

// PrepareDemoFighters builds the ready poses without publishing a display swap.
func (g *Game) PrepareDemoFighters() {
	g.Paused = false
	g.beginDemoArena(false)
	if holder := g.presentationState; holder != nil {
		g.CopyPresentationState(holder)
		holder.Mode = 0
		holder.OwnerCursor, holder.ActorCursor, holder.LastActor = 255, 255, 255
		holder.Defence, holder.Activity, holder.TargetPolicy = 1, 1, 0
	}
}

// FinishDemoPreparation publishes the first ready picture after setup.
func (g *Game) FinishDemoPreparation() {
	g.irqPictureBank, _ = g.SceneryBuffers()
	g.completeDisplayWait()
	g.rememberFightBoundary()
}
