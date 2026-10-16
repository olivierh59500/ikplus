package game

// These decision tables are copied byte-for-byte from the original Amiga
// executable. Each comment records its address for independent verification.
var (
	// Original address 0x6E5E.
	aiOther = [...]byte{1, 0, 0}
	// Original address 0x6E61.
	aiAlternative = [...]byte{2, 2, 1}
	// Original address 0x6FDE.
	aiTargets = [...]byte{0, 0, 0, 1, 1, 1, 2, 2, 1, 2, 2, 0, 0, 0, 0, 1, 1, 1}
	// Original address 0x6EEC.
	aiRoundDefence = [...]byte{128, 0, 0, 2, 128, 1, 128, 0}
	// Original address 0x6EF4.
	aiRoundActivity = [...]byte{128, 0, 0, 5, 0, 5, 5, 5}
	// Original address 0x6EFC.
	aiRoundTarget = [...]byte{0, 0, 0, 0, 0, 6, 0, 6}
	// Original address 0x72A2.
	aiCounterThreshold = [...]byte{240, 160, 96, 32, 8, 0}
	// Original address 0x72A8.
	aiCounterFallbackThreshold = [...]byte{224, 192, 128, 64, 16, 16}
	// Original address 0x72AE.
	aiActivityThreshold = [...]byte{232, 208, 96, 48, 16, 0}
	// Original address 0x72B4.
	aiRangeChoiceCount = [...]byte{12, 10, 8, 8, 6, 6}
	// Original address 0x72BA.
	aiFacingMismatch = [...]byte{0, 0, 1, 1, 1, 1, 0, 0}
	// Original address 0x72F6.
	aiTargetDownChoices = [...]byte{0, 0, 25, 27, 26, 0, 0, 0, 15, 0, 0, 0}
	// Original address 0x7303.
	aiTurnFarChoices = [...]byte{25, 27, 15, 14, 15, 15, 15, 14, 27, 27, 25, 27}
	// Original address 0x730F.
	aiTurnNearChoices = [...]byte{25, 25, 8, 8, 13, 27, 15, 25, 8, 13, 13, 26}
	// Original address 0x731B.
	aiAdvanceChoices = [...]byte{26, 26, 29, 25, 9, 25, 9, 9, 8, 13, 26, 26}
	// Original address 0x7367.
	aiRangeChoices = [...]byte{27, 15, 8, 5, 6, 4, 27, 15, 14, 11, 27, 30, 2, 4, 12, 11, 10, 30, 30, 15, 30, 2, 4, 10, 5, 6, 4, 12, 11, 10, 16, 30, 27, 26, 10, 2, 6, 2, 11, 27, 4, 10, 16, 29, 29, 27, 11, 2, 9, 4, 16, 9, 6, 29, 6, 29, 27, 4, 9, 6, 9, 4, 9, 4, 6, 5, 26, 29, 10, 2, 12, 11, 9, 9, 26, 6, 5, 30, 9, 6, 5, 16, 26, 5, 5, 9, 5, 9, 26, 6, 26, 26, 29, 5, 10, 2, 26, 26, 9, 9, 8, 29, 5, 9, 26, 6, 9, 29}
	// Original address 0x73D3.
	aiRangeOffsets = [...]byte{0, 0, 12, 24, 36, 48, 60, 72, 84, 96}
)

type aiPose struct {
	x, facing, action, flags, held, down, points, idle byte
}

type aiSnapshot struct {
	poses                                                      [3]aiPose
	id, stage, enabled, mode, second, targetOffset, difficulty byte
	defence, activity                                          int8
}

// originalDecision is a direct translation of 0x6FF0, with target selection
// from 0x6F42 and signed byte geometry from 0x7402. It returns -1 when the
// original program leaves its current animation unchanged.
func originalDecision(s aiSnapshot, random func() byte) (int, int) {
	id := int(s.id)
	if s.stage != 1 && s.enabled == 0 {
		return -1, id
	}
	target := 0
	if s.mode == 0 || s.mode == 2 {
		target = int(aiAlternative[id])
		other := int(aiOther[id])
		if int8(s.poses[target].points) <= int8(s.poses[other].points) {
			target = other
		}
		if s.poses[target].down != 0 {
			target = int(aiOther[target])
		}
	} else {
		index := int(byte(s.targetOffset + s.second*3 + s.id))
		target = int(aiTargets[index])
		if s.poses[target].down != 0 {
			target = int(aiTargets[index+6])
		}
	}
	self, other := s.poses[id], s.poses[target]
	if self.action != 0 && self.flags&16 == 0 {
		return -1, target
	}
	selfCenter := self.x + 12
	otherCenter := other.x + 12
	if self.facing != 0 {
		selfCenter += 16
	}
	if other.facing != 0 {
		otherCenter += 16
	}
	delta := int8(selfCenter - otherCenter)
	relation := (self.facing << 1) | other.facing
	if delta >= 0 {
		relation |= 4
	}
	if delta < 0 {
		delta = -delta
	}
	distance := byte(delta) >> 2
	activityGate := true
	if other.flags&64 != 0 && other.held < 4 {
		if distance < 9 {
			class := nativeGameplay.AI.CounterClasses[other.action]
			if int8(class) >= 0 {
				var choice byte
				selected := false
				if other.action == 5 || other.action == 6 || other.action == 2 {
					r := random()
					if s.defence < 0 || r >= aiCounterThreshold[s.defence] {
						choice, selected = r&3, true
					}
				}
				if !selected {
					r := random()
					if s.defence < 0 || r >= aiCounterFallbackThreshold[s.defence] {
						choice, selected = r&7, true
					}
				}
				if selected {
					if relation >= 4 {
						relation = (relation & 3) ^ 3
					}
					index := byte((class+relation*2)*8 + choice)
					return int(nativeGameplay.AI.Reactions[index]), target
				}
			}
		}
	} else if self.idle >= 48 {
		activityGate = false
	}
	if activityGate {
		if s.activity < 0 || random() < aiActivityThreshold[s.activity] {
			return -1, target
		}
	}
	if other.down != 0 {
		return int(aiTargetDownChoices[originalChoice(12, random)]), target
	}
	if aiFacingMismatch[relation] != 0 {
		choice := originalChoice(12, random)
		if distance >= 12 {
			return int(aiTurnFarChoices[choice]), target
		}
		if distance >= 8 {
			return 14, target
		}
		return int(aiTurnNearChoices[choice]), target
	}
	if distance >= 10 {
		return int(aiAdvanceChoices[originalChoice(12, random)]), target
	}
	choice := originalChoice(aiRangeChoiceCount[s.difficulty], random)
	return int(aiRangeChoices[int(aiRangeOffsets[distance])+int(choice)]), target
}

// originalChoice preserves the rejection and lower-nibble retry at 0x73DE.
func originalChoice(limit byte, random func() byte) byte {
	for {
		r := random() & 31
		if r < limit {
			return r
		}
		r &= 15
		if r < limit {
			return r
		}
	}
}

func (g *Game) chooseOriginalAI(id int) Input {
	retain := Input{hasOriginalAction: true, originalAction: -1}
	// The original computer controller runs once per normal animation update.
	if !g.animationTick() {
		return retain
	}
	level := g.sourceDifficulty()
	round := byte((g.Round/10)*16+g.Round%10) & 7
	s := aiSnapshot{id: byte(id), stage: 1, difficulty: level, defence: int8(level), activity: int8(level), targetOffset: aiRoundTarget[round]}
	if override := aiRoundDefence[round]; override != 0 {
		s.defence = int8(override)
	}
	if override := aiRoundActivity[round]; override != 0 {
		s.activity = int8(override)
	}
	for i, f := range g.Fighters {
		if f.Human {
			s.mode++
			if i == 1 {
				s.second = 1
			}
		}
		action := f.Action
		if f.Human && !f.computerAnimationReady {
			action = sourceAction(f.Move)
		}
		pose := aiPose{x: byte(int(f.X / 2)), action: byte(action), points: byte(f.Points), idle: byte(f.aiWait)}
		if f.Human && !f.computerAnimationReady {
			sequence := sourceAnimations[action]
			step := min(max(f.AnimationStep, 0), len(sequence.Flags)-1)
			pose.flags = byte(sequence.Flags[step])
			if pose.flags&8 != 0 {
				pose.held = byte(min(max(f.Age/AnimationTicks-step, 0), 255))
			}
		} else if f.computerAnimationReady && int(f.computerAnimation.Action) == f.Action {
			pose.flags = f.computerAnimation.Flags
			pose.held = f.computerAnimation.Held
			pose.idle = f.computerAnimation.Idle
		}
		if f.Facing < 0 {
			pose.facing = 1
		}
		if f.computerAnimationReady {
			// $6F42 reads $80B, which can survive the pause display even
			// after $5CA2 resets the visible action and fallen animation.
			pose.down = g.soundDown[i]
		} else if f.Move == Knockdown {
			pose.down = 1
		}
		s.poses[i] = pose
	}
	action, _ := originalDecision(s, g.originalRandomByte)
	if action >= 0 {
		return Input{hasOriginalAction: true, originalAction: action}
	}
	return retain
}

// originalRandomByte supplies the shared deterministic game stream.
func (g *Game) originalRandomByte() byte {
	value, next := RandomWord(g.random, g.OriginalVBL(), g.randomSample)
	g.random = next
	return byte(value)
}
