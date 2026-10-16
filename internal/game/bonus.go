package game

// BonusPreparing reports the clearing pictures before BA02 resets the shared
// counters and projectile memory for the new challenge.
func (g *Game) BonusPreparing() bool {
	return (g.Phase == BallBonus || g.Phase == BombBonus) && g.Bonus.clearTicks > 0
}

// StartBonus opens a bonus stage directly for practice and reference capture.
// Kind 0 selects balls; kind 1 selects bombs. Normal matches alternate them.
func (g *Game) StartBonus(kind, fighter int) {
	fighter = min(max(fighter, 0), 1)
	if !g.Fighters[fighter].Human {
		g.Fighters[fighter].Human = true
	}
	g.preparePracticeBonus(kind)
	g.Bonus = BonusState{queue: []int{fighter}}
	g.startBonus(fighter)
}

// StartBonusSequence practices all surviving humans in the original red-to-
// white order. Unlike StartBonus, it retains the second player's full turn.
func (g *Game) StartBonusSequence(kind int) {
	if !g.Fighters[0].Human && !g.Fighters[1].Human {
		g.Fighters[0].Human = true
	}
	g.preparePracticeBonus(kind)
	var queue []int
	for id := 1; id >= 0; id-- {
		if g.Fighters[id].Human {
			queue = append(queue, id)
		}
	}
	g.Bonus = BonusState{queue: queue}
	g.startBonus(queue[0])
}

func (g *Game) preparePracticeBonus(kind int) {
	g.demonstrating, g.demoDifficulty, g.clockTicks = false, 0, 0
	g.bonusCount = 1
	if kind == 1 {
		g.bonusCount = 2
	}
	g.rememberBonusProjectiles()
}

func (g *Game) bonusInput(inputs [2]Input) Input {
	id := g.Bonus.Fighter
	if id < len(inputs) {
		return normalize(inputs[id])
	}
	return Input{}
}

func (g *Game) advanceBombFuse(bomb *Bomb) bool {
	return g.advanceSourceBombFuse(bomb, bomb.Stage)
}

func (g *Game) finishBonus(success bool) {
	g.Bonus.Success = success
	g.Phase, g.PhaseTicks = BonusEnd, 0
	g.Bonus.resultActor = g.Bonus.Fighter
	g.Bonus.beginResultWait(g.OriginalVBL())
	g.Message = "BONUS OVER"
	if success {
		g.caption.state.Amount = 0x50
		if g.pacedBallPreparation() {
			g.caption.state.Message = 21
		} else {
			g.startCaption(21)
			g.tickCaption()
		}
		g.Fighters[g.Bonus.Fighter].addOriginalScore(5000)
		g.Message = "PERFECT BONUS +5000"
	}
	g.emit(BonusEvent, g.Bonus.Fighter, -1, 160, GroundY, 0)
}

func (g *Game) afterBonus() {
	g.rememberBonusProjectiles()
	queue := g.Bonus.queue
	if len(queue) > 1 {
		g.Bonus.queue = queue[1:]
		g.startBonus(queue[1])
		return
	}
	if g.Bonus.kind == 1 {
		g.speedValue = g.Bonus.savedSpeed + 1
		// $C78C restores COLOR1 to $00E after the bomb challenge.
		g.effectColor = 0x00e
	}
	g.advanceRound()
}
