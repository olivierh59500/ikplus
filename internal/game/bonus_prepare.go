package game

func (g *Game) rememberBonusProjectiles() {
	b := &g.Bonus
	if b.turn == 0 {
		return
	}
	g.bonusNumberMemory, g.bonusFailedMemory, g.bonusDoneMemory = b.Number, b.failed, b.spawnDone
	if b.kind == 1 {
		for id, p := range b.Bombs {
			if id < 10 {
				g.bonusMemory[id] = p
			}
		}
	} else {
		g.bonusAccumulator = b.spawnAccumulator
		for id, p := range b.Balls {
			if id >= 4 {
				break
			}
			g.bonusMemory[id].source = p.source
			g.bonusMemory[id].Frame, g.bonusMemory[id].Color = p.Frame, p.Color
			g.bonusMemory[id].drawn = p.drawn
		}
		g.bonusMemory[4].source = b.shieldMotion
		g.bonusMemory[4].Frame, g.bonusMemory[4].Color = b.shieldFrame, b.shieldColor
	}
}

// BA02 resets active motion and color, but leaves the vertical fraction and
// the auxiliary phase counters in the original shared projectile storage.
func resetBonusMotion(p *Bomb) {
	p.source.Flags, p.source.Gravity = 0, 0
	p.source.X, p.source.Y = 0, p.source.Y&255
	p.source.VX, p.source.VY = 0, 0
	p.Color, p.drawn = 0, false
}

func (g *Game) startBonus(id int) {
	g.rememberBonusProjectiles()
	queue, turn := g.Bonus.queue, g.Bonus.turn
	savedSpeed := g.OriginalSpeed()
	if turn != 0 {
		savedSpeed = g.Bonus.savedSpeed
	}
	if turn == 0 {
		g.hideCaption()
	}
	kind := 0
	if g.bonusCount%2 == 0 {
		kind = 1
	}
	g.Bonus = BonusState{Fighter: id, Total: 64, spawnWait: 88, launchRate: 128, ballGravity: 12, queue: queue, kind: kind, turn: turn + 1, introRemaining: 100}
	g.Bonus.savedSpeed, g.Bonus.clearingPeriod = savedSpeed, byte(g.animationTicks())
	g.Phase, g.PhaseTicks, g.Message = BallBonus, 0, "DEFLECT THE BALLS"
	if kind == 1 {
		g.Phase, g.Message = BombBonus, "KICK AWAY THE BOMBS"
		g.Bonus.Total, g.Bonus.spawnWait, g.Bonus.bombFuse = bombGoal, 25, 48
		g.Bonus.introRemaining = 150
		g.Bonus.bombDelay = int(nativeGameplay.Bonus.BombDelays[min(max(g.bonusCount/2-1, 0), 7)])
		g.Bonus.Bombs = make([]Bomb, 10)
		for slot := range g.Bonus.Bombs {
			g.Bonus.Bombs[slot] = g.bonusMemory[slot]
		}
	} else {
		g.Bonus.spawnAccumulator = g.bonusAccumulator
		g.Bonus.Balls = make([]Ball, 4)
		for slot := range g.Bonus.Balls {
			p := g.bonusMemory[slot]
			g.Bonus.Balls[slot] = Ball{source: p.source, Frame: p.Frame, Color: p.Color, drawn: p.drawn}
		}
		p := g.bonusMemory[4]
		g.Bonus.shieldMotion = p.source
		g.Bonus.shieldFrame, g.Bonus.shieldColor = p.Frame, p.Color
	}
	if turn == 0 {
		g.Bonus.Number, g.Bonus.failed, g.Bonus.spawnDone = g.bonusNumberMemory, g.bonusFailedMemory, g.bonusDoneMemory
		// B8FE and C7C4 hide all actors before the first player turn.
		g.Bonus.clearTicks = 2
		if kind == 1 {
			g.Bonus.clearTicks = 2 * int(g.Bonus.clearingPeriod)
		} else if !g.Bonus.failed && !g.pacedBallPreparation() {
			g.prepareClearingShieldPosition()
		}
		for actor := range g.Fighters {
			f := &g.Fighters[actor]
			f.SpriteFrame = 87
			f.X = 272
			if kind == 1 {
				f.X, f.Facing = 240, -1
			}
			f.computerAnimation.Frame = 87
			f.computerAnimation.Position = byte(f.X / 2)
			f.computerAnimation.Facing = 0
			if f.Facing < 0 {
				f.computerAnimation.Facing = 1
			}
		}
	}
	g.animationOrigin = g.Frame
	if g.Bonus.clearTicks == 0 {
		g.setupBonusActor()
	}
	if g.pacedBallPreparation() {
		g.Bonus.preparation = ballPreparationWork{stage: ballWorkEntry, debt: 51000}
		g.advanceBallPreparation(g.joystick, g.ballMaintenanceWork())
	} else {
		g.rememberPicture()
	}
	g.emit(BonusEvent, id, -1, g.Fighters[id].X, GroundY, g.bonusCount)
}

func (g *Game) prepareClearingShieldPosition() {
	neutral := shieldControls.States[0]
	shield := &g.Bonus.shieldMotion
	shield.X = uint16(neutral.ShieldX&248)<<8 | shield.X&255
	shield.Y = uint16(neutral.Y-96)<<8 | shield.Y&255
}

func (g *Game) setupBonusActor() {
	b := &g.Bonus
	b.introIRQTimer = byte(b.introRemaining)
	// BA02 runs after the two clearing pictures, not at scene dispatch.
	b.Number, b.failed, b.spawnDone = 0, false, false
	if b.kind == 1 {
		g.speedValue = 2 // $C648 temporarily selects source speed one.
		for slot := range b.Bombs {
			resetBonusMotion(&b.Bombs[slot])
		}
	} else {
		for slot := range b.Balls {
			p := Bomb{source: b.Balls[slot].source, Frame: b.Balls[slot].Frame, Color: b.Balls[slot].Color}
			resetBonusMotion(&p)
			b.Balls[slot].source, b.Balls[slot].Color, b.Balls[slot].drawn = p.source, p.Color, false
		}
		p := Bomb{source: b.shieldMotion, Frame: b.shieldFrame, Color: b.shieldColor}
		resetBonusMotion(&p)
		b.shieldMotion, b.shieldColor = p.source, p.Color
	}
	f := &g.Fighters[b.Fighter]
	g.soundDown[b.Fighter] = 0
	g.ownerFlash[b.Fighter] = 20
	f.Y, f.Height, f.Age = GroundY, 0, 0
	f.selected, f.lastInput, f.humanCommandFrame = Idle, Input{}, 0
	if b.kind == 1 {
		// $C5CC selects the bomb palette before skipping an absent red owner.
		g.effectColor = 0xe00
		f.Move, f.Action = Idle, 0
		threat := f.computerAnimation.Threat
		g.resetSourceFighter(b.Fighter)
		f.computerAnimation.Threat = threat
	} else {
		b.entryFacing = f.Facing
		f.Move = Shield
		g.setShieldState(b.Fighter, shieldControls.Initial)
		g.tickInitialShield(g.bonusInput(g.joystick))
		b.ShieldVisible = true
		b.shieldMotion.Flags, b.shieldMotion.Floor = 1, 115
		g.syncShieldMotionFromPose()
	}
	g.animationOrigin = g.Frame
	message := byte(22)
	if b.kind == 1 {
		message = 32
	}
	if g.pacedBallPreparation() || g.pacedBombPreparation() {
		g.caption.state.Message = message
	} else {
		g.startCaption(message)
	}
	b.contactFighter, b.maskReady = *f, true
}

func (g *Game) syncShieldMotionFromPose() {
	g.syncShieldMotionFromActor(g.Bonus.Fighter)
}

func (g *Game) syncShieldMotionFromActor(id int) {
	f := g.Fighters[id]
	p := &g.Bonus.shieldMotion
	p.X = uint16(int(f.ShieldX+32)/2)<<8 | p.X&255
	p.Y = uint16(int(f.ShieldY)-96)<<8 | p.Y&255
}

func (g *Game) tickBonusPreparation(inputs [2]Input) bool {
	b := &g.Bonus
	if b.clearTicks > 0 {
		g.updateSoundEffects()
		b.clearTicks--
		if b.clearTicks == 0 {
			if b.kind == 1 && b.turn == 1 {
				// $C5FA..$C60A clears both pointer lists after $C7C4/$96E0
				// finishes the initial clearing waits, before the first turn.
				for slot := range b.Bombs {
					b.Bombs[slot].drawn = false
					b.Bombs[slot].groundPicture = bombGroundPicture{}
				}
			}
			g.setupBonusActor()
			if g.pacedBombPreparation() {
				g.beginBombRestore()
				g.advanceBombPreparation(inputs, 236*picturePreparationQuota/313+200)
				return true
			}
			g.tickCaption()
			if b.kind == 1 {
				g.tickBombActor(inputs)
			} else {
				g.tickShield(b.Fighter, g.bonusInput(inputs))
				g.syncShieldMotionFromPose()
			}
		}
		return true
	}
	if b.live {
		return false
	}
	if b.kind == 0 && b.turn > 1 {
		g.tickOriginalBallObjects(b.contactFighter)
	}
	g.updateSoundEffects()
	if b.introIRQTimer == 0 && (b.kind == 0 || b.introExpiredFrozen) {
		if b.kind == 1 && !g.animationTick() {
			return true
		}
		b.introRemaining, b.introExpiredFrozen = 0, false
		if b.kind == 1 {
			g.tickBombActor(inputs)
		} else {
			g.Fighters[b.Fighter].Age++
			b.contactFighter = g.Fighters[b.Fighter]
			g.tickShield(b.Fighter, g.bonusInput(inputs))
			g.syncShieldMotionFromPose()
		}
		b.live = true
		g.hideCaption()
		return true
	}
	if b.kind == 0 {
		// $B972 tests the IRQ-owned timer after every completed beam wait.
		// A freeze may consume only part of that deadline; main-loop ticks
		// must not extend it or prepare another caption after it expires.
		b.introRemaining = int(b.introIRQTimer)
	} else if b.introRemaining > 0 {
		b.introRemaining--
	}
	if b.kind == 1 {
		g.tickBombActor(inputs)
		if g.animationTick() {
			g.tickCaption()
		}
		if b.introRemaining == 0 && g.animationTick() {
			b.live = true
			g.hideCaption()
		}
	} else {
		g.tickCaption()
		g.Fighters[b.Fighter].Age++
		b.contactFighter = g.Fighters[b.Fighter]
		g.tickShield(b.Fighter, g.bonusInput(inputs))
		g.syncShieldMotionFromPose()
		if b.introRemaining == 0 {
			b.live = true
			g.hideCaption()
		}
	}
	return true
}

// The bomb main loop polls both human slots but animates only the active turn.
// After a hit it selects the original reaction, then advances at normal speed.
func (g *Game) tickBombActor(inputs [2]Input) bool {
	b := &g.Bonus
	f := &g.Fighters[b.Fighter]
	if g.animationTick() {
		if g.Frame > g.animationOrigin {
			// $C87E dispatches the preceding pose after its completed wait.
			g.dispatchSound()
		}
		b.contactFighter, b.maskReady = *f, true
	}
	if b.failed {
		if !b.deathStarted {
			if !g.animationTick() {
				return false
			}
			b.deathStarted = true
			g.speedValue = 3 // $C840 restores normal cadence for the death pose.
			g.soundDown[b.Fighter] = 1
			g.animationOrigin = g.Frame
			f.Action, f.Move, f.reactionAction = b.failureAction, Knockdown, b.failureAction
			g.resetSourceFighter(b.Fighter)
			f.computerAnimation.Pending = byte(b.failureAction)
			g.advanceBonusDeathActor(7)
			b.deathVisible = true
			return true
		}
		f.Age++
		if !g.animationTick() {
			return false
		}
		s := &f.computerAnimation
		if s.Fallen == 0 {
			g.advanceBonusDeathActor(7)
		}
		b.deathVisible = true
		return true
	}
	if g.animationTick() {
		poses := g.humanPoses()
		for id := 0; id < 2; id++ {
			if g.Fighters[id].Human {
				g.pollHumanCommand(id, inputs[id], poses)
			}
		}
	}
	g.tickFighter(b.Fighter, g.bonusInput(inputs))
	if g.animationTick() {
		f.computerAnimation.Idle &= 15
		f.aiWait = int(f.computerAnimation.Idle)
		return true
	}
	return false
}

func (g *Game) advanceBonusDeathActor(stage byte) {
	f := &g.Fighters[g.Bonus.Fighter]
	s := &f.computerAnimation
	if s.Fallen != 0 {
		return
	}
	round := byte((g.Round/10)*16 + g.Round%10)
	s.advanceHumanWithKey(stage, round, g.selectorKey(), byte(min(f.recoveryWait, 255)), byte(min(f.GetUp, 255)), g.originalRandomByte, func(flags byte) { g.publishFighterEffects(g.Bonus.Fighter, flags) })
	if stage == 7 && !g.pacedBombPreparation() {
		s.Idle &= 15
	}
	f.Action, f.SpriteFrame = int(s.Action), int(s.Frame)
	if g.audioControl != nil && (s.Frame == 42 || s.Frame == 46) {
		// $BA5A supplies the bonus death animation's delayed vocal sample.
		id := byte(14)
		if s.Frame == 46 {
			id = 13
		}
		g.audioControl.State.Delay, g.audioControl.State.Delayed = 3, id
	}
	f.AnimationStep = int(s.Current) - int(nativeGameplay.Animation.Starts[s.Action])
	f.X = float64(s.Position) * 2
	f.Facing = 1
	if s.Facing != 0 {
		f.Facing = -1
	}
}

func (g *Game) tickBombs(inputs [2]Input) {
	if g.pacedBombPreparation() {
		g.tickPacedBombs(inputs)
		return
	}
	b := &g.Bonus
	b.Ticks++
	if g.tickBonusPreparation(inputs) {
		return
	}
	g.tickOriginalBombObjects()
	g.updateSoundEffects()
	if !g.tickBombActor(inputs) {
		return
	}
	g.tickOriginalBombBlasts()
	if b.activeCount == 0 {
		if b.failed && b.deathVisible {
			g.finishBonus(false)
		} else if !b.failed && b.spawnDone {
			g.finishBonus(true)
		}
	}
}

// A second surviving player starts in the same VBL as the preceding return.
// The bomb failure's 75-VBL pause also lets the first normal wait pass without
// another interrupt; the source therefore prepares two actor pictures at once.
func (g *Game) continueAfterBonus(inputs [2]Input) {
	extraPicture := g.Bonus.kind == 1 && !g.Bonus.Success
	g.afterBonus()
	if (g.Phase != BallBonus && g.Phase != BombBonus) || g.Bonus.turn < 2 {
		return
	}
	pictures := 1
	if extraPicture {
		pictures++
	}
	for picture := 0; picture < pictures; picture++ {
		if picture > 0 && g.Bonus.kind == 1 {
			// The already-expired first wait reaches $C87E between the
			// two preparations even though no new VBL has occurred.
			g.dispatchSound()
		}
		g.tickCaption()
		if g.Bonus.kind == 1 {
			g.tickBombActor(inputs)
		} else {
			g.tickShield(g.Bonus.Fighter, g.bonusInput(inputs))
			g.syncShieldMotionFromPose()
		}
		if extraPicture && picture == 0 {
			// The source paints this actor/caption before its immediately
			// completed wait swaps banks for the second preparation.
			g.rememberPicture()
			g.completeDisplayWait()
		}
	}
}

// $B7B2 enters state three before the first control transition has written
// the actor's facing. Its side choice therefore uses the preceding pose.
func (g *Game) tickInitialShield(input Input) {
	b := &g.Bonus
	direction := shieldControls.Directions[joystickNibble(input)]
	state := shieldControls.Transitions[shieldControls.Initial][direction]
	if state&128 != 0 {
		facing := 0
		if b.entryFacing < 0 {
			facing = 1
		}
		state = (state & 3) + ((facing ^ 1) << 2)
	}
	g.setShieldState(b.Fighter, state)
}
