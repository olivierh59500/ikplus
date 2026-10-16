package game

// pauseDisplay owns $5B42's scripted actors while the tournament clock waits.
// Music, scenery, VBL counters and the three fighter timers continue running.
type pauseDisplay struct {
	positions, facing [3]byte
	indices           [3]byte
	requested         bool
	sound             bool
}

func (g *Game) RequestPause() {
	if g.OriginalMode() == 1 || g.OriginalMode() == 6 {
		g.pauseDemo.requested = true
	}
}

func (g *Game) SceneClearFrame() uint64 { return g.sceneClearFrame }

func (g *Game) PauseRequested() bool { return g.pauseDemo.requested }

func (g *Game) pauseToggleReady() bool {
	if g.pauseCooldown != 0 {
		return false
	}
	pressed := g.pauseDemo.requested || g.speedKeys[0] == 0x39
	if g.Phase == PauseDemo {
		pressed = pressed || g.joystick[0].Fire || g.joystick[1].Fire
	}
	if !pressed {
		return false
	}
	g.pauseDemo.requested = false
	g.speedKeys[0], g.pauseCooldown = 0, 25
	return true
}

func (g *Game) enterPauseDemo() {
	g.Phase, g.PhaseTicks = PauseDemo, 0
	g.sceneClearFrame = g.Frame
	g.clearArenaPictures()
	g.pauseDemo.indices = [3]byte{}
	g.pauseDemo.sound = false
	for id := 2; id >= 0; id-- {
		f := &g.Fighters[id]
		s := &f.computerAnimation
		g.pauseDemo.positions[id], g.pauseDemo.facing[id] = s.Position, s.Facing
		s.Position, s.Facing, s.Frame, s.Pending, s.Idle = nativeGameplay.Pause.Positions[id], 0, 0, nativeGameplay.Pause.Actions[0], 0
		s.selectDirect(0)
		g.publishPauseActor(id)
	}
	if g.originalRandomByte()&1 == 0 {
		g.advancePauseActor(0)
		g.advancePauseActor(0)
		g.advancePauseActor(1)
	}
	g.preparePausePicture()
}

func (g *Game) advancePauseActor(id int) {
	f := &g.Fighters[id]
	s := &f.computerAnimation
	round := byte(g.Round/10*16 + g.Round%10)
	if f.Human {
		s.advanceHumanWithKey(6, round, g.selectorKey(), byte(f.recoveryWait), byte(f.GetUp), g.originalRandomByte, func(flags byte) { g.publishFighterEffects(id, flags) })
	} else {
		s.advanceWithKey(6, g.sourceDifficulty(), round, g.selectorKey(), byte(f.recoveryWait), byte(f.GetUp), g.originalRandomByte, func(flags byte) { g.publishFighterEffects(id, flags) })
	}
	g.publishPauseActor(id)
}

func (g *Game) publishPauseActor(id int) {
	f := &g.Fighters[id]
	s := f.computerAnimation
	f.X, f.Y, f.Height = float64(s.Position)*2, GroundY, 0
	f.Facing = 1
	if s.Facing != 0 {
		f.Facing = -1
	}
	f.Action, f.SpriteFrame, f.Move = int(s.Action), int(s.Frame), moveForSourceAction(int(s.Action))
	f.computerAnimationReady = true
}

func (g *Game) preparePausePicture() {
	for id := 2; id >= 0; id-- {
		s := &g.Fighters[id].computerAnimation
		if s.Flags&16 != 0 {
			action := nativeGameplay.Pause.Actions[g.pauseDemo.indices[id]]
			s.Pending, s.Idle = action, action
			s.selectDirect(action)
			s.Pending = 0
			g.pauseDemo.indices[id] = (g.pauseDemo.indices[id] + 1) % 13
		}
	}
	for id := 2; id >= 0; id-- {
		g.advancePauseActor(id)
	}
	g.rememberPicture()
}

func (g *Game) tickPauseDemo() {
	if g.displayElapsed <= g.waitThreshold() {
		return
	}
	g.completeDisplayWait()
	if g.pauseDemo.sound {
		g.dispatchSound()
	}
	if !g.pauseToggleReady() {
		g.preparePausePicture()
		return
	}
	g.sceneClearFrame = g.Frame
	g.clearArenaPictures()
	for id := 2; id >= 0; id-- {
		s := &g.Fighters[id].computerAnimation
		s.Idle = 0
		s.selectDirect(0)
		s.Pending = 0
		s.Position, s.Facing = g.pauseDemo.positions[id], g.pauseDemo.facing[id]
		g.publishPauseActor(id)
	}
	g.Phase, g.PhaseTicks, g.fightPrepared = Fight, 0, false
	if g.demonstrating {
		g.Phase = Attract
	}
}
