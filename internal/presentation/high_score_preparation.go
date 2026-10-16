package presentation

import "github.com/olivierh59500/ikplus/internal/game"

const (
	scorePreparationQuota = 65536
	scoreEntryWork        = 1100
	scoreClearBankWork    = 30170
	scoreRestoreWork      = 30552
	scoreClockWork        = 10309
	scoreShadowWork       = 3200
	scoreUpdateWork       = 7500
	scoreRippleWork       = 1500
	scoreWaterWork        = 1500
	// Isolated scenery instruction counts are CPU cycles. The fitted physical
	// water and drop allowances are color clocks from the complete score recording.
	scoreSelectionWork   = 2152 * scorePreparationQuota / (2 * 313 * 227)
	scoreOutlineIdleWork = (130 - 772) * scorePreparationQuota / (2 * 313 * 227)
	scoreOutlineDrawWork = (2214 - 130) * scorePreparationQuota / (2 * 313 * 227)
	scoreWaterWrapWork   = 1230 * scorePreparationQuota / (313 * 227)
	scoreDropSelectWork  = 542 * scorePreparationQuota / (313 * 227)
	scoreDropInitialWork = 1146 * scorePreparationQuota / (313 * 227)
	scoreDropEvenWork    = 1517 * scorePreparationQuota / (313 * 227)
	scoreDropOddWork     = 1777 * scorePreparationQuota / (313 * 227)
	scoreDropEndWork     = 2751 * scorePreparationQuota / (313 * 227)
	scoreTableSetupWork  = 2431
	scoreWindowStart     = 210
	scoreWindowEnd       = 250
)

type scorePaintJob struct {
	work  int
	paint func()
}

// scorePreparation spreads authored picture operations over native updates.
// Its drawing list owns work capacity, never source instructions or memory.
type scorePreparation struct {
	sequence      *highScoreSequence
	jobs          []scorePaintJob
	position      int
	waiting       bool
	windowPending bool
}

func newPacedHighScoreSequence(p *sceneryPlanes, position int) *highScoreSequence {
	h := initializeHighScoreActors(p)
	w := &scorePreparation{sequence: h, position: position + scoreEntryWork}
	h.preparation = w
	w.clearBank(&p.pictures[0], nil)
	w.clearBank(&p.pictures[1], func() {
		p.state.ScoresVisible, p.state.FrameWraps = 1, 0
		p.state.Frame &= 15
		w.picture()
	})
	w.advance()
	return h
}

func (w *scorePreparation) clearBank(bank *packedBitmap, done func()) {
	const rows = 83
	for row := 117; row < 200; row++ {
		work := scoreClearBankWork / rows
		if row-117 < scoreClearBankWork%rows {
			work++
		}
		start := row * 160
		w.add(work, func() { copy(bank.pixels[start:start+160], w.sequence.planes.artwork.Background[start:start+160]) })
	}
	w.add(0, done)
}

func (w *scorePreparation) add(work int, paint func()) {
	w.jobs = append(w.jobs, scorePaintJob{work, paint})
}

func (w *scorePreparation) beginUpdate() {
	w.position = w.sequence.updateWork
}

// The physical score entry reaches this wait after its next publication when
// the active ripple is on the captured phase boundary. Retain that ordering
// from typed score/scenery state rather than from a frame-number exception.
func (w *scorePreparation) deferDisplayWait() bool {
	h := w.sequence
	p := h.planes
	return w.position >= (scoreWindowEnd+1)*scorePreparationQuota/313 &&
		p.state.FirstScore == 34 && h.ambient.Mode == 3 && h.ambient.X&7 == 7 && h.ambient.Counter == 251
}

func scoreSceneUpdateWork(before, after game.AmbientState, frame byte, waterWrap bool) int {
	work := scoreUpdateWork
	if before.Mode == 3 {
		work += scoreRippleWork
	}
	if frame&1 == 0 {
		work += scoreWaterWork
	}
	if after.SelectionCount != before.SelectionCount {
		// The original idle dispatcher selects and initializes the authored
		// scenery here. Its measured 2,152 instruction cycles are part of the
		// update that competes with retained score drawing.
		work += scoreSelectionWork
	} else if before.Mode == 0 && after.Mode == 5 {
		// Script selection is a separate authored branch and does not advance
		// the ordinary scenery selection counter.
		work += scoreDropSelectWork
	}
	if before.Mode == 2 {
		work += scoreOutlineIdleWork
		if after.Mode == 2 && after.Counter&3 == 0 {
			work += scoreOutlineDrawWork
		}
	}
	if waterWrap {
		// Wrapping the water phase refreshes all twenty-one jitter entries.
		work += scoreWaterWrapWork
	}
	if before.Mode == 5 {
		switch {
		case after.Mode == 0:
			work += scoreDropEndWork
		case before.ScriptInitial:
			work += scoreDropInitialWork
		case frame&1 == 0:
			work += scoreDropEvenWork
		default:
			work += scoreDropOddWork
		}
	}
	return work
}

func (w *scorePreparation) advance() {
	h, p := w.sequence, w.sequence.planes
	for !h.ended && w.position < scorePreparationQuota {
		if len(w.jobs) != 0 {
			used := min(w.jobs[0].work, scorePreparationQuota-w.position)
			w.jobs[0].work -= used
			w.position += used
			if w.jobs[0].work != 0 {
				return
			}
			paint := w.jobs[0].paint
			w.jobs = w.jobs[1:]
			if paint != nil {
				paint()
			}
			continue
		}
		if !w.windowPending {
			if !w.waiting || p.state.Elapsed <= p.artwork.Score.SpeedDelay[p.state.Speed+p.state.Fast] {
				return
			}
			if w.deferDisplayWait() {
				return
			}
			h.beginDisplayWait()
			w.waiting, w.windowPending = false, true
		}
		if w.position >= (scoreWindowEnd+1)*scorePreparationQuota/313 {
			return
		}
		w.position = max(w.position, scoreWindowStart*scorePreparationQuota/313)
		w.windowPending = false
		h.finishDisplayWait()
		w.add(scoreClockWork, func() {
			if !h.advanceScoreRow() {
				return
			}
			copy(h.actors.Wait[:], p.state.Recovery[:3])
			copy(h.actors.Rest[:], p.state.Rest[:3])
			before := uint16(h.random >> 48)
			h.actors.Step(func() byte { return byte(h.randomWord()) }, p.state.Keys[0])
			h.publishActors()
			draws := int(uint16(h.random>>48) - before)
			w.add((2241+draws*580)*scorePreparationQuota/(313*227), w.picture)
		})
	}
}

func (w *scorePreparation) picture() {
	h, p := w.sequence, w.sequence.planes
	bank := p.pictureBuffer(p.state.Drawing)
	w.add(scoreRestoreWork, func() {
		h.restoreBottom(bank)
		w.actor(bank, 1, func() {
			w.actor(bank, 0, func() {
				if p.state.ScoreForeground != 0 {
					w.table(bank, func() { w.blueAndShadow(bank, func() { w.waiting = true }) })
				} else {
					w.blueAndShadow(bank, func() { w.table(bank, func() { w.waiting = true }) })
				}
			})
		})
	})
}

func (w *scorePreparation) actor(bank *packedBitmap, id int, done func()) {
	p := w.sequence.planes
	p.state.ActorCursor = byte(id)
	a := p.state.Actors[id]
	rows, runs := 0, 0
	for _, run := range p.artwork.Fighters[a.Frame].Runs {
		rows += len(run.Rows)
		runs++
	}
	rowWork := 140
	if id == 1 {
		rowWork += 8
	} else if id == 2 {
		rowWork -= 2
		if a.Frame == 0 && a.Position == 78 && p.state.FirstScore == 8 {
			// The empty blue pose has the lower physical row return observed
			// independently at the late score-window boundary.
			rowWork -= 4
		}
	}
	work := 1439 + 13*int(a.Facing) + rows*rowWork + runs*166 + (rows*3+40)*int(a.Position&7)*1844/1000
	if a.Frame >= 36 && a.Frame <= 47 {
		work += 132
	} else if a.Frame >= 48 {
		work += 13
	}
	w.add(work, func() {
		painter := originalScene{planes: p}
		painter.frameAt(bank, id, int(a.Frame), int(a.Position), a.Facing != 0, nil, true)
		done()
	})
}

func (w *scorePreparation) blueAndShadow(bank *packedBitmap, done func()) {
	w.actor(bank, 2, func() {
		w.add(scoreShadowWork, func() {
			painter := originalScene{planes: w.sequence.planes}
			painter.projectShadowsAt(bank)
			done()
		})
	})
}

func (w *scorePreparation) table(bank *packedBitmap, done func()) {
	p := w.sequence.planes
	plan := (highScoreTable{p}).prepare()
	rows := len(plan.glyphs) * 5
	lines := min(6, 50-int(p.state.FirstScore))
	work := scoreTableSetupWork + lines*892 + len(plan.glyphs)*1196 + rows*922/1000
	for row := 0; row < rows; row++ {
		cost := work / rows
		if row < work%rows {
			cost++
		}
		w.add(cost, func() { plan.advance(p, bank, 1) })
	}
	w.add(0, done)
}
