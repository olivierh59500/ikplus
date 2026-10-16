package game

// CopyPresentationState publishes a native match snapshot for scene controllers.
// Graphics, score history and title particles keep their existing ownership.
func (g *Game) CopyPresentationState(state *PresentationState) {
	flag := func(value bool) byte {
		if value {
			return 1
		}
		return 0
	}
	bcd := func(value int) byte { return byte(value/10*16 + value%10) }
	state.Frame, state.Mode = g.OriginalVBL(), g.OriginalMode()
	state.Speed, state.Fast = g.OriginalSpeed(), g.OriginalFast()
	state.KeyDelay = g.speedKeyDelay
	copy(state.Keys[0:4], g.speedKeys[:])
	state.LastRawKey = uint16(g.speedLastRaw)
	state.KeyHistory = g.speedKeyHistory
	state.FreezeLocation = g.pausePoint
	state.BootMarker = g.sourceBootMarker
	state.Music, state.Effects = flag(g.MusicEnabled()), flag(g.EffectsEnabled())
	state.Volume = g.MasterVolume()
	state.HintCount = g.hint.Count
	state.FontOutline = g.hintOutline
	state.HintTimer = g.hint.Timer
	state.HintIndex = g.hint.Index
	state.FontPosition = g.hintPosition
	state.FontOffset = g.hintOffset
	state.FontLine = g.hintLine
	state.GlyphCounter = g.hintGlyphCounter
	state.FontColor = g.hintColor
	state.Filter = uint16(flag(g.FilterLatch()))
	state.Graphics, state.Referee = g.graphicsIndex, g.refereeVariant
	state.GroundShade = g.GroundShade()
	state.German = uint16(flag(g.GermanText()))
	state.AlternateShield = uint16(flag(g.AlternateShield()))
	state.Frozen = uint16(flag(g.SourceFrozen()))
	state.GraphicsColor = g.PublishedGraphicsColor()
	state.VisibleGroundShade = g.PublishedGroundShade()
	state.Round, state.RoundCycle, state.Difficulty = bcd(g.Round), byte(g.roundCycle), g.sourceDifficulty()
	state.Seconds = bcd(g.Remaining / TickRate)
	_, subsecond, _ := originalRoundClock(g.clockTicks)
	if g.judge.active && g.Phase == Judge {
		subsecond = int(g.judge.subsecond)
	}
	state.Subsecond, state.DisplayCounter, state.Elapsed = byte(subsecond), g.displayCounter, g.displayElapsed
	state.DrawPriority = g.drawPriority
	state.PauseCooldown = g.pauseCooldown
	state.PauseSound = uint16(flag(g.pauseDemo.sound))
	copy(state.PauseIndices[0:3], g.pauseDemo.indices[:])
	copy(state.PausePositions[0:3], g.pauseDemo.positions[:])
	copy(state.PauseFacing[0:3], g.pauseDemo.facing[:])
	// Arena entry resets name controls. Input repeat and animation wraps
	// continue until the next score handoff.
	state.ResetName()
	state.InputRepeat, state.FrameWraps = g.inputRepeatCounter, g.sourceVBLWraps
	state.HumanCount = 0
	for id, f := range g.Fighters {
		state.Actors[id] = f.computerAnimation
		state.Humans[id] = flag(f.Human)
		state.HumanCount += flag(f.Human)
		state.Points[id] = byte(f.Points)
		for digit, divisor := range []int{10000, 100, 1} {
			state.ScoreDigits[id*3+digit] = bcd(f.Score / divisor % 100)
		}
		state.RoundScoreHigh[id], state.RoundScoreLow[id] = bcd(f.RoundScore/10000), bcd(f.RoundScore/100%100)
		state.Stun[id], state.Recovery[id], state.Rest[id] = byte(f.Stun), byte(f.recoveryWait), byte(f.GetUp)
		state.FighterActivity[id] = f.activityByte
		state.StarFrame[id], state.StarColor[id], state.StarX[id], state.StarY[id] = f.StarFrame, f.StarColor, f.StarX, f.StarY
		state.DroolFrame[id], state.Puff[id], state.PuffX[id] = f.DroolFrame, flag(f.Puff), f.PuffX
	}
	for id, color := range g.beltColors {
		state.BeltColors[id] = color
	}
	state.VisibleWhiteBelt, state.VisibleRedBelt, state.VisibleBlueBelt = g.beltColorsVisible[0], g.beltColorsVisible[1], g.beltColorsVisible[2]
	state.EffectColor = g.effectColor
	state.VisibleEffectColor = g.effectColorVisible
	copy(state.OwnerFlash[0:2], g.ownerFlash[:])
	background, drawing := g.SceneryBuffers()
	banks := [...]PictureBuffer{PictureFirst, PictureSecond}
	state.Background = banks[background]
	state.Drawing = banks[drawing]
	state.Displayed = banks[g.irqPictureBank]
	if !g.visibleValid {
		state.Displayed = PictureFirst
	}
	selector := 4
	if g.displayCounter&1 != 0 {
		selector = 0
	}
	state.BufferPhase = byte(selector)
	state.EraseStart = RestorationCursor{Buffer: banks[drawing]}
	state.EraseCursor = RestorationCursor{Buffer: banks[drawing], Count: uint16(g.presentationEraseRectangles(drawing))}
	if selector == 0 {
		state.CollisionRead, state.CollisionReadMirror = CoverageFirst, CoverageFirstMirrored
		state.CollisionWrite, state.CollisionWriteMirror = CoverageSecond, CoverageSecondMirrored
	} else {
		state.CollisionRead, state.CollisionReadMirror = CoverageSecond, CoverageSecondMirrored
		state.CollisionWrite, state.CollisionWriteMirror = CoverageFirst, CoverageFirstMirrored
	}

	s := g.Caption()
	state.Message, state.PreviousMessage, state.CaptionWidth = s.Message, s.Previous, s.Width
	state.CaptionVisible, state.CaptionAnchor, state.CaptionColumn, state.CaptionBudget = flag(s.Visible), s.Anchor, s.Column, s.Budget
	copy(state.Order[0:3], s.Order[:])
	copy(state.SparkleCounters[0:8], s.Counters[:])
	copy(state.SparkleX[0:8], s.X[:])
	copy(state.SparkleY[0:8], s.Y[:])
	state.BonusAmount = s.Amount
	state.CandidateOwner = g.candidateOwner
	if g.judge.active {
		copy(state.Order[0:3], g.judge.order[:])
		state.MessageTimer = g.judge.timer
		if g.judge.part == 6 || g.judge.part == 8 {
			state.HintTimer = 15
			state.HintIndex = 0
		}
	}
	if (g.Phase == BallBonus || g.Phase == BombBonus) && g.Bonus.clearTicks == 0 && !g.Bonus.live {
		state.MessageTimer = g.Bonus.introIRQTimer
	}
	if g.Phase == BonusEnd {
		state.MessageTimer = g.Bonus.resultIRQTimer
	}
	state.SetRandomState(g.random)
	phases := g.ReflectionPhases()
	copy(state.ReflectionPhases[0:21], phases[:])
	state.WaterUpdates, state.WaterSlow, state.WaterChoice = g.reflectionUpdates, g.reflectionSlow, g.reflectionSlow>>4
	state.SetAmbient(g.Ambient())
}

func (g *Game) presentationEraseRectangles(bank int) int {
	picture := g.Picture(bank)
	if !picture.Valid || g.paletteWaitFrame == g.Frame && picture.Frame != g.Frame {
		// A publication resets the next surface's restoration queue before
		// actor preparation starts.
		return 0
	}
	frames := func(frame int) int {
		if frame < 0 || frame >= len(animationRunCounts) {
			return 0
		}
		return int(animationRunCounts[frame])
	}
	count := 0
	if picture.Mode == 5 && picture.ShieldVisible && picture.Active >= 0 {
		count++
		if picture.Fighters[picture.Active].ShieldType == 1 {
			count++
		}
	}
	for id, fighter := range picture.Fighters {
		if picture.Active >= 0 && picture.Active != id {
			continue
		}
		count += frames(fighter.SpriteFrame)
	}
	if picture.Caption.Visible {
		count += frames(int(picture.Caption.RefereeFrame))
		if picture.CaptionText && picture.Caption.Previous < 40 {
			count++
		}
		for _, draw := range picture.Caption.Draw {
			if draw {
				count++
			}
		}
	}
	return count
}
