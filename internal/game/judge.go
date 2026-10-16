package game

// judgeState translates $5D12's main-thread waits. Its timer decrements on
// every VBL, while a picture finishes only when $65A6's elapsed byte exceeds4.
type judgeState struct {
	preparation                            *judgePreparationWork
	active, complete, waiting, resultReady bool
	part                                   int
	mode, timer, level, subsecond          byte
	order                                  [3]byte
	selection, announcement, humanCount    byte
	allInactive, awardPrelude, ending      bool
	soundSuppressed                        bool
	continuationTimerPending               bool
}

// OriginalMode exposes the source dispatcher byte for HUD/caption decisions.
func (g *Game) OriginalMode() byte {
	if g.presentationState != nil {
		return g.presentationState.Mode
	}
	switch g.Phase {
	case Judge:
		if g.judge.active {
			return g.judge.mode
		}
		return 3
	case PauseDemo:
		return 6
	case BallBonus:
		return 5
	case BombBonus:
		return 7
	case BonusEnd:
		if g.Bonus.kind == 1 {
			return 7
		}
		return 5
	case GameOver:
		if g.judge.active {
			return g.judge.mode
		}
		return 4
	default:
		return 1
	}
}

func (g *Game) beginJudge() {
	g.rememberFightBoundaryMode(3)
	g.Remaining = max(0, g.Remaining/TickRate) * TickRate
	level := g.sourceDifficulty()
	_, subsecond, _ := originalRoundClock(g.clockTicks)
	g.judge = judgeState{active: true, mode: 3, timer: 100, level: level, subsecond: byte(subsecond), order: [3]byte{0, 1, 2}}
	j := &g.judge
	if g.audioControl != nil {
		// $5E9A latches $958 when the retained delayed sample is six.
		j.soundSuppressed = g.audioControl.State.Delayed == 6
	}
	better := func(a, b byte) bool {
		x, y := g.Fighters[a], g.Fighters[b]
		return x.Points > y.Points || x.Points == y.Points && x.RoundScore > y.RoundScore
	}
	equal := func(a, b byte) bool {
		x, y := g.Fighters[a], g.Fighters[b]
		return x.Points == y.Points && x.RoundScore == y.RoundScore
	}
	for _, pair := range [3][2]int{{0, 1}, {1, 2}, {0, 1}} {
		if better(j.order[pair[1]], j.order[pair[0]]) {
			j.order[pair[0]], j.order[pair[1]] = j.order[pair[1]], j.order[pair[0]]
		}
	}
	mask := byte(0)
	for _, id := range j.order {
		mask <<= 1
		if g.Fighters[id].Human {
			mask |= 1
			j.humanCount++
		}
	}
	switch {
	case equal(j.order[0], j.order[1]):
		j.selection = 14
		if equal(j.order[0], j.order[2]) {
			j.selection = 21
		}
	case equal(j.order[1], j.order[2]):
		j.selection = 7
	}
	j.selection += mask
	j.announcement = nativeGameplay.Judge.Choices[j.selection] >> 4
	if mask == 0 {
		j.announcement = nativeGameplay.Judge.Idle[g.originalRandomByte()&31]
	}
	for id := range g.Fighters {
		f := &g.Fighters[id]
		f.Rank = 1
		for other := range g.Fighters {
			if better(byte(other), byte(id)) {
				f.Rank++
			}
		}
		if !f.computerAnimationReady || int(f.computerAnimation.Action) != f.Action {
			g.resetSourceFighter(id)
		}
		f.computerAnimation.Fresh, f.computerAnimation.Pending, f.computerAnimation.Idle = 0, 0, 0
		if f.Stun != 0 {
			f.Stun = int(byte(f.Stun) & 63)
		}
		if f.recoveryWait != 0 {
			f.recoveryWait = int(byte(f.recoveryWait) & 63)
		}
		if f.GetUp != 0 {
			f.GetUp = 120
		}
	}
	g.caption.state.Order = j.order
	g.Phase, g.PhaseTicks, g.Message = Judge, 0, "ROUND OVER"
	g.pumpJudge()
}

func (g *Game) tickJudge() {
	j := &g.judge
	if !j.active || j.complete {
		return
	}
	if j.continuationTimerPending {
		j.continuationTimerPending = false
		j.timer = 20
	}
	if j.mode == 1 && g.Fighters[0].Action != 28 {
		j.subsecond--
		if j.subsecond == 0 && g.Remaining != 0 {
			g.Remaining -= TickRate
			j.subsecond = 50
		}
	}
	g.pumpJudge()
}

func (g *Game) pumpJudge() {
	j := &g.judge
	position := g.judgeMaintenanceWork()
	for !j.complete {
		if !j.waiting {
			g.beginPicture()
			if g.stagedJudgeCaption() {
				g.beginJudgePreparation()
				j.waiting = true
				position = 259 * picturePreparationQuota / 313
			} else {
				if j.part == 0 {
					g.animateJudgeActors()
					if !j.soundSuppressed {
						g.dispatchSound()
					}
				} else {
					g.tickCaption()
				}
				if j.part == 0 && g.palFrameProfile.FrameClocks() == 227*313 {
					g.prepareTimedFighterEffects()
				} else {
					for id := range g.Fighters {
						g.advanceFighterEffects(id)
					}
				}
				g.rememberPicture()
				j.waiting = true
			}
		}
		if j.preparation != nil && !g.advanceJudgePreparation(position) {
			return
		}
		if g.displayElapsed <= g.waitThreshold() {
			return
		}
		g.completeDisplayWait()
		j.preparation = nil
		j.waiting = false
		g.finishJudgePicture()
	}
	g.afterJudge()
}

// $777E runs during the first100 VBLs. Mode3 clears human pending commands
// and disables computer selection; existing animations can still finish.
func (g *Game) animateJudgeActors() {
	j := &g.judge
	for id := range g.Fighters {
		if g.Fighters[id].Human {
			g.Fighters[id].computerAnimation.Pending = 0
		}
	}
	for id := range g.Fighters {
		f := &g.Fighters[id]
		s := &f.computerAnimation
		s.Position = byte(int(f.X / 2))
		s.Facing = 0
		if f.Facing < 0 {
			s.Facing = 1
		}
		round := byte(g.Round/10*16 + g.Round%10)
		recovered := false
		if f.Human {
			recovered = s.advanceHumanWithKey(3, round, g.selectorKey(), byte(f.recoveryWait), byte(f.GetUp), g.originalRandomByte, func(flags byte) { g.publishFighterEffects(id, flags) })
		} else {
			recovered = s.advanceWithKey(3, j.level, round, g.selectorKey(), byte(f.recoveryWait), byte(f.GetUp), g.originalRandomByte, func(flags byte) { g.publishFighterEffects(id, flags) })
		}
		if recovered {
			g.soundDown[id] = 0
		}
		f.Action, f.SpriteFrame = int(s.Action), int(s.Frame)
		f.AnimationStep = int(s.Current) - int(nativeGameplay.Animation.Starts[s.Action])
		f.Move = moveForSourceAction(int(s.Action))
		f.X, f.Y, f.Height = float64(s.Position)*2, GroundY, 0
		f.Facing = 1
		if s.Facing != 0 {
			f.Facing = -1
		}
		f.selected, f.aiWait = moveForSourceAction(int(s.Pending)), int(s.Idle)
	}
}

func (g *Game) finishJudgePicture() {
	j := &g.judge
	switch j.part {
	case 0:
		if j.timer == 0 {
			j.part = 1
			j.timer = nativeGameplay.Judge.Timers[j.level]
			g.startCaption(j.announcement)
		}
	case 1:
		if j.timer == 0 {
			g.judgeAfterAnnouncement()
		}
	case 2:
		if j.timer == 0 {
			if j.allInactive {
				j.mode = 8
				j.ending, j.resultReady, j.complete = true, true, true
			} else {
				g.judgeTimeOrHold()
			}
		}
	case 3:
		if j.awardPrelude {
			j.awardPrelude = false
			j.timer = 50
		} else if j.timer == 0 {
			g.judgeAwardOneSecond()
		}
	case 4:
		g.judgeAwardOneSecond()
	case 5:
		if j.timer == 0 {
			g.judgeChooseContinuation()
		}
	case 6, 8:
		if j.timer == 0 {
			j.complete = true
		}
	case 7:
		if j.timer == 0 {
			j.part, j.timer = 8, 10
			g.hint.Timer, g.hint.Index = 15, 0
		}
	}
}

func sourceScoreWord(score int) uint16 {
	n := score / 100 % 10000
	return uint16(n/1000)<<12 | uint16(n/100%10)<<8 | uint16(n/10%10)<<4 | uint16(n%10)
}

func (g *Game) judgeAfterAnnouncement() {
	j := &g.judge
	if j.humanCount == 0 || j.selection == 1 {
		j.part, j.timer = 5, 50
		return
	}
	if int8(byte(g.Round/10*16+g.Round%10)) >= 2 && (j.announcement == 6 || j.announcement == 9) {
		if g.Fighters[0].Human && g.Fighters[1].Human && !g.Fighters[0].roundActive && !g.Fighters[1].roundActive {
			for id := 0; id < 2; id++ {
				g.Fighters[id].Human, g.Fighters[id].Eliminated = false, true
				g.ownerIndicators[id] = false
			}
			j.allInactive, j.humanCount, j.part, j.timer = true, 0, 2, 200
			g.startCaption(35)
			return
		}
		for id := 0; id < 2; id++ {
			f := &g.Fighters[id]
			// The original CMP.W is signed, including packed BCD scores whose
			// high bit is set. Preserve the high-score inactivity edge case.
			if f.Human && !f.roundActive && int16(sourceScoreWord(f.Score)) < 0x50 {
				f.Human, f.Eliminated = false, true
				g.ownerIndicators[id] = false
				j.part, j.timer = 2, 200
				g.startCaption(byte(36 + id))
				return
			}
		}
	}
	g.judgeTimeOrHold()
}

func (g *Game) judgeTimeOrHold() {
	j := &g.judge
	if g.Remaining == 0 {
		j.part, j.timer = 5, 50
		return
	}
	seconds := g.Remaining / TickRate
	g.caption.state.Amount = byte(seconds/10*16 + seconds%10)
	j.part, j.timer, j.awardPrelude = 3, 0, true
	g.startCaption(16)
}

func (g *Game) judgeAwardOneSecond() {
	j := &g.judge
	if g.Remaining == 0 {
		j.part, j.timer = 5, 50
		return
	}
	f := &g.Fighters[j.order[0]]
	f.addOriginalScore(100)
	g.Remaining -= TickRate
	// $6082 paints the new seconds immediately after each awarded point.
	g.displaySeconds = max(0, g.Remaining/TickRate)
	// The award precedes this picture's $A938 palette update in the source.
	g.updateBeltPalette()
	j.part, j.timer = 4, 0
}

func (g *Game) judgeChooseContinuation() {
	j := &g.judge
	mask := nativeGameplay.Judge.Choices[j.selection] & 7
	j.resultReady = true
	if mask == 0 && j.humanCount != 0 {
		g.candidateOwner = j.order[2]
		j.part, j.timer, j.mode, j.ending = 7, 200, 4, true
		for id := range g.Fighters {
			if g.Fighters[id].Human {
				g.Fighters[id].Eliminated = true
			}
		}
		g.startCaption(17)
		return
	}
	j.mode = 1
	g.candidateOwner = 0x80
	if mask == 0 {
		j.mode = 4
	}
	for rank := 2; rank >= 0; rank-- {
		id := j.order[rank]
		f := &g.Fighters[id]
		if mask&1 == 0 && f.Human {
			f.Human, f.Eliminated = false, true
		}
		mask >>= 1
	}
	g.Fighters[2].Human = false
	j.ending = j.mode == 4 || !g.Fighters[0].Human && !g.Fighters[1].Human
	j.part, j.timer = 6, 20
	if g.palFrameProfile.FrameClocks() == 227*313 {
		// Original caption clearing completes before the final hold starts.
		j.timer, j.continuationTimerPending = 0, true
	}
	g.hint.Timer, g.hint.Index = 15, 0
	g.hideCaption()
}
