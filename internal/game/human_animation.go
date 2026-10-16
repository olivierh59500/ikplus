package game

// humanCommand follows the original controller at $6D24. The previous turn's
// mirror flag keeps a held direction attached to its selected action. Defensive
// commands use the original $7402 relation and eight-unit distance threshold.
func humanCommand(id int, input Input, poses [3]computerAnimationState) byte {
	self := poses[id]
	joystick := joystickNibble(input)
	if input.Fire {
		joystick |= 16
	}
	if self.Held == 0 {
		other := id ^ 1
		front, distance := humanDefenceRelation(poses[other], self)
		if !front {
			front, distance = humanDefenceRelation(self, poses[int(aiAlternative[id])])
		}
		if poses[other].Threat != 0 && front && distance < 8 {
			pose := 2
			if self.Facing != 0 {
				pose = 3
			}
			return nativeGameplay.Input.Commands[pose][joystick]
		}
	}
	pose := 0
	if self.Facing^self.Mirror != 0 {
		pose = 1
	}
	return nativeGameplay.Input.Commands[pose][joystick]
}

func humanDefenceRelation(self, other computerAnimationState) (bool, byte) {
	a, b := self.Position+12, other.Position+12
	if self.Facing != 0 {
		a += 16
	}
	if other.Facing != 0 {
		b += 16
	}
	distance := int8(a - b)
	if distance < 0 {
		distance = -distance
	}
	relation := (self.Facing << 1) | other.Facing
	if int8(a-b) >= 0 {
		relation |= 4
	}
	return nativeGameplay.Input.DefenceRelation[relation] != 0, byte(distance) >> 2
}

// advanceHuman translates the human-owned branches of $98F8, including $9C62
// rollback, held contact frames, walk release correction and original get-up.
// It uses source byte indices, not an age-to-frame approximation.
func (s *computerAnimationState) advanceHuman(stage, round byte, downWait, rest byte, random func() byte, onPublish ...func(byte)) bool {
	return s.advanceHumanWithKey(stage, round, 0, downWait, rest, random, onPublish...)
}

func (s *computerAnimationState) advanceHumanWithKey(stage, round, key byte, downWait, rest byte, random func() byte, onPublish ...func(byte)) bool {
	if stage != 6 && s.Fallen != 0 {
		if downWait != 0 || s.Pending == 0 && rest != 0 {
			return false
		}
		action := byte(37)
		if s.Frame == 39 {
			action = 35
		} else if s.Frame == 43 {
			action = 36
		}
		s.Pending = 0
		s.Idle = action
		s.selectDirect(action)
		return true
	}

humanEntry:
	s.Facing &= 1
	if stage != 6 {
		if s.Pending != s.Action {
			s.Fresh = 0
		}
		if s.Fresh == 0 && s.Cancelable != 0 {
			s.Rollback = 1
		}
		if s.Rollback != 0 {
			s.Current--
			if s.Current >= nativeGameplay.Animation.Starts[s.Action] {
				s.Frame = nativeGameplay.Animation.Frames[s.Current]
				s.move(s.Rollback ^ s.Facing)
				return false
			}
			s.dispatchWithKey(round, key, random)
			goto publish
		}
	}
	s.Current = s.Next
	if s.Action == 0 {
		s.dispatchWithKey(round, key, random)
	}

publish:
	for s.Current >= nativeGameplay.Animation.Ends[s.Action] {
		s.dispatchWithKey(round, key, random)
	}
	s.Frame = min(nativeGameplay.Animation.Frames[s.Current], 87)
	s.PreviousFlags, s.Flags = s.Flags, nativeGameplay.Animation.Flags[s.Current]
	s.Cancelable, s.Threat = s.Flags&4, s.Flags&64
	if stage != 6 {
		if s.Flags&8 != 0 {
			if s.Pending == s.Action {
				if s.Held == 0 {
					s.move(s.Facing)
				}
				if s.Held < 120 {
					s.Held++
				}
				return false
			}
			if s.Held != 0 {
				s.Held, s.Rollback, s.Cancelable = 0, 0, 0
				s.Next = s.Current + 1
				goto humanEntry
			}
		}
		if s.Flags&16 != 0 && s.Fresh == 0 {
			if s.Action == 3 || s.Action == 7 || s.Pending != 0 {
				if s.Action == 7 {
					if s.Facing != 0 {
						s.Position += 3
					} else {
						s.Position -= 3
					}
				}
				s.dispatchWithKey(round, key, random)
				goto publish
			}
		}
	}
	if s.Flags&2 != 0 {
		// Facing changes are applied after the source pose is published.
		s.Facing ^= 1
		s.Mirror = 1
	}
	for _, publish := range onPublish {
		publish(s.Flags)
	}
	if s.Flags&32 != 0 {
		s.Fallen = 1
	}
	s.move(s.Facing)
	s.Next = s.Current + 1
	return false
}

func (g *Game) humanPoses() [3]computerAnimationState {
	var poses [3]computerAnimationState
	for id, f := range g.Fighters {
		poses[id] = f.computerAnimation
		poses[id].Position = byte(int(f.X / 2))
		poses[id].Facing = 0
		if f.Facing < 0 {
			poses[id].Facing = 1
		}
	}
	return poses
}

func (g *Game) pollHumanCommand(id int, input Input, poses [3]computerAnimationState) {
	f := &g.Fighters[id]
	if !f.computerAnimationReady || int(f.computerAnimation.Action) != f.Action {
		g.resetSourceFighter(id)
		poses[id] = f.computerAnimation
	}
	f.lastInput = normalize(input)
	f.computerAnimation.Pending = humanCommand(id, f.lastInput, poses)
	f.humanCommandFrame = g.Frame
	if f.computerAnimation.Pending != 0 {
		f.roundActive = true
		f.activityByte = f.computerAnimation.Pending
	}
}

func (g *Game) tickHumanAnimation(id int, input Input) {
	f := &g.Fighters[id]
	f.Age++
	if !g.animationTick() {
		return
	}
	if !f.computerAnimationReady || int(f.computerAnimation.Action) != f.Action {
		g.resetSourceFighter(id)
	}
	if f.humanCommandFrame != g.Frame {
		g.pollHumanCommand(id, input, g.humanPoses())
	}
	s := &f.computerAnimation
	s.Position = byte(int(f.X / 2))
	s.Facing = 0
	if f.Facing < 0 {
		s.Facing = 1
	}
	previous := s.Action
	round := byte((g.Round/10)*16 + g.Round%10)
	recovered := s.advanceHumanWithKey(1, round, g.selectorKey(), byte(min(f.recoveryWait, 255)), byte(min(f.GetUp, 255)), g.originalRandomByte, func(flags byte) { g.publishFighterEffects(id, flags) })
	if recovered {
		g.soundDown[id] = 0
	}
	if recovered || s.Action != previous {
		f.hitMask, f.Age = 0, 0
		if _, attacking := sourceStrikeMove(int(s.Frame)); attacking {
			g.emit(AttackEvent, id, -1, f.X, f.Y, int(moveForSourceAction(int(s.Action))))
		}
	}
	f.Action, f.SpriteFrame = int(s.Action), int(s.Frame)
	f.AnimationStep = int(s.Current) - int(nativeGameplay.Animation.Starts[s.Action])
	f.Move = moveForSourceAction(int(s.Action))
	f.X, f.Y, f.Height = float64(s.Position)*2, GroundY, 0
	f.Facing = 1
	if s.Facing != 0 {
		f.Facing = -1
	}
	f.selected = moveForSourceAction(int(s.Pending))
	f.aiWait = int(s.Idle)
}
