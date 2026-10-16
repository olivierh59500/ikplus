package game

func (g *Game) tickCombat(inputs [2]Input) {
	g.animateCombat(inputs)
	if g.animationTick() {
		g.resolveHits()
	}
}

func (g *Game) animateCombat(inputs [2]Input) {
	// $777E polls both humans through $6D24 before any actor's animation.
	if g.animationTick() {
		poses := g.humanPoses()
		for id := 0; id < len(inputs); id++ {
			if g.Fighters[id].Human {
				g.pollHumanCommand(id, inputs[id], poses)
			}
		}
	}
	for i := range g.Fighters {
		f := &g.Fighters[i]
		var in Input
		if f.Human && i < len(inputs) {
			in = f.lastInput
		} else {
			in = g.chooseOriginalAI(i)
		}
		g.tickFighter(i, in)
	}
}

func (g *Game) tickFighter(id int, input Input) {
	if input.hasOriginalAction {
		g.tickComputerAnimation(id, input)
		return
	}
	g.tickHumanAnimation(id, input)
}

// $10FE runs these timers for all three fighters on each active PAL VBL,
// including judging and bonus presentation. Animation ownership is irrelevant.
func (g *Game) tickFighterTimers() {
	for id := range g.Fighters {
		f := &g.Fighters[id]
		if f.Stun > 0 {
			f.Stun--
		}
		if f.recoveryWait > 0 {
			f.recoveryWait--
		} else if f.GetUp > 0 {
			f.GetUp--
		}
	}
}

type contact struct {
	attacker, target, score, points, stun int
	move                                  Move
	frame                                 int
	orientation                           int
}

func (g *Game) resolveHits() {
	for _, h := range g.findHits() {
		g.applyContact(h)
	}
}

func (g *Game) findHits() []contact {
	g.soundHits = [3]byte{}
	// $7666 visits the original six ordered pairs from index5 down to0.
	// $78BA skips the rest of the attacker's group after a hit; newly down
	// victims cannot receive another award in the same update.
	var hits []contact
	pending := [3]bool{}
	attackers := [6]int{0, 0, 1, 1, 2, 2}
	targets := [6]int{2, 1, 2, 0, 1, 0}
	skips := [6]int{0, 0, 1, 1, 3, 3}
	for pair := 5; pair >= 0; pair-- {
		i, j := attackers[pair], targets[pair]
		a, t := &g.Fighters[i], &g.Fighters[j]
		frame := poseFrame(*a)
		move := a.Move
		if a.computerAnimationReady {
			if sourceMove, recognized := sourceStrikeMove(frame); recognized {
				move = sourceMove
			}
		}
		index := strikeIndex(frame)
		if index < 0 || a.computerAnimationReady && a.computerAnimation.Held >= 2 {
			continue
		}
		down := t.Move == Knockdown
		if t.computerAnimationReady {
			// $7700 tests the shared $80B byte, independently of the pose.
			down = g.soundDown[j] != 0
		}
		if pending[j] || down {
			continue
		}
		orientation, intersects := sourceContact(*a, *t, frame)
		if !intersects {
			continue
		}
		front := sourceCollisions.rear[orientation] == 0
		points := 2
		score, stun := strikeAward(index, front)
		if !front {
			points = 1
		}
		hits = append(hits, contact{i, j, score, points, stun, move, frame, orientation})
		pending[j] = true
		pair = skips[pair]
	}

	return hits
}

func (g *Game) applyContact(h contact) {
	g.Fighters[h.attacker].hitMask |= 1 << h.target
	g.soundHits[h.attacker] = 1
	if h.attacker < 2 {
		// $7B8C makes the last white/red scorer the final actor drawn.
		g.drawPriority = byte(h.attacker)
	}
	a, t := &g.Fighters[h.attacker], &g.Fighters[h.target]
	g.soundDown[h.target] = byte(h.points)
	g.setHitEffects(h)
	// $7CCC saturates at7, so a fighter on5 can win with a frontal hit.
	a.Points = min(7, a.Points+h.points)
	a.addOriginalScore(h.score)
	t.Move, t.Age, t.Height, t.velocity, t.lastAnimationStep = Knockdown, 0, 0, 0, -1
	t.Stun = max(t.Stun, h.stun-25)
	t.recoveryWait = max(t.recoveryWait, h.stun)
	t.reactionAction = int(sourceCollisions.reactions[strikeIndex(h.frame)*4+(h.orientation&3)] & 127)
	t.Action = t.reactionAction
	// $9830 selects a reaction immediately for either owner; $98F8
	// publishes its first pose on the next animation update.
	previousThreat := t.computerAnimation.Threat
	g.resetSourceFighter(h.target)
	// $7C0A publishes the reaction as pending input. $9844 clears
	// animation flags but leaves $817 until the next pose is published.
	t.computerAnimation.Pending = byte(t.reactionAction)
	t.computerAnimation.Threat = previousThreat
	t.GetUp = int(nativeGameplay.Animation.GetUp[g.sourceDifficulty()])
	g.emit(HitEvent, h.attacker, h.target, t.X, GroundY-24, h.score)
	g.Events[len(g.Events)-1].Move = h.move
	g.Events[len(g.Events)-1].Frame = h.frame
}
