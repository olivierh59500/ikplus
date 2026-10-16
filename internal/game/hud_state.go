package game

// DisplayCounter is the byte used by the last completed $65A6/$6AD8 update.
// The fast ball path prepares its next swap before the next VBL; that pending
// increment is not yet the counter used to draw the displayed clock.
func (g *Game) DisplayCounter() byte { return g.displayCounter }

// DisplayClock retains the seconds and mode last painted by $6AD8. Starting
// a new round updates the logical countdown before its next header drawing.
func (g *Game) DisplayClock() (int, byte) { return g.displaySeconds, g.displayMode }

// DisplayRound is the round number last written to the original header.
func (g *Game) DisplayRound() int { return g.displayRound }

// DisplayVBL is the source byte sampled by the last $6AD8 header refresh.
func (g *Game) DisplayVBL() byte { return g.displayFrame }

func (g *Game) OwnerIndicators() [2]bool { return g.ownerIndicators }

func (g *Game) resetOwnerIndicators() {
	g.ownerFlash = [2]byte{30, 30}
	for id := 0; id < 2; id++ {
		if !g.Fighters[id].Human {
			g.ownerIndicators[id] = false
		}
	}
}

// $6C8C updates the two original hand indicators every fourth absolute VBL.
func (g *Game) tickOwnerIndicators() {
	g.ownerIndicatorPaints = 0
	if g.OriginalVBL()&3 != 0 {
		return
	}
	for id := 0; id < 2; id++ {
		if g.Fighters[id].Human && g.ownerFlash[id] != 0 {
			g.ownerIndicatorPaints++
			g.ownerFlash[id]--
			g.ownerIndicators[id] = g.ownerFlash[id]&4 == 0
		}
	}
}

func (g *Game) tickDisplayCounter() {
	switch g.Phase {
	case BallBonus:
		if !g.pacedBallPreparation() {
			g.completeDisplayWait()
		}
	case BombBonus:
		if g.Bonus.clearTicks > 0 {
			period := max(1, int(g.Bonus.clearingPeriod))
			if (g.Bonus.clearTicks-1)%period == 0 {
				g.completeDisplayWait()
			}
		} else if !g.pacedBombPreparation() && g.animationTick() {
			g.completeDisplayWait()
		}
	case BonusEnd:
		if !g.pacedBallPreparation() {
			g.tickBonusResultDisplayWait()
		}
	}
}

func (g *Game) completeDisplayWait() {
	g.visibleFighters, g.visibleValid = g.Fighters, true
	g.visibleAt = g.Frame
	g.displayCounter++
	g.displaySeconds, g.displayMode = max(0, g.Remaining/TickRate), g.OriginalMode()
	g.displayFrame = g.OriginalVBL()
	g.displayGermanText = g.germanText
	g.displayRound = g.Round
	g.displayElapsed = 0
	g.updateBeltPalette()
	g.paletteWaitFrame = g.Frame
}
