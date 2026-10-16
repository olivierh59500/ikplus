// Package game implements the deterministic, display-independent IK+ match.
// One call to Tick advances one 50 Hz PAL simulation step.
package game

import "github.com/olivierh59500/ikplus/internal/audio"

const (
	TickRate   = 50
	Width      = 320
	Height     = 200
	GroundY    = 194.0
	RoundTicks = 30 * TickRate
)

// Input is a joystick state. Y=-1 means up; Y=1 means down.
// Directions are absolute screen directions and are mirrored by the fighter.
type Input struct {
	X, Y              int
	Fire              bool
	originalAction    int
	hasOriginalAction bool
}

// Phase is the current match stage.
type Phase uint8

const (
	Attract Phase = iota
	Ready
	Fight
	Judge
	BallBonus
	BombBonus
	BonusEnd
	GameOver
	PauseDemo
)

func (p Phase) String() string {
	return [...]string{"DEMO", "READY", "FIGHT", "JUDGE", "BALL BONUS", "BOMB BONUS", "BONUS RESULT", "GAME OVER", "PAUSE"}[p]
}

// Move describes the visible fighter action. Attack positions are held until
// another joystick selection is made, rather than repeated by autofire.
type Move uint8

const (
	Idle Move = iota
	WalkForward
	WalkBackward
	Jump
	FrontPunch
	ReversePunch
	ShinKick
	Sweep
	CrouchPunch
	FlyingKick
	Headbutt
	StomachKick
	FaceKick
	ReverseSweep
	BackstepKick
	Backflip
	DoubleKick
	Block
	Knockdown
	Rise
	Shield
)

func (m Move) String() string {
	return [...]string{"STANCE", "WALK", "RETREAT", "JUMP", "FACE PUNCH", "REVERSE PUNCH", "SHIN KICK", "FOOTSWEEP", "LOW PUNCH", "FLYING KICK", "HEADBUTT", "STOMACH KICK", "FACE KICK", "REVERSE SWEEP", "BACKSTEP KICK", "BACKFLIP", "DOUBLE KICK", "BLOCK", "KNOCKDOWN", "GET UP", "SHIELD"}[m]
}

// Fighter contains renderable and score state. X/Y locate the source origin.
// SpriteFrame already contains airborne body offsets and is drawn at the ground.
// Height is retained for the pending bonus-collision reconstruction; combat
// contact uses the original published pose rather than a synthetic lift.
// ID 0 is white, 1 red, and 2 blue.
type Fighter struct {
	ID                        int
	X, Y                      float64
	Height                    float64
	Facing                    int
	Move                      Move
	Action                    int
	Age                       int
	SpriteFrame               int
	AnimationStep             int
	Points, RoundScore, Score int
	Human                     bool
	Eliminated                bool
	Rank                      int
	Stun, GetUp               int
	StarFrame, StarColor      byte
	StarX, StarY, DroolFrame  byte
	Puff                      bool
	PuffX                     byte
	ShieldY                   float64
	ShieldX                   float64
	ShieldState, ShieldType   int
	ShieldAlternate           bool
	ShieldSide                int
	velocity                  float64
	selected                  Move
	hitMask                   uint8
	aiInput                   Input
	aiWait                    int
	lastInput                 Input
	lastAnimationStep         int
	recoveryWait              int
	reactionAction            int
	riseAction                int
	roundActive               bool
	activityByte              byte
	computerAnimation         computerAnimationState
	computerAnimationReady    bool
	humanCommandFrame         uint64
}

// Event lets presentation code trigger audio and effects without coupling the
// simulation to an audio device. Events are replaced at the next Tick call.
type Event struct {
	Kind            EventKind
	Move            Move
	Frame           int
	Fighter, Target int
	X, Y            float64
	Value           int
}

type EventKind uint8

const (
	AttackEvent EventKind = iota
	HitEvent
	BlockEvent
	RoundEvent
	BonusEvent
	DeflectEvent
	ExplosionEvent
	GameOverEvent
)

// Ball is one of the original challenge's four concurrent projectile slots.
type Ball struct {
	Active       bool
	X, Y, VX, VY float64
	Flashing     bool
	Deflected    bool
	Frame, Color int
	drawn        bool
	source       originalBallState
}

// Bomb moves onto the arena, burns its fuse, and can be kicked away.
type Bomb struct {
	X, Y, VX, VY          float64
	Fuse                  int
	Kicked                bool
	Explosion             int
	Active                bool
	Stage, StageTicks     int
	Frame, Color, Variant int
	BlastFrame            int
	Fragment              bool
	burnTicks             int
	blastCounter          int
	drawn                 bool
	pictureBank           int
	pictureFrame          uint64
	groundPicture         bombGroundPicture
	source                originalBallState
}

type bombGroundPicture struct {
	valid bool
	bank  int
	x, y  float64
}

// BonusState exposes projectile and shield information to the renderer.
type BonusState struct {
	preparation                ballPreparationWork
	bombPreparation            bombPreparationWork
	Fighter                    int
	Number                     int
	Total                      int
	Deflected                  int
	Success                    bool
	Ball                       Ball
	Bombs                      []Bomb
	Balls                      []Ball
	Ticks                      int
	ShieldVisible              bool
	spawnWait                  int
	spawnAccumulator           int
	launchRate                 int
	ballGravity                int
	bombFuse                   int
	bombDelay                  int
	activeCount                int
	spawnDone                  bool
	failed                     bool
	failureAction              int
	kind                       int
	turn                       int
	clearTicks                 int
	introRemaining             int
	introIRQTimer              byte
	introExpiredFrozen         bool
	live                       bool
	deathStarted               bool
	deathVisible               bool
	resultStage                bonusResultStage
	resultActor                int
	resultIRQTimer             byte
	resultFrames               byte
	resultFrame                byte
	resultWaitComplete         bool
	resultRetainPicture        bool
	finalClearing              bool
	savedSpeed, clearingPeriod byte
	shieldMotion               originalBallState
	shieldFrame                int
	shieldColor                int
	contactFighter             Fighter
	maskReady                  bool
	entryFacing                int
	queue                      []int
}

// Game owns the match and a private reproducible random stream.
type Game struct {
	ballWorkTraceEnabled               bool
	ballPreparationTasks               []BallPreparationTask
	ballMaintenanceSnapshots           []BallMaintenanceSnapshot
	captionTraceEnabled                bool
	captionPreparations                []CaptionPreparationSnapshot
	ballPreparationEnabled             bool
	bombPreparationEnabled             bool
	bombPreparations                   []BombPreparationSnapshot
	ownerIndicatorPaints               int
	ballPreparedPictures               []BallPreparedPicture
	displayLatch                       bool
	displayPicture                     PreparedPicture
	displayPictureFrame                uint64
	displayPictureBank                 int
	fightCompletionPending             bool
	fightPendingContacts               []contact
	fightPaintDebt                     int
	fightBonusReturnHold               bool
	fighterEffectPreparation           *fighterEffectPreparation
	fightSnapshot                      FightSnapshot
	fightSnapshots                     []FightSnapshot
	roundPreparation                   roundPreparationStage
	Phase                              Phase
	Fighters                           [3]Fighter
	Round                              int
	Remaining                          int
	PhaseTicks                         int
	Frame                              uint64
	sourceFrameOffset                  byte
	presentationState                  *PresentationState
	retainedPresentation               *retainedPresentationData
	presentationSpeedChanged           func()
	Bonus                              BonusState
	Events                             []Event
	Paused                             bool
	Message                            string
	seed, random                       uint64
	players                            int
	bonusCount                         int
	clockTicks                         int
	roundCycle                         int
	animationOrigin                    uint64
	demoDifficulty                     int
	demonstrating                      bool
	candidateOwner                     byte
	caption                            sourceCaption
	hint                               HintState
	hintOutline                        uint16
	hintPosition, hintOffset, hintLine uint16
	hintGlyphCounter                   uint16
	hintColor                          byte
	bonusMemory                        [10]Bomb
	bonusAccumulator                   int
	bonusNumberMemory                  int
	bonusFailedMemory, bonusDoneMemory bool
	reflectionPhases                   [21]byte
	reflectionUpdates, reflectionSlow  byte
	reflectionReady                    bool
	displayCounter                     byte
	displayElapsed                     byte
	sourceVBLWraps, inputRepeatCounter byte
	speedValue, displaySpeedValue      byte
	sourceFast                         byte
	speedKeys                          [4]byte
	speedLastRaw, speedKeyDelay        byte
	speedKeyHistory                    uint32
	speedKeyValid                      bool
	musicDisabled, effectsDisabled     bool
	filterLatch, ledFilterEnabled      bool
	masterVolumeValue                  uint16
	graphicsIndex, refereeVariant      byte
	groundShade                        uint16
	graphicsPublished, groundPublished uint16
	germanText, displayGermanText      bool
	alternateShield                    bool
	sourceFrozen                       bool
	restartRequested                   bool
	sourceJoinPlayers                  int
	waitPoint, pausePoint              WaitPoint
	sourceBootMarker                   uint32
	pauseDemo                          pauseDisplay
	pauseCooldown                      byte
	sceneClearFrame                    uint64
	judge                              judgeState
	ambient                            AmbientState
	ambientFinishWork                  int
	ambientReady                       bool
	fightPrepared                      bool
	visibleFighters                    [3]Fighter
	pictures                           [2]PreparedPicture
	bombOperations                     [2][]BombPictureOperation
	bombIRQOperations                  []BombPictureOperation
	ballRestores                       [2][]BallPictureOperation
	ballPictureBank                    int
	ballPictureValid                   bool
	irqPictureBank                     int
	effectColor, effectColorVisible    uint16
	beltColors, beltColorsVisible      [3]uint16
	paletteWaitFrame                   uint64
	displaySeconds                     int
	displayRound                       int
	drawPriority                       byte
	displayMode                        byte
	displayFrame                       byte
	visibleValid                       bool
	visibleAt                          uint64
	joystick                           [2]Input
	physicalJoystickPins               bool
	ownerFlash                         [2]byte
	ownerIndicators                    [2]bool
	audioControl                       *audio.Control
	audioEngine                        *audio.Player
	audioPreparationTrace              bool
	audioPreparations                  []AudioPreparationSnapshot
	palFrameProfile                    audio.PALFrameProfile
	audioTicks                         uint64
	audioEffectsDone                   bool
	audioInFrame                       bool
	soundCooldown                      [2]byte
	soundHits                          [3]byte
	soundDown                          [3]byte
}
