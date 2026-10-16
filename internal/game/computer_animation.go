package game

// computerAnimationState preserves the original byte state surrounding $98F8.
// Pending is $7F3, Current/Next are $7F6/$7F9, Action is $7FC, Fresh is $7FF,
// Rollback/Cancelable/Held are $805/$808/$802, Idle is $811 and Fallen is $814.
type computerAnimationState struct {
	Position, Facing, Frame                         byte
	Pending, Current, Next, Action                  byte
	Fresh, Rollback, Cancelable, Held, Idle, Fallen byte
	Flags, PreviousFlags                            byte
	Mirror, Threat                                  byte
}

func (s *computerAnimationState) selectDirect(action byte) {
	s.Action = action
	s.Current = nativeGameplay.Animation.Starts[action]
	s.Next = s.Current
	s.Fresh = 1
	s.Rollback, s.Cancelable, s.Held, s.Fallen, s.Flags, s.PreviousFlags = 0, 0, 0, 0, 0, 0
	s.Mirror = 0
}

// dispatch translates $97B0, including the idle counter and occasional pose32.
// Nonzero commands store their action in $811; zero increments it instead.
func (s *computerAnimationState) dispatch(round byte, random func() byte) {
	s.dispatchWithKey(round, 0, random)
}

func (s *computerAnimationState) dispatchWithKey(round, key byte, random func() byte) {
	action := s.Pending
	if action != 0 {
		s.Idle = action
		s.selectDirect(action)
		return
	}
	s.Idle++
	// $97DE also selects pose32 for live keyboard T in rounds below five.
	// A negative idle counter takes the separate random gate first.
	pose := round < 5 && key == 0x14
	if int8(s.Idle) < 0 {
		pose = random()&127 < 8
	}
	if pose {
		s.Action = 32
		s.Idle = (random() & 63) + 1
		s.selectDirect(32)
		return
	}
	s.selectDirect(0)
}

// issue matches $727E..$729C: execute a selection, then clear pending input.
func (s *computerAnimationState) issue(action byte, round byte, random func() byte) {
	s.issueWithKey(action, round, 0, random)
}

func (s *computerAnimationState) issueWithKey(action, round, key byte, random func() byte) {
	s.Pending = action
	s.dispatchWithKey(round, key, random)
	s.Pending = 0
}

// advance translates the computer-owned path of $98F8. The controller and
// animation share state; reaching the end dispatches pending zero, not a loop
// of the completed movement. Recovery selects a new sequence one update before
// its first frame appears, preserving the original visible fallen frame.
func (s *computerAnimationState) advance(stage, level, round byte, downWait, rest byte, random func() byte, onPublish ...func(byte)) bool {
	return s.advanceWithKey(stage, level, round, 0, downWait, rest, random, onPublish...)
}

func (s *computerAnimationState) advanceWithKey(stage, level, round, key byte, downWait, rest byte, random func() byte, onPublish ...func(byte)) bool {
	if stage != 6 && s.Fallen != 0 {
		if downWait != 0 || rest >= nativeGameplay.Animation.Recovery[level] && rest != nativeGameplay.Animation.Recovery[level] {
			return false
		}
		action := byte(37)
		if s.Frame == 39 {
			action = 35
		}
		if s.Frame == 43 {
			action = 36
		}
		s.Pending = 0
		s.Idle = action
		s.selectDirect(action)
		return true
	}
	s.Current = s.Next
	if s.Action == 0 {
		s.dispatchWithKey(round, key, random)
	}
	for s.Current >= nativeGameplay.Animation.Ends[s.Action] {
		s.dispatchWithKey(round, key, random)
	}
	s.Frame = min(nativeGameplay.Animation.Frames[s.Current], 87)
	s.PreviousFlags = s.Flags
	s.Flags = nativeGameplay.Animation.Flags[s.Current]
	s.Cancelable = s.Flags & 4
	s.Threat = s.Flags & 64
	for _, publish := range onPublish {
		publish(s.Flags)
	}
	if s.Flags&2 != 0 {
		s.Facing ^= 1
		s.Mirror = 1
	}
	if s.Flags&32 != 0 {
		s.Fallen = 1
	}
	s.move(s.Facing)
	s.Next = s.Current + 1
	return false
}

// move translates $9CC6/$9CD8 and the pose-width screen clamps at $9D48.
// All position arithmetic uses original unsigned bytes, including wrap checks.
func (s *computerAnimationState) move(direction byte) {
	motion := nativeGameplay.Animation.Motion[s.Current]
	width := nativeGameplay.Animation.Widths[s.Frame]
	var left, right byte
	if direction != 0 {
		s.Position -= motion
		right = s.Position + 40
		left = right - width
	} else {
		s.Position += motion
		left = s.Position
		right = s.Position + width
	}
	if left <= 17 {
		if s.Facing == 0 {
			s.Position = 17
		} else {
			s.Position = byte(max(0, 17+int(width)-40))
		}
	} else if right > 174 {
		if s.Facing == 0 {
			s.Position = 174 - width
		} else {
			s.Position = 134
		}
	}
	if s.Position >= 208 {
		s.Position = 16
	}
}

func (g *Game) resetSourceFighter(id int) {
	f := &g.Fighters[id]
	s := computerAnimationState{Position: byte(int(f.X / 2)), Frame: byte(f.SpriteFrame)}
	if f.Facing < 0 {
		s.Facing = 1
	}
	s.Idle = byte(f.Action)
	s.selectDirect(byte(f.Action))
	f.computerAnimation = s
	f.computerAnimationReady = true
}

func (g *Game) sourceDifficulty() byte {
	if g.Phase == Judge && g.judge.active {
		return g.judge.level
	}
	if g.demonstrating {
		return byte(min(max(g.demoDifficulty, 0), 5))
	}
	return byte(min(g.bonusCount, 5))
}

func (g *Game) tickComputerAnimation(id int, input Input) {
	f := &g.Fighters[id]
	f.Age++
	if !g.animationTick() {
		return
	}
	if !f.computerAnimationReady || int(f.computerAnimation.Action) != f.Action {
		g.resetSourceFighter(id)
	}
	s := &f.computerAnimation
	s.Position = byte(int(f.X / 2))
	s.Facing = 0
	if f.Facing < 0 {
		s.Facing = 1
	}
	round := byte((g.Round/10)*16 + g.Round%10)
	if input.originalAction >= 0 {
		s.issueWithKey(byte(input.originalAction), round, g.selectorKey(), g.originalRandomByte)
		f.hitMask = 0
		f.Age = 0
	}
	stage := byte(1)
	if g.Phase == Attract {
		stage = 1
	}
	recovered := s.advanceWithKey(stage, g.sourceDifficulty(), round, g.selectorKey(), byte(min(f.recoveryWait, 255)), byte(min(f.GetUp, 255)), g.originalRandomByte, func(flags byte) { g.publishFighterEffects(id, flags) })
	if recovered {
		g.soundDown[id] = 0
		f.hitMask = 0
		f.Age = 0
	}
	f.Action = int(s.Action)
	f.SpriteFrame = int(s.Frame)
	f.AnimationStep = int(s.Current) - int(nativeGameplay.Animation.Starts[s.Action])
	f.Facing = 1
	if s.Facing != 0 {
		f.Facing = -1
	}
	f.X = float64(s.Position) * 2
	f.Move = moveForSourceAction(int(s.Action))
	f.lastInput = input
	f.aiWait = int(s.Idle)
	f.Height = 0
}
