package game

// Both challenges restore two 83-row scene regions and clear four 200-row
// collision planes before sampling the final one-update hold.
const bonusSceneryClearWork = 2*83*356 + 4*200*58 + 880

// Result messages use the IRQ-owned $7D1 byte. The subsequent $773C frame
// holds instead retain a sampled $608 byte and count completed main waits.
// Keeping these clocks separate matters when FREZ suspends the main thread.
type bonusResultStage byte

const (
	bonusResultMessage bonusResultStage = iota
	bonusResultTail
	bonusResultFailureHold
	bonusResultHold
	bonusResultClearing
	bonusResultFinalFrame
	bonusResultScenery
)

func (b *BonusState) beginResultWait(frame byte) {
	b.resultIRQTimer = 0
	if b.Success {
		b.resultStage, b.resultIRQTimer = bonusResultMessage, 100
	} else if b.kind == 1 {
		b.startResultFrames(bonusResultFailureHold, 75, frame)
	} else {
		b.startResultFrames(bonusResultHold, 40, frame)
	}
}

func (b *BonusState) startResultFrames(stage bonusResultStage, count, frame byte) {
	b.resultStage, b.resultFrames, b.resultFrame = stage, count, frame
	if stage == bonusResultHold {
		// $B8AC/$C766 scans the absent white slot before the final hold,
		// even when red was the only surviving human. Prepared actor
		// pictures retain the preceding owner's pose in their own banks.
		b.Fighter = 0
	}
}

// $774A observes a byte change once, regardless of how many IRQs elapsed
// during a freeze. An exact wrap back to the observed byte completes nothing.
func (b *BonusState) advanceResultFrames(frame byte) bool {
	if frame == b.resultFrame {
		return false
	}
	b.resultFrame = frame
	if b.resultFrames != 0 {
		b.resultFrames--
	}
	return b.resultFrames == 0
}

func (g *Game) tickBonusResultDisplayWait() {
	b := &g.Bonus
	b.resultWaitComplete = false
	b.resultRetainPicture = false
	switch b.resultStage {
	case bonusResultMessage, bonusResultTail, bonusResultClearing:
		if b.kind == 0 || g.displayElapsed > g.waitThreshold() {
			g.completeDisplayWait()
			b.resultWaitComplete = true
			if b.kind == 1 {
				// Animation updates follow completed $65A6 waits rather than
				// the absolute host frame, which continues during FREZ.
				g.animationOrigin = g.Frame - uint64(g.animationTicks())
			}
		}
	}
}

func (g *Game) tickBonusResult(inputs [2]Input) {
	b := &g.Bonus
	if b.kind == 0 && b.resultStage != bonusResultFinalFrame {
		g.tickOriginalBallObjects(b.contactFighter)
	}
	g.updateSoundEffects()
	switch b.resultStage {
	case bonusResultMessage, bonusResultTail:
		if !b.resultWaitComplete {
			return
		}
		if b.resultIRQTimer == 0 {
			if b.kind == 0 && b.resultStage == bonusResultMessage {
				// $B834/$B83A begins a fresh fifty-IRQ wait after the
				// suspended first wait returns, even if it expired frozen.
				b.resultStage, b.resultIRQTimer = bonusResultTail, 50
				g.hideCaption()
				g.prepareBallResultActor(inputs)
				return
			}
			if b.kind == 1 {
				// $C87E dispatches before $C882 tests the timer. $C6EE
				// changes the request; the retained picture still has text.
				g.dispatchSound()
				g.caption.state.Message = 255
				b.resultRetainPicture = true
			}
			g.finishResultOwner(inputs)
			return
		}
		if b.kind == 1 {
			g.tickBombActor(inputs)
			g.tickCaption()
		} else {
			g.prepareBallResultActor(inputs)
			if b.resultStage == bonusResultMessage {
				g.tickCaption()
			}
		}
	case bonusResultFailureHold, bonusResultHold:
		if !b.advanceResultFrames(g.OriginalVBL()) {
			return
		}
		if b.resultStage == bonusResultFailureHold {
			g.finishResultOwner(inputs)
		} else {
			g.beginResultClearing(inputs)
		}
	case bonusResultClearing:
		if !b.resultWaitComplete {
			return
		}
		if b.kind == 1 {
			g.effectColor = 0x00e
			if g.bombPreparationEnabled && g.palFrameProfile.FrameClocks() == 227*313 {
				// Two scenery banks copy 83 rows each. Four collision planes
				// then clear 200 authored rows before the final frame hold.
				b.resultStage = bonusResultScenery
				b.bombPreparation.debt = bonusSceneryClearWork
				g.advanceBombScenery(259 * picturePreparationQuota / 313)
				return
			}
			g.speedValue = b.savedSpeed + 1
		}
		b.startResultFrames(bonusResultFinalFrame, 1, g.OriginalVBL())
	case bonusResultScenery:
		g.advanceBombScenery(g.bombMaintenanceWork())
	case bonusResultFinalFrame:
		if b.advanceResultFrames(g.OriginalVBL()) {
			if b.kind == 1 && g.bombPreparationEnabled && g.palFrameProfile.FrameClocks() == 227*313 {
				g.rememberBonusProjectiles()
				g.roundPreparation = roundPreparationBonusReturn
				g.Phase, g.PhaseTicks, g.Message = Ready, 0, "READY"
				return
			}
			g.continueAfterBonus(inputs)
		}
	}
}

func (g *Game) prepareBallResultActor(inputs [2]Input) {
	b := &g.Bonus
	b.contactFighter = g.Fighters[b.Fighter]
	g.tickShield(b.Fighter, g.bonusInput(inputs))
	g.syncShieldMotionFromPose()
}

func (g *Game) finishResultOwner(inputs [2]Input) {
	b := &g.Bonus
	if len(b.queue) > 1 {
		g.continueAfterBonus(inputs)
		return
	}
	b.startResultFrames(bonusResultHold, 40, g.OriginalVBL())
}

func (g *Game) beginResultClearing(inputs [2]Input) {
	b := &g.Bonus
	b.resultStage, b.resultIRQTimer, b.finalClearing = bonusResultClearing, 0, true
	g.hideCaption()
	b.ShieldVisible = false
	for id := range g.Fighters {
		f := &g.Fighters[id]
		f.SpriteFrame, f.X = 87, 272
		if b.kind == 1 {
			f.X, f.Facing = 240, -1
		}
		f.computerAnimation.Position, f.computerAnimation.Frame = byte(int(f.X/2)), byte(f.SpriteFrame)
		f.computerAnimation.Facing = 0
		if f.Facing < 0 {
			f.computerAnimation.Facing = 1
		}
	}
	if b.kind == 0 && b.Success {
		// $B8AC selects white even when absent; $B8FE still calls $C3EC
		// while preparing the two actor-free clearing pictures.
		g.tickShield(0, normalize(inputs[0]))
		g.syncShieldMotionFromActor(0)
	}
	// Accumulated $82E lets the first clearing wait return immediately.
	// The second prepared picture then waits for its own display swap.
	g.rememberPicture()
	g.completeDisplayWait()
	if b.kind == 0 && b.Success {
		// The immediate first return enters the second $B93E preparation
		// in this same VBL, including another live-input $C3EC call.
		g.tickShield(0, normalize(inputs[0]))
		g.syncShieldMotionFromActor(0)
	}
}
