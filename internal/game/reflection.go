package game

// ReflectionPhases returns the original live water phases. Bonus launches
// sample this same storage, so it belongs to the PAL simulation clock.
func (g *Game) ReflectionPhases() [21]byte {
	if g.reflectionReady {
		return g.reflectionPhases
	}
	return fighterArtwork.InitialWater
}

func (g *Game) resetReflection() {
	g.reflectionReady = false
	g.reflectionPhases = g.ReflectionPhases()
	g.reflectionReady = true
	g.reflectionUpdates, g.reflectionSlow = 0, 0
}

// $18C7E updates the 21 phases on even VBL bytes. $191A0 adds slow jitter
// whenever its byte counter wraps, before the ball interrupt samples them.
func (g *Game) tickReflection() {
	if !g.reflectionReady {
		g.resetReflection()
	}
	if g.OriginalVBL()&1 != 0 {
		return
	}
	g.reflectionUpdates++
	for row := range g.reflectionPhases {
		g.reflectionPhases[row] = (g.reflectionPhases[row] + 31) & 31
	}
	if g.reflectionUpdates != 0 {
		return
	}
	g.reflectionSlow++
	for row := range g.reflectionPhases {
		index := ((row & 3) + int(g.reflectionSlow>>4)) & 15
		g.reflectionPhases[row] = (g.reflectionPhases[row] + fighterArtwork.Water.Jitter[index]) & 31
	}
}
