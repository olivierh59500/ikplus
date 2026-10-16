package game

// RandomSample identifies one semantic input to the deterministic mixer.
// The static lookup vocabulary is recovered asset data, not a memory map.
type RandomSample struct {
	Kind  randomSampleKind
	Index uint16
	Shift uint8
}
type randomSampleKind uint16

const (
	sampleFrame                     randomSampleKind = 0
	sampleDisplayCounter            randomSampleKind = 1
	sampleHumanCount                randomSampleKind = 2
	sampleHumanMask                 randomSampleKind = 3
	sampleKeyDelay                  randomSampleKind = 4
	sampleSpeed                     randomSampleKind = 5
	sampleMusic                     randomSampleKind = 6
	sampleEffects                   randomSampleKind = 7
	sampleFast                      randomSampleKind = 8
	sampleGraphics                  randomSampleKind = 9
	sampleBufferPhase               randomSampleKind = 10
	sampleSoundStep                 randomSampleKind = 11
	sampleReferee                   randomSampleKind = 12
	sampleDigitBlank                randomSampleKind = 13
	sampleDigitValue                randomSampleKind = 14
	sampleRankDigits                randomSampleKind = 15
	sampleRankIndex                 randomSampleKind = 16
	sampleSortRemaining             randomSampleKind = 17
	sampleMode                      randomSampleKind = 18
	sampleOwnerCursor               randomSampleKind = 19
	sampleActorCursor               randomSampleKind = 20
	sampleRowCursor                 randomSampleKind = 21
	sampleMessage                   randomSampleKind = 22
	samplePreviousMessage           randomSampleKind = 23
	sampleRound                     randomSampleKind = 24
	sampleMessageTimer              randomSampleKind = 25
	sampleSeconds                   randomSampleKind = 26
	sampleMotionTimer               randomSampleKind = 27
	sampleDrawPriority              randomSampleKind = 28
	sampleScoreForeground           randomSampleKind = 29
	sampleElapsed                   randomSampleKind = 30
	sampleDifficulty                randomSampleKind = 31
	sampleSubsecond                 randomSampleKind = 32
	sampleBonusAmount               randomSampleKind = 33
	sampleRoundCycle                randomSampleKind = 34
	sampleLastActor                 randomSampleKind = 35
	sampleCandidateOwner            randomSampleKind = 36
	sampleCandidateIndex            randomSampleKind = 37
	sampleInputRepeat               randomSampleKind = 38
	sampleNameGlyph                 randomSampleKind = 39
	sampleFireReleased              randomSampleKind = 40
	samplePreviousFire              randomSampleKind = 41
	sampleDirections                randomSampleKind = 42
	sampleNameColumn                randomSampleKind = 43
	sampleFrameWraps                randomSampleKind = 44
	sampleNameReserved              randomSampleKind = 45
	sampleCaptionWidth              randomSampleKind = 46
	sampleCaptionVisible            randomSampleKind = 47
	sampleCaptionAnchor             randomSampleKind = 48
	sampleCaptionColumn             randomSampleKind = 49
	sampleCaptionBudget             randomSampleKind = 50
	samplePauseCooldown             randomSampleKind = 51
	sampleScoresVisible             randomSampleKind = 52
	sampleCandidateRow              randomSampleKind = 53
	sampleFirstScore                randomSampleKind = 54
	sampleDefence                   randomSampleKind = 55
	sampleActivity                  randomSampleKind = 56
	sampleTargetPolicy              randomSampleKind = 57
	sampleClockGlyph                randomSampleKind = 58
	sampleFontColor                 randomSampleKind = 59
	sampleJoinPending               randomSampleKind = 60
	sampleHumans                    randomSampleKind = 61
	sampleOrder                     randomSampleKind = 62
	sampleRest                      randomSampleKind = 63
	sampleSoundCooldown             randomSampleKind = 64
	samplePauseIndices              randomSampleKind = 65
	samplePoints                    randomSampleKind = 66
	sampleScoreDigits               randomSampleKind = 67
	sampleSoundDown                 randomSampleKind = 68
	sampleSoundHits                 randomSampleKind = 69
	sampleFighterActivity           randomSampleKind = 70
	sampleRoundScoreHigh            randomSampleKind = 71
	sampleRoundScoreLow             randomSampleKind = 72
	sampleJoystick                  randomSampleKind = 73
	sampleSparkleCounters           randomSampleKind = 74
	sampleSparkleX                  randomSampleKind = 75
	sampleSparkleY                  randomSampleKind = 76
	sampleStun                      randomSampleKind = 77
	sampleRecovery                  randomSampleKind = 78
	sampleDroolFrame                randomSampleKind = 79
	sampleStarFrame                 randomSampleKind = 80
	sampleStarColor                 randomSampleKind = 81
	sampleStarX                     randomSampleKind = 82
	sampleStarY                     randomSampleKind = 83
	samplePuff                      randomSampleKind = 84
	samplePuffX                     randomSampleKind = 85
	samplePausePositions            randomSampleKind = 86
	samplePauseFacing               randomSampleKind = 87
	sampleOwnerFlash                randomSampleKind = 88
	sampleInitials                  randomSampleKind = 89
	sampleKeys                      randomSampleKind = 90
	sampleScoreHigh                 randomSampleKind = 91
	sampleScoreLow                  randomSampleKind = 92
	sampleScoreCandidate            randomSampleKind = 93
	sampleNameFirst                 randomSampleKind = 94
	sampleNameSecond                randomSampleKind = 95
	sampleNameThird                 randomSampleKind = 96
	sampleBlackScreen               randomSampleKind = 97
	sampleGroundShade               randomSampleKind = 98
	sampleEffectColor               randomSampleKind = 99
	sampleVolume                    randomSampleKind = 100
	sampleHintCount                 randomSampleKind = 101
	sampleTitleLanguage             randomSampleKind = 102
	sampleStartupReserved           randomSampleKind = 103
	sampleFontOutline               randomSampleKind = 104
	sampleFilter                    randomSampleKind = 105
	sampleGerman                    randomSampleKind = 106
	sampleFontPosition              randomSampleKind = 107
	sampleRandomCounter             randomSampleKind = 108
	sampleGlyphCounter              randomSampleKind = 109
	sampleBorderFrame               randomSampleKind = 110
	sampleBorderOffset              randomSampleKind = 111
	sampleBorderEnabled             randomSampleKind = 112
	sampleAlternateShield           randomSampleKind = 113
	samplePauseSound                randomSampleKind = 114
	sampleFrozen                    randomSampleKind = 115
	sampleFontOffset                randomSampleKind = 116
	sampleHintTimer                 randomSampleKind = 117
	sampleHintIndex                 randomSampleKind = 118
	sampleFontLine                  randomSampleKind = 119
	sampleFadeDivider               randomSampleKind = 120
	sampleTitleStage                randomSampleKind = 121
	sampleTitleAdvance              randomSampleKind = 122
	sampleTitleEnding               randomSampleKind = 123
	sampleTitleTimer                randomSampleKind = 124
	sampleTitleExitTimer            randomSampleKind = 125
	sampleTitleActorsEnabled        randomSampleKind = 126
	sampleTitlePicture              randomSampleKind = 127
	sampleColorSequence             randomSampleKind = 128
	sampleColorRotations            randomSampleKind = 129
	sampleColorRotationTimer        randomSampleKind = 130
	sampleColorTimer                randomSampleKind = 131
	sampleLastRawKey                randomSampleKind = 132
	sampleRainbowY                  randomSampleKind = 133
	sampleRainbowColor              randomSampleKind = 134
	sampleParticleOrigin            randomSampleKind = 135
	sampleLetterCount               randomSampleKind = 136
	sampleLetterIndex               randomSampleKind = 137
	sampleLetterStep                randomSampleKind = 138
	sampleLetterMotion              randomSampleKind = 139
	sampleLetterOperation           randomSampleKind = 140
	sampleLetterComplete            randomSampleKind = 141
	sampleScrollCount               randomSampleKind = 142
	sampleScrollDirection           randomSampleKind = 143
	sampleRandomMultiplier          randomSampleKind = 144
	sampleRandomAddition            randomSampleKind = 145
	sampleRandomXor                 randomSampleKind = 146
	sampleCollisionEnabled          randomSampleKind = 147
	sampleSecondaryCollisionEnabled randomSampleKind = 148
	sampleGraphicsColor             randomSampleKind = 149
	sampleVisibleEffectColor        randomSampleKind = 150
	sampleVisibleBlueBelt           randomSampleKind = 151
	sampleVisibleWhiteBelt          randomSampleKind = 152
	sampleVisibleRedBelt            randomSampleKind = 153
	sampleVisibleGroundShade        randomSampleKind = 154
	sampleBorderCell                randomSampleKind = 155
	sampleBeltColors                randomSampleKind = 156
	sampleTitlePalette              randomSampleKind = 157
	sampleBackground                randomSampleKind = 158
	sampleDrawing                   randomSampleKind = 159
	sampleEraseStart                randomSampleKind = 160
	sampleEraseCursor               randomSampleKind = 161
	sampleDisplayed                 randomSampleKind = 162
	sampleFreezeLocation            randomSampleKind = 163
	sampleBootMarker                randomSampleKind = 164
	sampleKeyHistory                randomSampleKind = 165
	sampleCollisionRead             randomSampleKind = 166
	sampleCollisionReadMirror       randomSampleKind = 167
	sampleCollisionWrite            randomSampleKind = 168
	sampleCollisionWriteMirror      randomSampleKind = 169
	sampleActorsPosition            randomSampleKind = 170
	sampleActorsFacing              randomSampleKind = 171
	sampleActorsFrame               randomSampleKind = 172
	sampleActorsPending             randomSampleKind = 173
	sampleActorsCurrent             randomSampleKind = 174
	sampleActorsNext                randomSampleKind = 175
	sampleActorsAction              randomSampleKind = 176
	sampleActorsFresh               randomSampleKind = 177
	sampleActorsRollback            randomSampleKind = 178
	sampleActorsCancelable          randomSampleKind = 179
	sampleActorsHeld                randomSampleKind = 180
	sampleActorsIdle                randomSampleKind = 181
	sampleActorsFallen              randomSampleKind = 182
	sampleActorsFlags               randomSampleKind = 183
	sampleActorsPreviousFlags       randomSampleKind = 184
	sampleActorsMirror              randomSampleKind = 185
	sampleActorsThreat              randomSampleKind = 186
	sampleAudioPrevious             randomSampleKind = 187
	sampleAudioSelected             randomSampleKind = 188
	sampleAudioFrame                randomSampleKind = 189
	sampleAudioFlags                randomSampleKind = 190
	sampleAudioActor                randomSampleKind = 191
	sampleAudioDelay                randomSampleKind = 192
	sampleAudioDelayed              randomSampleKind = 193
	sampleAudioLength               randomSampleKind = 194
	sampleAudioBusy                 randomSampleKind = 195
	sampleAudioCommands             randomSampleKind = 196
	sampleAudioMainPending          randomSampleKind = 197
	sampleAudioActive               randomSampleKind = 198
	sampleAudioMainID               randomSampleKind = 199
	sampleAudioID                   randomSampleKind = 200
	sampleAudioObjectID             randomSampleKind = 201
	sampleAudioObjectPending        randomSampleKind = 202
	sampleAudioSubstep              randomSampleKind = 203
	sampleAudioPitch                randomSampleKind = 204
	sampleAudioVoices               randomSampleKind = 205
	sampleParticles                 randomSampleKind = 206
	sampleRainbowLine0              randomSampleKind = 207
	sampleRainbowLine1              randomSampleKind = 208
	sampleRainbowLine2              randomSampleKind = 209
	sampleRainbowLine3              randomSampleKind = 210
	sampleRainbowLine4              randomSampleKind = 211
	sampleRainbowLine5              randomSampleKind = 212
	sampleRainbowLine6              randomSampleKind = 213
	sampleRainbowLine7              randomSampleKind = 214
	sampleRainbowLine8              randomSampleKind = 215
)

var randomSampleDefinitions = loadRandomAsset("original/random-inputs.bin", 23145)

func randomSampleAt(index int) RandomSample {
	data := randomSampleDefinitions[index*5:]
	return RandomSample{randomSampleKind(uint16(data[0]) | uint16(data[1])<<8), uint16(data[2]) | uint16(data[3])<<8, data[4]}
}

// SampleRandom reads ordinary Go state for a requested random input.
func (s *PresentationState) SampleRandom(sample RandomSample) (byte, bool) {
	var value uint32
	switch sample.Kind {
	case sampleFrame:
		value = uint32(s.Frame)
	case sampleDisplayCounter:
		value = uint32(s.DisplayCounter)
	case sampleHumanCount:
		value = uint32(s.HumanCount)
	case sampleHumanMask:
		value = uint32(s.HumanMask)
	case sampleKeyDelay:
		value = uint32(s.KeyDelay)
	case sampleSpeed:
		value = uint32(s.Speed)
	case sampleMusic:
		value = uint32(s.Music)
	case sampleEffects:
		value = uint32(s.Effects)
	case sampleFast:
		value = uint32(s.Fast)
	case sampleGraphics:
		value = uint32(s.Graphics)
	case sampleBufferPhase:
		value = uint32(s.BufferPhase)
	case sampleSoundStep:
		value = uint32(s.SoundStep)
	case sampleReferee:
		value = uint32(s.Referee)
	case sampleDigitBlank:
		value = uint32(s.DigitBlank)
	case sampleDigitValue:
		value = uint32(s.DigitValue)
	case sampleRankDigits:
		value = uint32(s.RankDigits)
	case sampleRankIndex:
		value = uint32(s.RankIndex)
	case sampleSortRemaining:
		value = uint32(s.SortRemaining)
	case sampleMode:
		value = uint32(s.Mode)
	case sampleOwnerCursor:
		value = uint32(s.OwnerCursor)
	case sampleActorCursor:
		value = uint32(s.ActorCursor)
	case sampleRowCursor:
		value = uint32(s.RowCursor)
	case sampleMessage:
		value = uint32(s.Message)
	case samplePreviousMessage:
		value = uint32(s.PreviousMessage)
	case sampleRound:
		value = uint32(s.Round)
	case sampleMessageTimer:
		value = uint32(s.MessageTimer)
	case sampleSeconds:
		value = uint32(s.Seconds)
	case sampleMotionTimer:
		value = uint32(s.MotionTimer)
	case sampleDrawPriority:
		value = uint32(s.DrawPriority)
	case sampleScoreForeground:
		value = uint32(s.ScoreForeground)
	case sampleElapsed:
		value = uint32(s.Elapsed)
	case sampleDifficulty:
		value = uint32(s.Difficulty)
	case sampleSubsecond:
		value = uint32(s.Subsecond)
	case sampleBonusAmount:
		value = uint32(s.BonusAmount)
	case sampleRoundCycle:
		value = uint32(s.RoundCycle)
	case sampleLastActor:
		value = uint32(s.LastActor)
	case sampleCandidateOwner:
		value = uint32(s.CandidateOwner)
	case sampleCandidateIndex:
		value = uint32(s.CandidateIndex)
	case sampleInputRepeat:
		value = uint32(s.InputRepeat)
	case sampleNameGlyph:
		value = uint32(s.NameGlyph)
	case sampleFireReleased:
		value = uint32(s.FireReleased)
	case samplePreviousFire:
		value = uint32(s.PreviousFire)
	case sampleDirections:
		value = uint32(s.Directions)
	case sampleNameColumn:
		value = uint32(s.NameColumn)
	case sampleFrameWraps:
		value = uint32(s.FrameWraps)
	case sampleNameReserved:
		value = uint32(s.NameReserved)
	case sampleCaptionWidth:
		value = uint32(s.CaptionWidth)
	case sampleCaptionVisible:
		value = uint32(s.CaptionVisible)
	case sampleCaptionAnchor:
		value = uint32(s.CaptionAnchor)
	case sampleCaptionColumn:
		value = uint32(s.CaptionColumn)
	case sampleCaptionBudget:
		value = uint32(s.CaptionBudget)
	case samplePauseCooldown:
		value = uint32(s.PauseCooldown)
	case sampleScoresVisible:
		value = uint32(s.ScoresVisible)
	case sampleCandidateRow:
		value = uint32(s.CandidateRow)
	case sampleFirstScore:
		value = uint32(s.FirstScore)
	case sampleDefence:
		value = uint32(s.Defence)
	case sampleActivity:
		value = uint32(s.Activity)
	case sampleTargetPolicy:
		value = uint32(s.TargetPolicy)
	case sampleClockGlyph:
		value = uint32(s.ClockGlyph)
	case sampleFontColor:
		value = uint32(s.FontColor)
	case sampleJoinPending:
		value = uint32(s.JoinPending)
	case sampleHumans:
		value = uint32(s.Humans[sample.Index])
	case sampleOrder:
		value = uint32(s.Order[sample.Index])
	case sampleRest:
		value = uint32(s.Rest[sample.Index])
	case sampleSoundCooldown:
		value = uint32(s.SoundCooldown[sample.Index])
	case samplePauseIndices:
		value = uint32(s.PauseIndices[sample.Index])
	case samplePoints:
		value = uint32(s.Points[sample.Index])
	case sampleScoreDigits:
		value = uint32(s.ScoreDigits[sample.Index])
	case sampleSoundDown:
		value = uint32(s.SoundDown[sample.Index])
	case sampleSoundHits:
		value = uint32(s.SoundHits[sample.Index])
	case sampleFighterActivity:
		value = uint32(s.FighterActivity[sample.Index])
	case sampleRoundScoreHigh:
		value = uint32(s.RoundScoreHigh[sample.Index])
	case sampleRoundScoreLow:
		value = uint32(s.RoundScoreLow[sample.Index])
	case sampleJoystick:
		value = uint32(s.Joystick[sample.Index])
	case sampleSparkleCounters:
		value = uint32(s.SparkleCounters[sample.Index])
	case sampleSparkleX:
		value = uint32(s.SparkleX[sample.Index])
	case sampleSparkleY:
		value = uint32(s.SparkleY[sample.Index])
	case sampleStun:
		value = uint32(s.Stun[sample.Index])
	case sampleRecovery:
		value = uint32(s.Recovery[sample.Index])
	case sampleDroolFrame:
		value = uint32(s.DroolFrame[sample.Index])
	case sampleStarFrame:
		value = uint32(s.StarFrame[sample.Index])
	case sampleStarColor:
		value = uint32(s.StarColor[sample.Index])
	case sampleStarX:
		value = uint32(s.StarX[sample.Index])
	case sampleStarY:
		value = uint32(s.StarY[sample.Index])
	case samplePuff:
		value = uint32(s.Puff[sample.Index])
	case samplePuffX:
		value = uint32(s.PuffX[sample.Index])
	case samplePausePositions:
		value = uint32(s.PausePositions[sample.Index])
	case samplePauseFacing:
		value = uint32(s.PauseFacing[sample.Index])
	case sampleOwnerFlash:
		value = uint32(s.OwnerFlash[sample.Index])
	case sampleInitials:
		value = uint32(s.Initials[sample.Index])
	case sampleKeys:
		value = uint32(s.Keys[sample.Index])
	case sampleScoreHigh:
		value = uint32(s.ScoreHigh[sample.Index])
	case sampleScoreLow:
		value = uint32(s.ScoreLow[sample.Index])
	case sampleScoreCandidate:
		value = uint32(s.ScoreCandidate[sample.Index])
	case sampleNameFirst:
		value = uint32(s.NameFirst[sample.Index])
	case sampleNameSecond:
		value = uint32(s.NameSecond[sample.Index])
	case sampleNameThird:
		value = uint32(s.NameThird[sample.Index])
	case sampleBlackScreen:
		value = uint32(s.BlackScreen)
	case sampleGroundShade:
		value = uint32(s.GroundShade)
	case sampleEffectColor:
		value = uint32(s.EffectColor)
	case sampleVolume:
		value = uint32(s.Volume)
	case sampleHintCount:
		value = uint32(s.HintCount)
	case sampleTitleLanguage:
		value = uint32(s.TitleLanguage)
	case sampleStartupReserved:
		value = uint32(s.StartupReserved)
	case sampleFontOutline:
		value = uint32(s.FontOutline)
	case sampleFilter:
		value = uint32(s.Filter)
	case sampleGerman:
		value = uint32(s.German)
	case sampleFontPosition:
		value = uint32(s.FontPosition)
	case sampleRandomCounter:
		value = uint32(s.RandomCounter)
	case sampleGlyphCounter:
		value = uint32(s.GlyphCounter)
	case sampleBorderFrame:
		value = uint32(s.BorderFrame)
	case sampleBorderOffset:
		value = uint32(s.BorderOffset)
	case sampleBorderEnabled:
		value = uint32(s.BorderEnabled)
	case sampleAlternateShield:
		value = uint32(s.AlternateShield)
	case samplePauseSound:
		value = uint32(s.PauseSound)
	case sampleFrozen:
		value = uint32(s.Frozen)
	case sampleFontOffset:
		value = uint32(s.FontOffset)
	case sampleHintTimer:
		value = uint32(s.HintTimer)
	case sampleHintIndex:
		value = uint32(s.HintIndex)
	case sampleFontLine:
		value = uint32(s.FontLine)
	case sampleFadeDivider:
		value = uint32(s.FadeDivider)
	case sampleTitleStage:
		value = uint32(s.TitleStage)
	case sampleTitleAdvance:
		value = uint32(s.TitleAdvance)
	case sampleTitleEnding:
		value = uint32(s.TitleEnding)
	case sampleTitleTimer:
		value = uint32(s.TitleTimer)
	case sampleTitleExitTimer:
		value = uint32(s.TitleExitTimer)
	case sampleTitleActorsEnabled:
		value = uint32(s.TitleActorsEnabled)
	case sampleTitlePicture:
		value = uint32(s.TitlePicture)
	case sampleColorSequence:
		value = uint32(s.ColorSequence)
	case sampleColorRotations:
		value = uint32(s.ColorRotations)
	case sampleColorRotationTimer:
		value = uint32(s.ColorRotationTimer)
	case sampleColorTimer:
		value = uint32(s.ColorTimer)
	case sampleLastRawKey:
		value = uint32(s.LastRawKey)
	case sampleRainbowY:
		value = uint32(s.RainbowY)
	case sampleRainbowColor:
		value = uint32(s.RainbowColor)
	case sampleParticleOrigin:
		value = uint32(s.ParticleOrigin)
	case sampleLetterCount:
		value = uint32(s.LetterCount)
	case sampleLetterIndex:
		value = uint32(s.LetterIndex)
	case sampleLetterStep:
		value = uint32(s.LetterStep)
	case sampleLetterMotion:
		value = uint32(s.LetterMotion)
	case sampleLetterOperation:
		value = uint32(s.LetterOperation)
	case sampleLetterComplete:
		value = uint32(s.LetterComplete)
	case sampleScrollCount:
		value = uint32(s.ScrollCount)
	case sampleScrollDirection:
		value = uint32(s.ScrollDirection)
	case sampleRandomMultiplier:
		value = uint32(s.RandomMultiplier)
	case sampleRandomAddition:
		value = uint32(s.RandomAddition)
	case sampleRandomXor:
		value = uint32(s.RandomXor)
	case sampleCollisionEnabled:
		value = uint32(s.CollisionEnabled)
	case sampleSecondaryCollisionEnabled:
		value = uint32(s.SecondaryCollisionEnabled)
	case sampleGraphicsColor:
		value = uint32(s.GraphicsColor)
	case sampleVisibleEffectColor:
		value = uint32(s.VisibleEffectColor)
	case sampleVisibleBlueBelt:
		value = uint32(s.VisibleBlueBelt)
	case sampleVisibleWhiteBelt:
		value = uint32(s.VisibleWhiteBelt)
	case sampleVisibleRedBelt:
		value = uint32(s.VisibleRedBelt)
	case sampleVisibleGroundShade:
		value = uint32(s.VisibleGroundShade)
	case sampleBorderCell:
		value = uint32(s.BorderCell)
	case sampleBeltColors:
		value = uint32(s.BeltColors[sample.Index])
	case sampleTitlePalette:
		value = uint32(s.TitlePalette[sample.Index])
	case sampleBackground:
		return pictureBufferSample[s.Background][3-int(sample.Shift)/8], true
	case sampleDrawing:
		return pictureBufferSample[s.Drawing][3-int(sample.Shift)/8], true
	case sampleEraseStart:
		return restorationSample(s.EraseStart, sample.Shift), true
	case sampleEraseCursor:
		return restorationSample(s.EraseCursor, sample.Shift), true
	case sampleDisplayed:
		return pictureBufferSample[s.Displayed][3-int(sample.Shift)/8], true
	case sampleFreezeLocation:
		return waitPointSample(s.FreezeLocation, sample.Shift), true
	case sampleBootMarker:
		value = uint32(s.BootMarker)
	case sampleKeyHistory:
		value = uint32(s.KeyHistory)
	case sampleCollisionRead:
		return coveragePlaneSample[s.CollisionRead][3-int(sample.Shift)/8], true
	case sampleCollisionReadMirror:
		return coveragePlaneSample[s.CollisionReadMirror][3-int(sample.Shift)/8], true
	case sampleCollisionWrite:
		return coveragePlaneSample[s.CollisionWrite][3-int(sample.Shift)/8], true
	case sampleCollisionWriteMirror:
		return coveragePlaneSample[s.CollisionWriteMirror][3-int(sample.Shift)/8], true
	case sampleActorsPosition:
		value = uint32(s.Actors[sample.Index].Position)
	case sampleActorsFacing:
		value = uint32(s.Actors[sample.Index].Facing)
	case sampleActorsFrame:
		value = uint32(s.Actors[sample.Index].Frame)
	case sampleActorsPending:
		value = uint32(s.Actors[sample.Index].Pending)
	case sampleActorsCurrent:
		value = uint32(s.Actors[sample.Index].Current)
	case sampleActorsNext:
		value = uint32(s.Actors[sample.Index].Next)
	case sampleActorsAction:
		value = uint32(s.Actors[sample.Index].Action)
	case sampleActorsFresh:
		value = uint32(s.Actors[sample.Index].Fresh)
	case sampleActorsRollback:
		value = uint32(s.Actors[sample.Index].Rollback)
	case sampleActorsCancelable:
		value = uint32(s.Actors[sample.Index].Cancelable)
	case sampleActorsHeld:
		value = uint32(s.Actors[sample.Index].Held)
	case sampleActorsIdle:
		value = uint32(s.Actors[sample.Index].Idle)
	case sampleActorsFallen:
		value = uint32(s.Actors[sample.Index].Fallen)
	case sampleActorsFlags:
		value = uint32(s.Actors[sample.Index].Flags)
	case sampleActorsPreviousFlags:
		value = uint32(s.Actors[sample.Index].PreviousFlags)
	case sampleActorsMirror:
		value = uint32(s.Actors[sample.Index].Mirror)
	case sampleActorsThreat:
		value = uint32(s.Actors[sample.Index].Threat)
	case sampleParticles:
		p := s.Particles[sample.Index/6]
		value = uint32([6]uint16{p.X, p.Y, p.VX, p.VY, p.PreviousX, p.PreviousY}[sample.Index%6])
	case sampleRainbowLine0:
		value = uint32(s.RainbowLines[0])
	case sampleRainbowLine1:
		value = uint32(s.RainbowLines[1])
	case sampleRainbowLine2:
		value = uint32(s.RainbowLines[2])
	case sampleRainbowLine3:
		value = uint32(s.RainbowLines[3])
	case sampleRainbowLine4:
		value = uint32(s.RainbowLines[4])
	case sampleRainbowLine5:
		value = uint32(s.RainbowLines[5])
	case sampleRainbowLine6:
		value = uint32(s.RainbowLines[6])
	case sampleRainbowLine7:
		value = uint32(s.RainbowLines[7])
	case sampleRainbowLine8:
		value = uint32(s.RainbowLines[8])
	default:
		return 0, false
	}
	return byte(value >> sample.Shift), true
}
