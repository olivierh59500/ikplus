package game

// PresentationState owns the title, score entry and scene-transition state.
// It is shared as ordinary Go values; it is never a memory image.
type PresentationState struct {
	Particles                 [300]TitleParticle
	RainbowLines              [9]uint16
	Actors                    [3]TitleActorState
	Frame                     byte
	DisplayCounter            byte
	HumanCount                byte
	HumanMask                 byte
	KeyDelay                  byte
	Speed                     byte
	Music                     byte
	Effects                   byte
	Fast                      byte
	Graphics                  byte
	BufferPhase               byte
	SoundStep                 byte
	Referee                   byte
	DigitBlank                byte
	DigitValue                byte
	RankDigits                byte
	RankIndex                 byte
	SortRemaining             byte
	Mode                      byte
	OwnerCursor               byte
	ActorCursor               byte
	RowCursor                 byte
	Message                   byte
	PreviousMessage           byte
	Round                     byte
	MessageTimer              byte
	Seconds                   byte
	MotionTimer               byte
	DrawPriority              byte
	ScoreForeground           byte
	Elapsed                   byte
	Difficulty                byte
	Subsecond                 byte
	BonusAmount               byte
	RoundCycle                byte
	LastActor                 byte
	CandidateOwner            byte
	CandidateIndex            byte
	InputRepeat               byte
	NameGlyph                 byte
	FireReleased              byte
	PreviousFire              byte
	Directions                byte
	NameColumn                byte
	FrameWraps                byte
	NameReserved              byte
	CaptionWidth              byte
	CaptionVisible            byte
	CaptionAnchor             byte
	CaptionColumn             byte
	CaptionBudget             byte
	PauseCooldown             byte
	ScoresVisible             byte
	CandidateRow              byte
	FirstScore                byte
	Defence                   byte
	Activity                  byte
	TargetPolicy              byte
	ClockGlyph                byte
	FontColor                 byte
	JoinPending               byte
	WaterChoice               byte
	WaterUpdates              byte
	WaterSlow                 byte
	AmbientMode               byte
	AmbientY                  byte
	AmbientX                  byte
	AmbientFrame              byte
	AmbientCounter            byte
	AmbientDirection          byte
	AmbientColor              byte
	AmbientSelectionData      byte
	AmbientRippleCycle        byte
	AmbientSelector           byte
	AmbientSelection          byte
	AmbientRepeats            byte
	AmbientSelectionCount     byte
	AmbientStyle              byte
	AmbientStepClock          byte
	AmbientDelay              byte
	AmbientInitial            byte
	Humans                    [3]byte
	Order                     [3]byte
	Rest                      [3]byte
	SoundCooldown             [2]byte
	PauseIndices              [3]byte
	Points                    [3]byte
	ScoreDigits               [9]byte
	SoundDown                 [3]byte
	SoundHits                 [3]byte
	FighterActivity           [3]byte
	RoundScoreHigh            [3]byte
	RoundScoreLow             [3]byte
	Joystick                  [2]byte
	SparkleCounters           [8]byte
	SparkleX                  [8]byte
	SparkleY                  [8]byte
	Stun                      [3]byte
	Recovery                  [3]byte
	DroolFrame                [3]byte
	StarFrame                 [3]byte
	StarColor                 [3]byte
	StarX                     [3]byte
	StarY                     [3]byte
	Puff                      [3]byte
	PuffX                     [3]byte
	PausePositions            [3]byte
	PauseFacing               [3]byte
	OwnerFlash                [2]byte
	Initials                  [3]byte
	Keys                      [4]byte
	ScoreHigh                 [51]byte
	ScoreLow                  [51]byte
	ScoreCandidate            [51]byte
	NameFirst                 [51]byte
	NameSecond                [51]byte
	NameThird                 [51]byte
	ReflectionPhases          [21]byte
	ReflectionLeft            [21]byte
	BlackScreen               uint16
	GroundShade               uint16
	EffectColor               uint16
	Volume                    uint16
	HintCount                 uint16
	TitleLanguage             uint16
	StartupReserved           uint16
	FontOutline               uint16
	Filter                    uint16
	German                    uint16
	FontPosition              uint16
	RandomCounter             uint16
	GlyphCounter              uint16
	BorderFrame               uint16
	BorderOffset              uint16
	BorderEnabled             uint16
	AlternateShield           uint16
	PauseSound                uint16
	Frozen                    uint16
	FontOffset                uint16
	HintTimer                 uint16
	HintIndex                 uint16
	FontLine                  uint16
	FadeDivider               uint16
	TitleStage                uint16
	TitleAdvance              uint16
	TitleEnding               uint16
	TitleTimer                uint16
	TitleExitTimer            uint16
	TitleActorsEnabled        uint16
	TitlePicture              uint16
	ColorSequence             uint16
	ColorRotations            uint16
	ColorRotationTimer        uint16
	ColorTimer                uint16
	LastRawKey                uint16
	RainbowY                  uint16
	RainbowColor              uint16
	ParticleOrigin            uint16
	LetterCount               uint16
	LetterIndex               uint16
	LetterStep                uint16
	LetterMotion              uint16
	LetterOperation           uint16
	LetterComplete            uint16
	ScrollCount               uint16
	ScrollDirection           uint16
	RandomMultiplier          uint16
	RandomAddition            uint16
	RandomXor                 uint16
	CollisionEnabled          uint16
	SecondaryCollisionEnabled uint16
	ScriptX                   uint16
	ScriptY                   uint16
	ScriptInitial             uint16
	ScriptClock               uint16
	PacCycle                  uint16
	PeriScript                uint16
	PeriOffset                uint16
	PeriPause                 uint16
	PeriPose                  uint16
	PeriFrame                 uint16
	PeriClock                 uint16
	GraphicsColor             uint16
	VisibleEffectColor        uint16
	VisibleBlueBelt           uint16
	VisibleWhiteBelt          uint16
	VisibleRedBelt            uint16
	VisibleGroundShade        uint16
	BorderCell                uint16
	BeltColors                [3]uint16
	AmbientForces             [4]uint16
	TitlePalette              [16]uint16
	Background                PictureBuffer
	Drawing                   PictureBuffer
	EraseStart                RestorationCursor
	EraseCursor               RestorationCursor
	Displayed                 PictureBuffer
	FreezeLocation            WaitPoint
	BootMarker                uint32
	KeyHistory                uint32
	CollisionRead             CoveragePlane
	CollisionReadMirror       CoveragePlane
	CollisionWrite            CoveragePlane
	CollisionWriteMirror      CoveragePlane
}

// TitleParticle stores one animated point and its previous painted position.
type TitleParticle struct{ X, Y, VX, VY, PreviousX, PreviousY uint16 }

// SetAmbient preserves the scenery controller between arena and title scenes.
func (s *PresentationState) SetAmbient(a AmbientState) {
	s.AmbientMode, s.AmbientY, s.AmbientX = a.Mode, a.Y, a.X
	s.AmbientFrame, s.AmbientCounter = a.Frame, a.Counter
	s.AmbientDirection, s.AmbientColor = a.Direction, a.Color
	s.AmbientSelectionData, s.AmbientRippleCycle, s.AmbientSelector = a.SelectionData, a.RippleCycle, a.Selector
	s.AmbientSelection, s.AmbientRepeats, s.AmbientSelectionCount = a.Selection, a.Repeats, a.SelectionCount
	s.AmbientStyle, s.AmbientStepClock, s.AmbientDelay = a.Style*10, a.StepClock, a.Delay
	s.AmbientInitial = 0
	if a.Initial {
		s.AmbientInitial = 1
	}
	s.ScriptX, s.ScriptY, s.ScriptClock = a.ScriptX, a.ScriptY, a.ScriptClock
	s.ScriptInitial = 0
	if a.ScriptInitial {
		s.ScriptInitial = 1
	}
	s.PacCycle = a.PacCycle
	s.PeriScript, s.PeriOffset, s.PeriPause = a.PeriScript, a.PeriOffset, a.PeriPause
	s.PeriPose, s.PeriFrame, s.PeriClock = a.PeriPose, a.PeriFrame, a.PeriClock
	for i := range s.AmbientForces {
		s.AmbientForces[i] = a.ForceWord(i)
	}
}

func (s *PresentationState) Ambient() AmbientState {
	a := AmbientState{
		Mode: s.AmbientMode, Y: s.AmbientY, X: s.AmbientX, Frame: s.AmbientFrame, Counter: s.AmbientCounter,
		Direction: s.AmbientDirection, Color: s.AmbientColor, SelectionData: s.AmbientSelectionData,
		RippleCycle: s.AmbientRippleCycle, Selector: s.AmbientSelector, Selection: s.AmbientSelection,
		Repeats: s.AmbientRepeats, SelectionCount: s.AmbientSelectionCount, Style: s.AmbientStyle / 10,
		StepClock: s.AmbientStepClock, Delay: s.AmbientDelay, Initial: s.AmbientInitial != 0,
		ScriptX: s.ScriptX, ScriptY: s.ScriptY, ScriptInitial: s.ScriptInitial != 0, ScriptClock: s.ScriptClock,
		PacCycle: s.PacCycle, PeriScript: s.PeriScript, PeriOffset: s.PeriOffset, PeriPause: s.PeriPause,
		PeriPose: s.PeriPose, PeriFrame: s.PeriFrame, PeriClock: s.PeriClock,
	}
	for i, v := range s.AmbientForces {
		a.SetForceWord(i, v)
	}
	return a
}

// PresentationState returns the attached scene controller, if a scene is active.
func (g *Game) PresentationState() *PresentationState { return g.presentationState }
