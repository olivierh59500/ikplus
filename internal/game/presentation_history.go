package game

// retainedPresentationData keeps session-owned scores and title animation.
// Live match controllers are never copied into this retained state.
type retainedPresentationData struct{ state PresentationState }

func (g *Game) retainPresentationState(state *PresentationState) {
	if g.retainedPresentation == nil {
		g.retainedPresentation = new(retainedPresentationData)
	}
	g.retainedPresentation.state = *state
}
