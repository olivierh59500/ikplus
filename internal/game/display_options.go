package game

func (g *Game) GermanText() bool        { return g.germanText }
func (g *Game) DisplayGermanText() bool { return g.displayGermanText }
func (g *Game) RefereeVariant() byte    { return g.refereeVariant }
func (g *Game) AlternateShield() bool   { return g.alternateShield }
func (g *Game) GraphicsIndex() byte     { return g.graphicsIndex }
func (g *Game) GroundShade() uint16 {
	if g.groundShade == 0 {
		return 0x333
	}
	return g.groundShade
}
func (g *Game) PublishedGraphicsColor() uint16 {
	if g.graphicsPublished == 0 {
		return nativeGameplay.Defaults.Graphics
	}
	return g.graphicsPublished
}
func (g *Game) PublishedGroundShade() uint16 {
	if g.groundPublished == 0 {
		return nativeGameplay.Defaults.Ground
	}
	return g.groundPublished
}

func (g *Game) publishDisplayOptions() {
	// $EC4 runs before $667A. A key changes the next VBL's copper colors.
	g.graphicsPublished = fighterArtwork.GraphicsColors[g.graphicsIndex/2]
	g.groundPublished = g.GroundShade()
}

func (g *Game) readDisplayOptions(state *PresentationState) {
	holder := state
	g.graphicsIndex, g.refereeVariant = holder.Graphics, holder.Referee
	g.groundShade = holder.GroundShade
	g.germanText = holder.German != 0
	g.caption.state.Alternate = g.germanText
	g.alternateShield = holder.AlternateShield != 0
	g.graphicsPublished = holder.GraphicsColor
	g.groundPublished = holder.VisibleGroundShade
}

func (g *Game) applyDisplayKey(key byte, mode byte) bool {
	switch key {
	case 0x09, 0x66:
		g.graphicsIndex = (g.graphicsIndex + 2) & 6
		return true
	case 0x1f:
		if g.GroundShade() == 0x333 {
			g.groundShade = 0x444
		} else {
			g.groundShade = 0x333
		}
		return true
	case 0x24:
		if g.refereeVariant == 0 {
			g.refereeVariant = 12
		} else {
			g.refereeVariant = 0
		}
		return true
	case 0x30:
		if mode == 5 {
			g.alternateShield = !g.alternateShield
			return true
		}
	}
	if g.speedKeyHistory == 0xa2929332 {
		g.germanText = true
		g.caption.state.Alternate = true
		return true
	}
	if g.speedKeyHistory&0xffff == 0x9625 {
		g.germanText = false
		g.caption.state.Alternate = false
		return true
	}
	return false
}
