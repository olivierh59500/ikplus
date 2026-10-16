package game

import "github.com/olivierh59500/ikplus/internal/audio"

func (g *Game) inArena() bool {
	return g.roundPreparation == 0 && (g.Phase == Attract || g.Phase == Ready || g.Phase == Fight)
}

// $1E4A prepares controllers/poses before waiting, then resolves contacts
// after the display swap. Score feedback may finish on the following update
// before the next picture is prepared.
func (g *Game) prepareFightPicture(inputs [2]Input) {
	if !g.inArena() || g.fightPrepared {
		return
	}
	seconds, _, continuing := originalRoundClock(g.clockTicks)
	g.Remaining = seconds * TickRate
	if !continuing || g.anyWinner() {
		g.endRound()
		return
	}
	// A prepared picture is an animation update even when the preceding
	// scene leaves a partially elapsed wait; the old round-origin gate drifts.
	g.animationOrigin = g.Frame
	g.beginPicture()
	g.animateCombat(inputs)
	if g.palFrameProfile == audio.Supplied313 {
		g.prepareTimedFighterEffects()
	} else {
		for id := range g.Fighters {
			g.advanceFighterEffects(id)
		}
	}
	g.fightPrepared = true
	if g.ballPreparationEnabled && g.palFrameProfile == audio.Supplied313 && g.displayElapsed != 0 {
		_, bank := g.SceneryBuffers()
		old := g.Picture(bank)
		work := 12000 + 1046 + old.restoreRows*79 + old.restoreRuns*181 + 5500
		for _, fighter := range g.Fighters {
			work += actorPreparationWork(fighter.SpriteFrame, int(fighter.X/2))
		}
		g.fightPaintDebt = max(0, work-(picturePreparationQuota-259*picturePreparationQuota/313))
	}
	if g.Phase == Ready && g.Fighters[0].Action != 28 {
		g.Phase, g.PhaseTicks, g.Message = Fight, 0, "FIGHT"
	}
	g.rememberPicture()
}

func (g *Game) tickFight(inputs [2]Input) {
	if g.Fighters[0].Action != 28 {
		g.clockTicks++
	}
	seconds, _, _ := originalRoundClock(g.clockTicks)
	g.Remaining = seconds * TickRate
	if g.fightBonusReturnHold {
		g.fightBonusReturnHold = false
		return
	}
	if g.fightCompletionPending {
		// Later impacts follow the first score's completed painting. Their
		// recovery timers begin after this update's ordinary timer tick.
		for _, h := range g.fightPendingContacts {
			g.applyContact(h)
		}
		g.fightPendingContacts = g.fightPendingContacts[:0]
		g.fightCompletionPending = false
		g.finishFightStep(inputs)
		return
	}
	if g.fightPaintDebt != 0 {
		position := g.judgeMaintenanceWork()
		used := min(g.fightPaintDebt, picturePreparationQuota-position)
		g.fightPaintDebt -= used
		position += used
		if g.fightPaintDebt != 0 || position >= 251*picturePreparationQuota/313 && g.displayElapsed > g.waitThreshold() {
			return
		}
	}
	if g.displayElapsed <= g.waitThreshold() {
		return
	}
	g.completeDisplayWait()
	var score int
	for _, fighter := range g.Fighters {
		score += fighter.Score
	}
	hits := g.findHits()
	if g.palFrameProfile == audio.Supplied313 && len(hits) > 1 {
		// Score painting separates later contacts from the first impact.
		g.fightPendingContacts = append(g.fightPendingContacts[:0], hits[1:]...)
		hits = hits[:1]
	}
	for _, h := range hits {
		g.applyContact(h)
	}
	var updated int
	for _, fighter := range g.Fighters {
		updated += fighter.Score
	}
	if g.palFrameProfile == audio.Supplied313 && (updated != score || len(g.fightPendingContacts) != 0) {
		// Complete score publication before pose sounds and the following
		// controller preparation are dispatched on the next update.
		g.fightCompletionPending = true
		return
	}
	g.finishFightStep(inputs)
}

func (g *Game) finishFightStep(inputs [2]Input) {
	g.dispatchSound()
	g.rememberFightBoundary()
	g.fightPrepared = false
	if g.pauseToggleReady() {
		g.enterPauseDemo()
		return
	}
	g.prepareFightPicture(inputs)
}

// $B968 stores20 before the next fast ball beam wait. During the result's
// forty-VBL pause that pending override is absent; clearing prepares it again.
func (g *Game) prepareBallWaitByte() {
	if g.pacedBallPreparation() {
		return
	}
	if g.Phase == BallBonus {
		g.displayElapsed = 20
		return
	}
	if g.Phase != BonusEnd || g.Bonus.kind != 0 {
		return
	}
	b := &g.Bonus
	if b.resultStage == bonusResultMessage || b.resultStage == bonusResultTail || b.resultStage == bonusResultClearing {
		g.displayElapsed = 20
	}
}

// VisibleFighters returns the last swapped poses with current HUD/ownership
// values. Prepared poses must not leak into rendering or the human pilot.
func (g *Game) VisibleFighters() [3]Fighter {
	view := g.visibleFighters
	for id := range view {
		if !g.visibleValid {
			view[id] = g.Fighters[id]
			view[id].SpriteFrame = 87
		}
		current := g.Fighters[id]
		view[id].Points, view[id].Score, view[id].RoundScore, view[id].Rank = current.Points, current.Score, current.RoundScore, current.Rank
		view[id].Human, view[id].Eliminated = current.Human, current.Eliminated
	}
	return view
}

// SceneryBuffers identifies the $634 background and $638 actor drawing banks:
// zero is $78000, one is $70000. The IRQ selects $634 for video before a main
// loop swap can change these RAM pointers. Fresh setup retains its initial
// pointers until the first completed wait, although its selector is four.
func (g *Game) SceneryBuffers() (background, actors int) {
	if !g.visibleValid {
		return 1, 0
	}
	background = int(g.displayCounter & 1)
	return background, background ^ 1
}

// DisplayBeltColors is the palette published by $EC4 at this VBL. A later
// $A938 main-thread update remains pending until the next interrupt.
func (g *Game) DisplayBeltColors() [3]uint16 { return g.beltColorsVisible }

// Observation contains public scene data only; it has no RNG or future copy.
type Observation struct {
	Phase        Phase
	Frame        uint64
	Round        int
	Fighters     [3]Fighter
	Bonus        BonusState
	PictureFrame uint64
}

func (g *Game) Observation() Observation {
	phase := g.Phase
	fighters := g.VisibleFighters()
	if phase == Fight && fighters[0].Action == 28 {
		phase = Ready
	}
	return Observation{phase, g.Frame, g.Round, fighters, g.Bonus, g.visibleAt}
}
