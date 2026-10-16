package game

// NewPresentationState returns the authored defaults as native values.
func NewPresentationState() PresentationState {
	var s PresentationState
	s.AmbientStepClock = 1
	s.AmbientDelay = 1
	s.AmbientInitial = 1
	s.GraphicsColor = 3176
	s.VisibleEffectColor = 15
	s.VisibleBlueBelt = 3264
	s.VisibleWhiteBelt = 3072
	s.VisibleRedBelt = 3084
	s.VisibleGroundShade = 546
	return s
}

// ResetArena clears the controllers owned by one match, preserving settings,
// the high-score table, title artwork and scenery animation.
func (s *PresentationState) ResetArena() {
	s.DisplayCounter = 0
	s.DigitBlank = 0
	s.DigitValue = 0
	s.RankDigits = 0
	s.RankIndex = 0
	s.SortRemaining = 0
	s.Mode = 0
	s.OwnerCursor = 0
	s.ActorCursor = 0
	s.RowCursor = 0
	s.Message = 0
	s.PreviousMessage = 0
	s.Round = 0
	s.MessageTimer = 0
	s.Seconds = 0
	s.MotionTimer = 0
	s.DrawPriority = 0
	s.ScoreForeground = 0
	s.Elapsed = 0
	s.Difficulty = 0
	s.Subsecond = 0
	s.BonusAmount = 0
	s.RoundCycle = 0
	s.LastActor = 0
	s.CandidateOwner = 0
	s.CandidateIndex = 0
	s.InputRepeat = 0
	s.NameGlyph = 0
	s.FireReleased = 0
	s.PreviousFire = 0
	s.Directions = 0
	s.NameColumn = 0
	s.FrameWraps = 0
	s.NameReserved = 0
	s.CaptionWidth = 0
	s.CaptionVisible = 0
	s.CaptionAnchor = 0
	s.CaptionColumn = 0
	s.CaptionBudget = 0
	s.PauseCooldown = 0
	s.ScoresVisible = 0
	s.CandidateRow = 0
	s.FirstScore = 0
	s.Defence = 0
	s.Activity = 0
	s.TargetPolicy = 0
	s.ClockGlyph = 0
	s.FontColor = 0
	s.JoinPending = 0
	clear(s.Order[:])
	clear(s.Rest[:])
	clear(s.SoundCooldown[:])
	clear(s.PauseIndices[:])
	clear(s.Points[:])
	clear(s.ScoreDigits[:])
	clear(s.SoundDown[:])
	clear(s.SoundHits[:])
	clear(s.FighterActivity[:])
	clear(s.RoundScoreHigh[:])
	clear(s.RoundScoreLow[:])
	clear(s.Joystick[:])
	clear(s.SparkleCounters[:])
	clear(s.SparkleX[:])
	clear(s.SparkleY[:])
	clear(s.Stun[:])
	clear(s.Recovery[:])
	clear(s.DroolFrame[:])
	clear(s.StarFrame[:])
	clear(s.StarColor[:])
	clear(s.StarX[:])
	clear(s.StarY[:])
	clear(s.Puff[:])
	clear(s.PuffX[:])
	clear(s.PausePositions[:])
	clear(s.PauseFacing[:])
	clear(s.OwnerFlash[:])
	clear(s.Initials[:])
	clear(s.Keys[:])
	s.FontPosition = 0
	s.RandomCounter = 0
	s.GlyphCounter = 0
	s.BorderFrame = 0
	s.BorderOffset = 0
	s.BorderEnabled = 0
	s.AlternateShield = 0
	s.PauseSound = 0
	s.Frozen = 0
	s.FontOffset = 0
	s.HintTimer = 0
	s.HintIndex = 0
	s.FontLine = 0
	s.FadeDivider = 0
	s.TitleStage = 0
	s.TitleAdvance = 0
	s.TitleEnding = 0
	s.TitleTimer = 0
	s.TitleExitTimer = 0
	s.TitleActorsEnabled = 0
	s.TitlePicture = 0
	s.ColorSequence = 0
	s.ColorRotations = 0
	s.ColorRotationTimer = 0
	s.ColorTimer = 0
	s.LastRawKey = 0
	s.KeyHistory = 0
	s.Actors = [3]TitleActorState{}
}

func (s *PresentationState) ResetName() {
	s.CandidateOwner = 0
	s.CandidateIndex = 0
	s.InputRepeat = 0
	s.NameGlyph = 0
	s.FireReleased = 0
	s.PreviousFire = 0
	s.Directions = 0
	s.NameColumn = 0
	s.FrameWraps = 0
	s.NameReserved = 0
}

func (s *PresentationState) SetRandomState(value uint64) {
	s.RandomCounter, s.RandomMultiplier = uint16(value>>48), uint16(value>>32)
	s.RandomAddition, s.RandomXor = uint16(value>>16), uint16(value)
}

func (s *PresentationState) RandomState() uint64 {
	return uint64(s.RandomCounter)<<48 | uint64(s.RandomMultiplier)<<32 | uint64(s.RandomAddition)<<16 | uint64(s.RandomXor)
}
