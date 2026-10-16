package game

func randomFlag(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

// randomSample supplies named state inputs to the recovered random mixer.
func (g *Game) randomSample(sample RandomSample) (byte, bool) {
	var value uint32
	switch sample.Kind {
	case sampleSpeed:
		value = uint32(g.OriginalSpeed())
	case sampleFast:
		value = uint32(g.OriginalFast())
	case sampleMusic:
		value = randomFlag(g.MusicEnabled())
	case sampleEffects:
		value = randomFlag(g.EffectsEnabled())
	default:
		return g.randomSceneSample(sample)
	}
	return byte(value >> sample.Shift), true
}

func (g *Game) randomSceneSample(sample RandomSample) (byte, bool) {
	var value uint32
	if g.presentationState == nil {
		switch sample.Kind {
		case sampleHintCount:
			value = uint32(g.hint.Count)
		case sampleFontOutline:
			value = uint32(g.hintOutline)
		case sampleHintTimer:
			value = uint32(g.hint.Timer)
		case sampleHintIndex:
			value = uint32(g.hint.Index)
		case sampleFontPosition:
			value = uint32(g.hintPosition)
		case sampleFontOffset:
			value = uint32(g.hintOffset)
		case sampleFontLine:
			value = uint32(g.hintLine)
		case sampleGlyphCounter:
			value = uint32(g.hintGlyphCounter)
		case sampleFontColor:
			value = uint32(g.hintColor)
		case sampleFreezeLocation:
			return waitPointSample(g.pausePoint, sample.Shift), true
		case sampleBootMarker:
			value = g.sourceBootMarker
		case sampleFilter:
			value = randomFlag(g.filterLatch)
		case sampleGraphics:
			value = uint32(g.graphicsIndex)
		case sampleReferee:
			value = uint32(g.refereeVariant)
		case sampleFrozen:
			value = randomFlag(g.sourceFrozen)
		case sampleGerman:
			value = randomFlag(g.germanText)
		case sampleAlternateShield:
			value = randomFlag(g.alternateShield)
		case sampleGroundShade:
			value = uint32(g.GroundShade())
		case sampleGraphicsColor:
			value = uint32(g.PublishedGraphicsColor())
		case sampleVisibleGroundShade:
			value = uint32(g.PublishedGroundShade())
		case samplePauseCooldown:
			value = uint32(g.pauseCooldown)
		case samplePauseIndices:
			value = uint32(g.pauseDemo.indices[sample.Index])
		case samplePausePositions:
			value = uint32(g.pauseDemo.positions[sample.Index])
		case samplePauseFacing:
			value = uint32(g.pauseDemo.facing[sample.Index])
		case sampleKeyDelay:
			value = uint32(g.speedKeyDelay)
		case sampleKeys:
			value = uint32(g.speedKeys[sample.Index])
		case sampleKeyHistory:
			value = g.speedKeyHistory
		case sampleLastRawKey:
			value = uint32(g.speedLastRaw)
		case sampleJoystick:
			input := g.joystick[sample.Index]
			value = uint32(joystickNibble(input))
			if input.Fire {
				value |= 16
			}
		case sampleMessageTimer:
			switch {
			case g.Phase == BonusEnd:
				value = uint32(g.Bonus.resultIRQTimer)
			case (g.Phase == BallBonus || g.Phase == BombBonus) && g.Bonus.clearTicks == 0 && !g.Bonus.live:
				value = uint32(g.Bonus.introIRQTimer)
			case g.judge.active:
				value = uint32(g.judge.timer)
			default:
				return g.randomAudioSample(sample)
			}
		default:
			return g.randomAudioSample(sample)
		}
		return byte(value >> sample.Shift), true
	}
	return g.randomAudioSample(sample)
}

func (g *Game) randomAudioSample(sample RandomSample) (byte, bool) {
	var value uint32
	if g.audioControl != nil {
		s := &g.audioControl.State
		switch sample.Kind {
		case sampleAudioPrevious:
			value = uint32(s.Previous[sample.Index])
		case sampleAudioSelected:
			value = uint32(s.Selected)
		case sampleAudioFrame:
			value = uint32(s.Frame)
		case sampleAudioFlags:
			value = uint32(s.Flags)
		case sampleAudioActor:
			value = uint32(s.Actor)
		case sampleAudioDelay:
			value = uint32(s.Delay)
		case sampleAudioDelayed:
			value = uint32(s.Delayed)
		case sampleAudioLength:
			value = uint32(s.LengthCounter)
		case sampleAudioBusy:
			value = uint32(s.Busy)
		case sampleAudioCommands:
			value = uint32(s.CopperCount)
		case sampleAudioMainPending:
			value = uint32(s.MainPending)
		case sampleAudioActive:
			value = uint32(s.Active)
		case sampleAudioMainID:
			value = uint32(s.MainID)
		case sampleAudioID:
			value = uint32(s.ID)
		case sampleAudioObjectID:
			value = uint32(s.IRQID)
		case sampleAudioObjectPending:
			value = uint32(s.IRQPending)
		case sampleSoundCooldown:
			value = uint32(g.soundCooldown[sample.Index])
		case sampleSoundHits:
			value = uint32(g.soundHits[sample.Index])
		case sampleSoundDown:
			value = uint32(g.soundDown[sample.Index])
		default:
			if g.audioEngine != nil {
				switch sample.Kind {
				case sampleAudioVoices:
					return g.audioEngine.Music.RandomByte(int(sample.Index)), true
				case sampleAudioPitch:
					return byte(g.audioEngine.PitchScratch() >> sample.Shift), true
				case sampleVolume:
					return byte(g.MasterVolume() >> sample.Shift), true
				case sampleAudioSubstep:
					return g.audioEngine.Substep, true
				}
			}
			return g.randomRetainedSample(sample)
		}
		return byte(value >> sample.Shift), true
	}
	return g.randomRetainedSample(sample)
}

func (g *Game) randomRetainedSample(sample RandomSample) (byte, bool) {
	if g.presentationState != nil {
		return g.presentationState.SampleRandom(sample)
	}
	var value uint32
	switch sample.Kind {
	case sampleInputRepeat:
		value = uint32(g.inputRepeatCounter)
	case sampleFrameWraps:
		value = uint32(g.sourceVBLWraps)
	case sampleCollisionEnabled, sampleSecondaryCollisionEnabled:
		if (g.Phase == BallBonus || g.Phase == BombBonus || g.Phase == BonusEnd) && g.Bonus.clearTicks == 0 {
			value = 1
		}
	case sampleCollisionRead, sampleCollisionReadMirror, sampleCollisionWrite, sampleCollisionWriteMirror:
		index := int(sample.Kind - sampleCollisionRead)
		phase := 0
		if g.displayCounter&1 == 0 {
			phase = 1
		}
		return collisionEntropy[phase][index][3-int(sample.Shift)/8], true
	default:
		if g.retainedPresentation == nil {
			return 0, false
		}
		switch sample.Kind {
		case sampleScoreHigh, sampleScoreLow, sampleScoreCandidate, sampleNameFirst, sampleNameSecond, sampleNameThird, sampleTitlePalette, sampleParticles:
			return g.retainedPresentation.state.SampleRandom(sample)
		default:
			return 0, false
		}
	}
	return byte(value >> sample.Shift), true
}
