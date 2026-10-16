package game

// AmbientState retains the source scenery dispatcher at $19214. It belongs
// to the PAL simulation because its random calls share the fighters' stream.
type AmbientState struct {
	forceWords                              [4]uint16
	Mode, Selector, X, Y, Frame, StepClock  byte
	Direction, Color, Style, Repeats, Delay byte
	Counter, RippleCycle                    byte
	// These source bytes retain the full selection ($18862), its running
	// count ($18864), and the authored group value ($1885D).
	Selection, SelectionCount, SelectionData                          byte
	Initial, Visible, ForceBird, ForceFish                            bool
	ForcePac, ForcePeri                                               bool
	ScriptInitial, ScriptVisible                                      bool
	ScriptX, ScriptY, ScriptClock, PacCycle                           uint16
	PeriScript, PeriOffset, PeriPause, PeriPose, PeriFrame, PeriClock uint16
}

// ForceWord preserves the original word rather than normalizing every
// nonzero request. Keyboard commands writeFFFF; seeded callers can supply1.
func (a AmbientState) ForceWord(index int) uint16 {
	flags := [4]bool{a.ForcePac, a.ForceFish, a.ForceBird, a.ForcePeri}
	if !flags[index] {
		return 0
	}
	if a.forceWords[index] != 0 {
		return a.forceWords[index]
	}
	return 1
}

func (a *AmbientState) SetForceWord(index int, value uint16) {
	a.forceWords[index] = value
	flags := [4]*bool{&a.ForcePac, &a.ForceFish, &a.ForceBird, &a.ForcePeri}
	*flags[index] = value != 0
}

func (g *Game) Ambient() AmbientState { return g.ambient }

func (g *Game) resetAmbient() {
	g.ambientFinishWork = 0
	g.ambient = AmbientState{StepClock: nativeGameplay.Ambient.InitialClock, Delay: nativeGameplay.Ambient.InitialDelay, Initial: nativeGameplay.Ambient.Initial}
	g.ambientReady = true
}

func (g *Game) tickAmbient() {
	if !g.ambientReady {
		g.resetAmbient()
	}
	before := g.ambient
	g.ambient.Tick(g.OriginalVBL(), g.OriginalMode(), g.reflectionPhases, g.originalRandomByte)
	g.ambientFinishWork = 0
	if before.Mode == 1 && g.ambient.Mode == 0 {
		// Keep the final bird restoration after the scenery becomes idle.
		g.ambientFinishWork = 133
		if nativeGameplay.Ambient.BirdDirection[before.Selector] != 0 {
			g.ambientFinishWork += 7
		}
		if before.Visible {
			g.ambientFinishWork += 598
		}
	}
	if before.Mode == 3 && g.ambient.Mode == 0 {
		// Ending the ripple still restores its saved five-row picture and
		// completes the stop decision. Keep this work after the mode changes.
		g.ambientFinishWork = 137
		if before.Visible {
			g.ambientFinishWork += 415
		}
		if g.ambient.X != before.X {
			g.ambientFinishWork += 19
		}
	}
}

func (g *Game) RequestAmbient(bird, fish, pac, peri bool) {
	a := &g.ambient
	a.ForceBird = a.ForceBird || bird
	a.ForceFish = a.ForceFish || fish
	a.ForcePac = a.ForcePac || pac
	a.ForcePeri = a.ForcePeri || peri
}

func (a *AmbientState) KeySequence(history uint32) bool {
	switch history {
	case 0xa1979f23:
		a.SetForceWord(1, 0xffff)
		return true
	case 0xb0979320:
		a.SetForceWord(2, 0xffff)
	case 0x99929397:
		a.SetForceWord(3, 0xffff)
	default:
		if history&0xffffff == 0x999e2e {
			a.SetForceWord(0, 0xffff)
		}
	}
	return false
}

func (a *AmbientState) Tick(frame, mode byte, phases [21]byte, random func() byte) {
	a.Counter++
	switch a.Mode {
	case 0:
		if mode != 5 && mode != 7 {
			a.choose(random)
		}
	case 1:
		if a.Counter&3 == 0 {
			if nativeGameplay.Ambient.BirdDirection[a.Selector] != 0 {
				a.X--
			} else {
				a.X++
			}
			if a.X == nativeGameplay.Ambient.BirdEnd[a.Selector] {
				a.Mode, a.Visible, a.ForceBird = 0, false, false
			} else {
				a.Visible = true
			}
		}
	case 2:
		if a.Counter&3 == 0 {
			if frame&0xc0 == 0 {
				if phases[17] < 15 {
					a.Direction ^= 1
				}
			} else if a.Direction != 0 {
				a.Frame--
				if int8(a.Frame) < 0 {
					a.Mode, a.Visible = 0, false
				}
			} else {
				a.Frame++
				if int8(a.Frame) >= int8(nativeGameplay.Ambient.OutlineEnd[a.Selector]) {
					a.Direction = 1
				}
			}
		}
	case 3:
		if a.Counter&3 == 0 {
			a.Frame++
			if a.Frame >= 12 {
				a.X += 3
				a.Frame = 0
			}
			if a.X >= nativeGameplay.Ambient.RippleEnd[a.Selector] {
				a.Mode, a.Visible = 0, false
				return
			}
		}
		a.Visible = true
	case 4:
		a.tickFish(random)
	case 5:
		a.tickDrop(frame, random)
	case 6:
		a.tickPac(frame)
	case 7:
		a.tickPeri(frame, random)
	}
}

func (a *AmbientState) choose(random func() byte) {
	a.Visible, a.ScriptVisible = false, false
	if a.ForcePac || a.ForcePeri {
		a.chooseScript(random)
		return
	}
	selector := byte(24)
	if !a.ForceFish {
		selector = random() & 7
		if !a.ForceBird {
			if a.Counter&0xfd != 0 {
				return
			}
			selector = random()
			if selector >= 128 {
				a.chooseScript(random)
				return
			}
			selector &= 31
		}
	}
	a.Selection, a.SelectionCount = selector, a.SelectionCount+1
	a.SelectionData = nativeGameplay.Ambient.SelectionData[selector>>3]
	a.Mode = nativeGameplay.Ambient.Modes[selector>>3]
	a.Selector = selector & 7
	if a.Mode == 3 {
		selector = 16 + a.RippleCycle
		a.Selector = selector & 7
		a.RippleCycle = (a.RippleCycle + 1) & 7
	}
	a.X, a.Y, a.Color = nativeGameplay.Ambient.X[selector], nativeGameplay.Ambient.Y[selector], nativeGameplay.Ambient.Colors[selector]
	a.Frame, a.Direction, a.Style = 0, 0, 0
	a.Repeats, a.Initial = 1, true
	value := random()
	if value&64 == 0 {
		a.Style = (value & 3) * 4
	}
}

func (a *AmbientState) chooseScript(random func() byte) {
	switch {
	case a.ForcePac:
		a.ForcePac = false
		a.Mode = 6
	case a.ForcePeri:
		a.ForcePeri = false
		a.Mode = 7
	default:
		a.Mode = nativeGameplay.Ambient.Scripts[random()&15]
	}
	a.ScriptInitial = true
}

func (a *AmbientState) tickFish(random func() byte) {
	if a.Initial {
		a.Initial, a.ForceFish = false, false
		a.Delay = (random() & 1) + 3
		value := random()
		index := int(value & 15)
		a.X, a.Y = nativeGameplay.Ambient.FishX[index], nativeGameplay.Ambient.FishY[index]
		a.Direction, a.Repeats = value>>5&1, 0
		if value&0xc0 != 0 {
			a.Repeats = nativeGameplay.Ambient.FishRepeats[index]
		}
	}
	a.StepClock++
	if a.StepClock >= a.Delay {
		a.Frame++
		a.StepClock = 0
	}
	if a.Frame >= 20 {
		a.Repeats--
		if int8(a.Repeats) < 0 {
			a.Mode, a.Visible = 0, false
			return
		}
		a.Frame, a.StepClock = 0, 0
		a.X -= 8
		if a.Direction != 0 {
			a.X += 16
		}
		a.Visible = false
		return
	}
	if a.Frame == 6 && a.StepClock == 0 {
		a.X -= 4
		if a.Direction != 0 {
			a.X += 8
		}
	}
	a.Visible = true
}

func (a *AmbientState) tickDrop(frame byte, random func() byte) {
	if a.ScriptInitial {
		a.ScriptInitial = false
		i := int(random() & 30)
		a.ScriptX, a.ScriptY = nativeGameplay.Ambient.DropX[i/2]+32, nativeGameplay.Ambient.DropY[i/2]
	}
	i := int(random() & 30)
	a.ScriptX += nativeGameplay.Ambient.DropMotion[i/2]
	if int16(a.ScriptX) < 242 {
		a.ScriptX += 3
	}
	if frame&1 != 0 {
		a.ScriptY++
		limit := random() & 7
		if limit >= 5 {
			limit &= 3
		}
		if int16(a.ScriptY) >= int16(limit)+108 {
			a.Mode = 0
		}
	}
	a.ScriptVisible = a.Mode != 0
}

func (a *AmbientState) tickPac(frame byte) {
	i := int(a.PacCycle&3) * 2
	direction := nativeGameplay.Ambient.PacDirection[i/2]
	if a.ScriptInitial {
		a.ScriptInitial = false
		a.ScriptClock = 0
		a.ScriptX, a.ScriptY = nativeGameplay.Ambient.PacX[i/2], nativeGameplay.Ambient.PacY[i/2]
	}
	if frame&1 == 0 {
		a.ScriptClock++
		if a.ScriptClock >= 10 {
			a.ScriptClock = 0
		}
		a.ScriptX += direction
		if a.ScriptX == nativeGameplay.Ambient.PacEnd[i/2] {
			a.PacCycle = (a.PacCycle + 1) & 3
			a.Mode, a.ScriptVisible = 0, false
			return
		}
	}
	a.ScriptVisible = true
}

func (a *AmbientState) tickPeri(frame byte, random func() byte) {
	if a.ScriptInitial {
		a.ScriptInitial = false
		a.PeriOffset, a.PeriPause, a.PeriPose, a.PeriFrame, a.PeriClock = 0, 0, 0, 0, 0
		a.PeriScript = uint16(random() & 7)
		i := int(random() & 14)
		a.ScriptX, a.ScriptY = nativeGameplay.Ambient.PeriX[i/2]>>1+16, nativeGameplay.Ambient.PeriY[i/2]
	}
	if a.PeriPause != 0 {
		a.PeriPause--
	} else {
		script := nativeGameplay.Ambient.PeriScripts[a.PeriScript]
		value := uint16(script[a.PeriOffset])
		if value == 255 {
			a.Mode, a.ScriptVisible = 0, false
			return
		}
		if value >= 128 {
			a.PeriPause = 10
			if value != 254 {
				value &= 31
			}
		}
		a.PeriPose = value
		if frame&3 == 0 {
			a.PeriOffset++
		}
	}
	a.PeriClock--
	if int16(a.PeriClock) < 0 {
		a.PeriClock = 5
		a.PeriFrame++
		if a.PeriFrame >= 7 {
			a.PeriFrame = 0
		}
	}
	a.ScriptVisible = true
}
