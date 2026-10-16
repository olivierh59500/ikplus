package game

// OriginalVBL is the mutable source byte at $608. The score/name screens mask
// that byte without resetting elapsed PAL time, prepared-picture timestamps
// or the independent audio clock.
func (g *Game) OriginalVBL() byte {
	if g.presentationState != nil {
		return g.presentationState.Frame
	}
	return byte(g.Frame) + g.sourceFrameOffset
}

// SetOriginalVBL applies a source counter write while retaining the monotonic
// native VBL count. Subsequent game ticks preserve its byte wrap and phase.
func (g *Game) SetOriginalVBL(value byte) {
	g.sourceFrameOffset = value - byte(g.Frame)
}
