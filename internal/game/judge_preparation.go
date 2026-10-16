package game

// Judge preparation draws the actors before the speech. The ordinary work
// queue keeps random sampling behind that artwork instead of running it when
// the preceding wait returns. No processor or interrupt handler runs here.
type judgePreparationWork struct {
	stage         byte
	debt          int
	copyRemainder int
	slotDebt      int
	sparkles      captionSparkleWork
	trace         CaptionPreparationSnapshot
}

const (
	judgePrepareRestore byte = iota
	judgePrepareActors
	judgePrepareText
	judgePrepareReferee
	judgePrepareSparkles
	judgePrepareShadows
	judgePrepareDone
)

func (g *Game) stagedJudgeCaption() bool {
	return g.palFrameProfile.FrameClocks() == 227*313 &&
		g.judge.part != 0 && g.caption.state.Visible && g.caption.budget != 0
}

func (g *Game) beginJudgePreparation() {
	_, bank := g.SceneryBuffers()
	p := g.Picture(bank)
	base := 1046
	if p.restoreRuns != 0 {
		base += 142
	}
	g.judge.preparation = &judgePreparationWork{stage: judgePrepareRestore,
		debt:  base + p.restoreRows*79 + p.restoreRuns*181,
		trace: CaptionPreparationSnapshot{Frame: g.Frame, Mode: g.OriginalMode(), Counter: g.displayCounter, Message: g.caption.state.Message, Before: g.Caption()}}
}

func (g *Game) judgeMaintenanceWork() int {
	work := 8000
	if g.OriginalVBL()&3 == 0 {
		work += 1500 + 6900*g.ownerIndicatorPaints
	} else if g.OriginalVBL()&3 == 2 {
		work += 3300
	}
	return work
}

func (g *Game) advanceJudgePreparation(position int) bool {
	w := g.judge.preparation
	for position < picturePreparationQuota {
		if w.debt != 0 {
			for w.debt > 0 && position < picturePreparationQuota {
				drawingRate := 0
				if w.stage == judgePrepareRestore || w.stage == judgePrepareReferee {
					drawingRate = 1070
					if w.stage == judgePrepareReferee {
						drawingRate = 1055
					}
				}
				limit, rate := preparationWorkRate(position, drawingRate)
				position = advancePreparationDebt(&w.debt, &w.copyRemainder, position, limit, rate)
			}
			if w.debt != 0 {
				return false
			}
		}
		switch w.stage {
		case judgePrepareRestore:
			g.recordJudgeWorkStage(w, position)
			w.stage = judgePrepareActors
			for _, fighter := range g.Fighters {
				work := actorPreparationWork(fighter.SpriteFrame, int(fighter.X/2))
				if fighter.SpriteFrame >= 36 && fighter.SpriteFrame <= 47 && fighter.Stun == 0 && fighter.recoveryWait == 0 && !fighter.Puff {
					// Empty effect routines return after their timer checks;
					// they do not perform the star, drool or puff image work.
					work -= 1250
				}
				w.debt += work
			}
		case judgePrepareActors:
			g.recordJudgeWorkStage(w, position)
			w.stage = judgePrepareText
			w.debt = captionTextWorkWithRows(g.Caption(), 81)
			if g.caption.state.Previous == g.caption.state.Message && g.caption.budget == 40 {
				w.debt += 23400
			}
		case judgePrepareText:
			g.recordJudgeWorkStage(w, position)
			g.caption.beginUpdate(g.caption.state.Message, g.captionContext())
			w.stage = judgePrepareReferee
			w.debt = actorPreparationWork(int(g.caption.state.RefereeFrame), int(g.caption.state.RefereeX))
		case judgePrepareReferee:
			g.recordJudgeWorkStage(w, position)
			w.stage = judgePrepareSparkles
			w.sparkles = g.caption.prepareSparkles()
		case judgePrepareSparkles:
			done := w.sparkles.advancePainting(&g.caption, func() (byte, bool) {
				cost := 200
				if w.sparkles.phase == 2 {
					cost = 275
				}
				if position+cost > picturePreparationQuota {
					return 0, false
				}
				position += cost
				before := g.random
				state := g.Caption()
				value := g.originalRandomByte()
				if g.captionTraceEnabled {
					w.trace.Random = append(w.trace.Random, CaptionRandomRequest{g.Frame, g.OriginalVBL(), before, g.random, value, state})
				}
				return value, true
			}, func(slot int) bool {
				if w.slotDebt == 0 {
					x := g.caption.state.Anchor<<2 + 14 + g.caption.x[slot]
					w.slotDebt = 1440 + int(x&7)*35
				}
				used := min(w.slotDebt, picturePreparationQuota-position)
				position += used
				w.slotDebt -= used
				return w.slotDebt == 0
			})
			if !done {
				return false
			}
			g.recordJudgeWorkStage(w, position)
			w.stage, w.debt = judgePrepareShadows, 5500
		case judgePrepareShadows:
			for id := range g.Fighters {
				g.advanceFighterEffects(id)
			}
			g.rememberPicture()
			if g.captionTraceEnabled {
				w.trace.After = g.Caption()
				w.trace.CompletedFrame = g.Frame
				g.captionPreparations = append(g.captionPreparations, w.trace)
			}
			w.stage = judgePrepareDone
		case judgePrepareDone:
			return true
		}
	}
	return w.stage == judgePrepareDone
}

func (g *Game) recordJudgeWorkStage(w *judgePreparationWork, position int) {
	if g.captionTraceEnabled {
		w.trace.Stages = append(w.trace.Stages, CaptionWorkStage{w.stage, g.Frame, position})
	}
}
