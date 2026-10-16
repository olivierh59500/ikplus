package game

// Bomb preparation retains drawing work before sampling the next command.
// It is an ordinary game task queue, independent of host rendering speed.
type bombPreparationWork struct {
	stage, timer   byte
	debt, slotDebt int
	copyRemainder  int
	sparkles       captionSparkleWork
	trace          CaptionPreparationSnapshot
}

const (
	bombPrepareRestore byte = iota
	bombPrepareText
	bombPrepareReferee
	bombPrepareSparkles
	bombPrepareControl
	bombPrepareActor
	bombPrepareDetails
	bombPrepareWait
	bombPrepareHeader
)

// BombPreparationSnapshot is development evidence only. Neither gameplay nor
// the pilot reads the preparation journal.
type BombPreparationSnapshot struct {
	Frame          uint64
	Position       int
	Stage, Counter byte
	Random         uint64
	Actor          TitleActorState
	Caption        CaptionState
	Bonus          BonusSnapshot
}

// UseNativeBombPreparation retains drawing work before the next actor command.
func (g *Game) UseNativeBombPreparation()                   { g.bombPreparationEnabled = true }
func (g *Game) BombPreparations() []BombPreparationSnapshot { return g.bombPreparations }

func (g *Game) pacedBombPreparation() bool {
	return g.bombPreparationEnabled && g.palFrameProfile.FrameClocks() == 227*313 && g.Phase == BombBonus
}

func (g *Game) beginBombRestore() {
	_, bank := g.SceneryBuffers()
	p := g.Picture(bank)
	w := &g.Bonus.bombPreparation
	if g.captionTraceEnabled {
		w.trace = CaptionPreparationSnapshot{Frame: g.Frame, Mode: g.OriginalMode(),
			Counter: g.displayCounter, Message: g.caption.state.Message, Before: g.Caption()}
	}
	w.stage = bombPrepareRestore
	w.debt = 1046 + p.restoreRows*100 + p.restoreRows/2 + p.restoreRuns*181
	if p.restoreRuns != 0 {
		w.debt += 142
	}
}

func (g *Game) bombMaintenanceWork() int {
	work := 9200
	if g.OriginalVBL()&1 == 0 {
		work += 1000
	}
	if g.OriginalVBL()&3 == 0 {
		work += 500 + 6900*g.ownerIndicatorPaints
	}
	return work
}

func (g *Game) advanceBombScenery(position int) {
	b := &g.Bonus
	used := min(b.bombPreparation.debt, picturePreparationQuota-position)
	b.bombPreparation.debt -= used
	if b.bombPreparation.debt != 0 {
		return
	}
	g.clearArenaPictures()
	g.speedValue = b.savedSpeed + 1
	b.startResultFrames(bonusResultFinalFrame, 1, g.OriginalVBL())
}

func (g *Game) tickPacedBombs(inputs [2]Input) {
	b := &g.Bonus
	b.Ticks++
	if b.clearTicks != 0 {
		g.tickBonusPreparation(inputs)
		return
	}
	work := g.bombMaintenanceWork()
	if b.live {
		var before [10]Bomb
		copy(before[:], b.Bombs)
		g.tickOriginalBombObjects()
		for id, previous := range before {
			if previous.source.Flags == 0 {
				continue
			}
			work += 400
			if previous.drawn {
				work += 2000
			}
			if b.Bombs[id].drawn {
				work += 3200
			}
		}
	}
	g.updateSoundEffects()
	g.advanceBombPreparation(inputs, work)
}

func (g *Game) recordBombPreparation(stage byte, position int) {
	g.bombPreparations = append(g.bombPreparations, BombPreparationSnapshot{
		Frame: g.Frame, Position: position, Stage: stage, Counter: g.displayCounter,
		Random: g.random, Actor: g.Fighters[g.Bonus.Fighter].computerAnimation,
		Caption: g.Caption(), Bonus: g.OriginalBonusSnapshot(),
	})
}

func (g *Game) advanceBombPreparation(inputs [2]Input, position int) {
	b := &g.Bonus
	w := &b.bombPreparation
	for position < picturePreparationQuota && g.pacedBombPreparation() {
		for w.debt > 0 && position < picturePreparationQuota {
			drawingRate := 0
			if w.stage == bombPrepareRestore || w.stage == bombPrepareActor || w.stage == bombPrepareReferee {
				drawingRate = 1055
				if w.stage == bombPrepareRestore {
					drawingRate = 1070
				}
			}
			limit, rate := preparationWorkRate(position, drawingRate)
			position = advancePreparationDebt(&w.debt, &w.copyRemainder, position, limit, rate)
		}
		if w.debt != 0 {
			return
		}
		switch w.stage {
		case bombPrepareRestore:
			g.recordBombPreparation(w.stage, position)
			previous, visible := g.caption.state.Previous, g.caption.state.Visible
			g.caption.state.Alternate = g.GermanText()
			g.caption.prepare(g.caption.state.Message, g.captionContext())
			w.stage = bombPrepareText
			if previous != g.caption.state.Previous {
				g.eraseCaptionPictures(g.caption.state.Anchor)
				w.debt = 18800
				if !visible && g.caption.state.Visible {
					w.debt += 4600
				}
			}
			w.debt += captionTextWorkWithRows(g.Caption(), 105)
		case bombPrepareText:
			if !g.caption.beginUpdate(g.caption.state.Message, g.captionContext()) {
				w.stage, w.debt = bombPrepareControl, 1000
				if b.failed {
					w.debt = 25
				}
				continue
			}
			w.stage = bombPrepareReferee
			w.debt = actorPreparationWork(int(g.caption.state.RefereeFrame), int(g.caption.state.RefereeX))
		case bombPrepareReferee:
			w.stage = bombPrepareSparkles
			w.sparkles = g.caption.prepareSparkles()
		case bombPrepareSparkles:
			done := w.sparkles.advancePainting(&g.caption, func() (byte, bool) {
				cost := 200
				if w.sparkles.phase == 2 {
					cost = 275
				}
				if position+cost > picturePreparationQuota {
					return 0, false
				}
				position += cost
				before, state := g.random, g.Caption()
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
				return
			}
			w.stage, w.debt = bombPrepareControl, 1000
			if b.failed {
				w.debt = 25
			}
		case bombPrepareControl:
			if g.captionTraceEnabled {
				w.trace.After, w.trace.CompletedFrame = g.Caption(), g.Frame
				g.captionPreparations = append(g.captionPreparations, w.trace)
			}
			if b.failed {
				g.speedValue = 3
			} else {
				g.animationOrigin = g.Frame
				poses := g.humanPoses()
				for id := 0; id < 2; id++ {
					if g.Fighters[id].Human {
						g.pollHumanCommand(id, inputs[id], poses)
					}
				}
				g.tickFighter(b.Fighter, g.bonusInput(inputs))
			}
			g.recordBombPreparation(w.stage, position)
			f := g.Fighters[b.Fighter]
			w.stage, w.debt = bombPrepareActor, actorPreparationWork(f.SpriteFrame, int(f.X/2))
		case bombPrepareActor:
			f := &g.Fighters[b.Fighter]
			g.advanceFighterEffects(b.Fighter)
			f.computerAnimation.Idle &= 15
			f.aiWait = int(f.computerAnimation.Idle)
			g.tickOriginalBombBlasts()
			w.stage, w.debt = bombPrepareDetails, 3300
			for _, blast := range b.Bombs[:2] {
				if blast.source.Flags != 0 && blast.Stage == 5 {
					rows, runs := authoredPictureRows(blast.BlastFrame)
					// Blast art uses the actor RLE painter with its body mask
					// disabled; those mask writes do not consume row work.
					w.debt += 1475 + rows*139 + rows*3/2 + runs*100
					if blast.blastCounter == 1 {
						w.debt += 220
					}
				}
			}
			if g.caption.state.Visible {
				w.debt += 220
			}
		case bombPrepareDetails:
			g.rememberPicture()
			b.contactFighter, b.maskReady = g.Fighters[b.Fighter], true
			g.recordBombPreparation(w.stage, position)
			w.stage = bombPrepareWait
		case bombPrepareWait:
			if g.displayElapsed <= g.waitThreshold() || position >= 251*picturePreparationQuota/313 {
				return
			}
			position = max(position, 210*picturePreparationQuota/313)
			w.stage, w.debt = bombPrepareHeader, 10250
		case bombPrepareHeader:
			g.completeDisplayWait()
			g.recordBombPreparation(w.stage, position)
			g.dispatchSound()
			position += 700
			if !b.live {
				if b.introIRQTimer == 0 {
					b.live = true
					g.caption.state.Message = 255
				}
			} else if b.failed {
				if b.deathStarted && w.timer == 0 && b.activeCount == 0 {
					g.finishBonus(false)
					return
				}
				if !b.deathStarted {
					b.deathStarted = true
					g.soundDown[b.Fighter] = 1
					f := &g.Fighters[b.Fighter]
					f.Action, f.Move, f.reactionAction = b.failureAction, Knockdown, b.failureAction
					g.resetSourceFighter(b.Fighter)
					f.computerAnimation.Pending = byte(b.failureAction)
				}
				if w.timer == 0 {
					animating := g.Fighters[b.Fighter].computerAnimation.Fallen == 0
					g.advanceBonusDeathActor(7)
					if animating {
						// The reaction advances before restoration, whereas a
						// settled actor skips that animation task entirely.
						position += 800
					}
					w.timer = 3
				}
			} else if b.spawnDone && b.activeCount == 0 {
				g.finishBonus(true)
				return
			}
			g.beginBombRestore()
		}
	}
}
