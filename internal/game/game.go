package game

import "math"

// New creates a demo match using a reproducible random seed.
func New(seed uint64) *Game {
	if seed == 0 {
		seed = 0x494b2b
	}
	g := NewCold(seed)
	g.beginDemo()
	return g
}

// NewCold creates the core before the original cold startup. It consumes no
// arena RNG and performs no display swap; ColdStart prepares the first title.
func NewCold(seed uint64) *Game {
	g := &Game{seed: seed, random: seed, groundShade: 0x333}
	g.resetReflection()
	g.resetAmbient()
	return g
}

// beginDemo initializes the source arena demonstration at $77AA/$623C.
// The native title controls stay available over this all-computer match.
func (g *Game) beginDemo() {
	g.beginDemoArena(true)
}

func (g *Game) beginDemoArena(swap bool) {
	g.roundPreparation = 0
	g.sourceJoinPlayers, g.restartRequested = 0, false
	g.clearHintArena()
	g.clearSpeedKeys()
	g.pauseDemo, g.pauseCooldown = pauseDisplay{}, 0
	g.sceneClearFrame = 0
	g.alternateShield = false
	g.candidateOwner = 0
	g.drawPriority = 0
	g.effectColor, g.effectColorVisible = 0x00f, 0x00f
	g.displaySeconds, g.displayMode = 30, 1
	g.displayRound = 1
	g.fightPrepared = false
	g.fightCompletionPending = false
	g.fightPendingContacts = g.fightPendingContacts[:0]
	g.fightPaintDebt = 0
	g.fightBonusReturnHold = false
	g.fighterEffectPreparation = nil
	g.displayPictureFrame = 0
	g.fightSnapshot = FightSnapshot{}
	g.fightSnapshots = g.fightSnapshots[:0]
	g.judge = judgeState{}
	// $77C6 clears the speech/sparkle controller before the new demo.
	g.caption = sourceCaption{}
	g.hideCaption()
	g.demonstrating, g.demoDifficulty = true, 1
	g.Phase, g.PhaseTicks, g.Round, g.Remaining, g.clockTicks = Attract, 0, 1, RoundTicks, 0
	g.Message = "PRESS FIRE TO START"
	g.players, g.bonusCount, g.Bonus = 0, 0, BonusState{}
	g.roundCycle, g.animationOrigin = 1, g.Frame
	g.resetFighters(false)
	g.displayCounter = 0
	g.displayElapsed = 0
	g.sourceVBLWraps, g.inputRepeatCounter = 0, 0
	g.ownerIndicators = [2]bool{}
	g.visibleValid = false
	g.pictures = [2]PreparedPicture{}
	g.bombOperations = [2][]BombPictureOperation{}
	g.bombIRQOperations = nil
	g.ballRestores = [2][]BallPictureOperation{}
	g.ballPictureValid = false
	g.resetOwnerIndicators()
	g.bonusMemory = [10]Bomb{}
	g.bonusAccumulator = 0
	g.bonusNumberMemory, g.bonusFailedMemory, g.bonusDoneMemory = 0, false, false
	// $63B8 consumes one random byte for blue, then red, then white. Its
	// raw bases are $78/$50/$24 and each jitter is the low two bits.
	for id := 2; id >= 0; id-- {
		f := &g.Fighters[id]
		f.X = float64([3]int{36, 80, 120}[id]+int(g.originalRandomByte()&3)) * 2
		f.Action = 28
		g.resetSourceFighter(id)
	}
	g.resetBeltPalette()
	// $77AA completes one immediate $65C4 swap after $77C6/$623C.
	if swap {
		g.completeDisplayWait()
	}
}

// ReturnToDemo opens a fresh arena demonstration while retaining the running
// music and audio clock. Original $77C6 clears the RNG counter, not its three
// code-section words or the low VBL byte used by $785C.
func (g *Game) ReturnToDemo() {
	g.resetArenaAudio()
	g.random &= 0x0000ffffffffffff
	g.Paused = false
	g.beginDemo()
}

// Start begins a fresh one- or two-player tournament and resets AI difficulty.
func (g *Game) Start(players int) {
	g.startTournament(players, g.seed, 0, false)
}

// Join begins human play through the original arena-reset boundary. The
// running VBL byte, code-section RNG words, scenery and audio remain live.
func (g *Game) Join(players int) {
	g.resetArenaAudio()
	g.startTournament(players, g.random&0x0000ffffffffffff, g.Frame, true)
}

func (g *Game) startTournament(players int, random, frame uint64, retainScenery bool) {
	g.roundPreparation = 0
	g.sourceJoinPlayers, g.restartRequested = 0, false
	g.clearHintArena()
	g.clearSpeedKeys()
	g.pauseDemo, g.pauseCooldown = pauseDisplay{}, 0
	g.sceneClearFrame = 0
	g.alternateShield = false
	g.candidateOwner = 0
	g.drawPriority = 0
	g.effectColor, g.effectColorVisible = 0x00f, 0x00f
	g.displaySeconds, g.displayMode = 30, 1
	g.displayRound = 1
	if players < 1 {
		players = 1
	}
	if players > 2 {
		players = 2
	}
	g.players, g.Round, g.bonusCount, g.roundCycle = players, 0, 0, 0
	g.demonstrating, g.demoDifficulty, g.clockTicks = false, 0, 0
	g.random, g.Frame, g.Paused = random, frame, false
	if !retainScenery {
		g.sourceFrameOffset = 0
	}
	g.Events, g.Bonus = nil, BonusState{}
	g.pictures = [2]PreparedPicture{}
	g.bombOperations = [2][]BombPictureOperation{}
	g.bombIRQOperations = nil
	g.ballRestores = [2][]BallPictureOperation{}
	g.ballPictureValid = false
	g.visibleValid = false
	g.bonusMemory = [10]Bomb{}
	g.bonusAccumulator = 0
	g.bonusNumberMemory, g.bonusFailedMemory, g.bonusDoneMemory = 0, false, false
	if !retainScenery {
		g.resetReflection()
		g.resetAmbient()
	}
	g.caption = sourceCaption{}
	g.hideCaption()
	g.displayCounter = 0
	g.displayElapsed = 0
	g.sourceVBLWraps, g.inputRepeatCounter = 0, 0
	g.ownerIndicators = [2]bool{}
	g.resetFighters(true)
	g.advanceRound()
	g.resetBeltPalette()
}

func (g *Game) resetFighters(human bool) {
	g.soundHits = [3]byte{}
	g.soundDown = [3]byte{}
	for i := range g.Fighters {
		g.Fighters[i] = Fighter{ID: i, X: []float64{72, 160, 248}[i], Y: GroundY, Facing: 1, Human: human && i < g.players, Move: Idle, selected: Idle, ShieldY: GroundY - 30, ShieldSide: 1, lastAnimationStep: -1}
		if i > 0 {
			g.Fighters[i].Facing = -1
		}
	}
}

func (g *Game) beginRound() {
	g.speedKeys[0] = 0
	g.soundHits = [3]byte{}
	g.soundDown = [3]byte{}
	g.drawPriority = 0
	g.fightPrepared = false
	g.fightCompletionPending = false
	g.fightPendingContacts = g.fightPendingContacts[:0]
	g.fightPaintDebt = 0
	g.fightBonusReturnHold = false
	g.fighterEffectPreparation = nil
	g.displayPictureFrame = 0
	g.fightSnapshot = FightSnapshot{}
	g.fightSnapshots = g.fightSnapshots[:0]
	g.judge = judgeState{}
	g.hideCaption()
	g.Phase, g.PhaseTicks, g.Remaining = Ready, 0, RoundTicks
	g.clockTicks, g.animationOrigin = 0, g.Frame
	g.Message = "READY"
	g.resetOwnerIndicators()
	humans := 0
	for _, f := range g.Fighters {
		if f.Human {
			humans++
		}
	}
	for i := range g.Fighters {
		f := &g.Fighters[i]
		previousThreat := f.computerAnimation.Threat
		f.X, f.Y, f.Height = []float64{72, 160, 248}[i], GroundY, 0
		f.Facing = 1
		if i == 2 || i == 1 && humans < 2 {
			f.Facing = -1
		}
		f.Move, f.selected, f.Age, f.Action = Idle, Idle, 0, 28
		f.Points, f.RoundScore, f.Rank = 0, 0, 0
		f.Stun, f.velocity, f.hitMask = 0, 0, 0
		f.SpriteFrame, f.AnimationStep, f.lastAnimationStep, f.recoveryWait = 0, 0, -1, 0
		f.lastInput, f.aiInput, f.aiWait = Input{}, Input{}, 0
		f.roundActive = false
		f.activityByte = 0
		g.resetSourceFighter(i)
		f.computerAnimation.Threat = previousThreat
	}
	// $63B8 consumes randomness for blue, red, then white, after each reset.
	for id := 2; id >= 0; id-- {
		f := &g.Fighters[id]
		f.X = float64([3]int{36, 80, 120}[id]+int(g.originalRandomByte()&3)) * 2
		f.computerAnimation.Position = byte(f.X / 2)
	}
	g.emit(RoundEvent, -1, -1, 0, 0, g.Round)
}

// advanceRound follows $623C's independent displayed-round and fight-cycle
// counters. Displayed rounds saturate at 99 while the bonus cycle continues.
func (g *Game) advanceRound() {
	g.soundHits = [3]byte{}
	g.soundDown = [3]byte{}
	g.Round = min(99, g.Round+1)
	g.roundCycle++
	g.clockTicks, g.Remaining = 0, RoundTicks
	for id := range g.Fighters {
		f := &g.Fighters[id]
		f.Points, f.RoundScore, f.Rank, f.roundActive = 0, 0, 0, false
		f.activityByte = 0
	}
	// $623C clears absent owners before its bonus/ordinary-round branch.
	for id := 0; id < 2; id++ {
		if !g.Fighters[id].Human {
			g.ownerIndicators[id] = false
		}
	}
	if g.roundCycle < 3 {
		g.beginRound()
		if g.roundPreparation == 0 {
			g.rememberFightBoundary()
		}
		return
	}
	g.roundCycle = 0
	g.bonusCount++
	var survivors []int
	for id := 1; id >= 0; id-- {
		if g.Fighters[id].Human {
			survivors = append(survivors, id)
		}
	}
	g.Bonus = BonusState{queue: survivors}
	mode := byte(5)
	if g.bonusCount%2 == 0 {
		mode = 7
	}
	g.rememberFightBoundaryMode(mode)
	g.startBonus(survivors[0])
}

// Tick advances exactly one PAL tick. Input slots remain assigned to white and
// red even when one of the two human players has been eliminated.
func (g *Game) Tick(inputs [2]Input) {
	g.Events = g.Events[:0]
	g.captionPreparations = g.captionPreparations[:0]
	g.ballPreparationTasks = g.ballPreparationTasks[:0]
	g.ballMaintenanceSnapshots = g.ballMaintenanceSnapshots[:0]
	g.ballPreparedPictures = g.ballPreparedPictures[:0]
	g.bombPreparations = g.bombPreparations[:0]
	if g.Paused {
		return
	}
	pictureCounter, pictureMode := g.displayCounter, g.OriginalMode()
	picturePhase := g.Phase
	g.waitPoint = g.arenaWaitPoint()
	g.SetSourceJoystick(inputs)
	if g.inArena() && !g.fightPrepared {
		g.prepareFightPicture(inputs)
	}
	g.irqPictureBank, _ = g.SceneryBuffers()
	if g.displayLatch || g.palFrameProfile.FrameClocks() == 227*313 {
		g.displayPicture = g.Picture(g.irqPictureBank)
		g.displayPictureBank, g.displayPictureFrame = g.irqPictureBank, g.Frame+1
	}
	g.bombIRQOperations = g.bombIRQOperations[:0]
	g.Frame++
	if g.OriginalVBL() == 0 {
		g.sourceVBLWraps++
	}
	if g.judge.timer != 0 {
		g.judge.timer--
	}
	if g.Phase == BonusEnd && g.Bonus.resultIRQTimer != 0 {
		g.Bonus.resultIRQTimer--
	}
	if g.Phase == BombBonus && g.Bonus.bombPreparation.timer != 0 {
		// The reaction deadline continues while main preparation is frozen.
		g.Bonus.bombPreparation.timer--
	}
	if (g.Phase == BallBonus || g.Phase == BombBonus) && g.Bonus.clearTicks == 0 && !g.Bonus.live && g.Bonus.introIRQTimer > 0 {
		g.Bonus.introIRQTimer--
	}
	g.beginSoundFrame()
	g.publishDisplayOptions()
	g.pollSpeedKey()
	if g.RestartRequested() || g.JoinRequested() != 0 {
		return // The frontend replaces the discarded program in this journal.
	}
	g.effectColorVisible = g.effectColor
	g.beltColorsVisible = g.beltColors
	if g.SourceFrozen() {
		if (g.Phase == BallBonus || g.Phase == BombBonus) && g.Bonus.clearTicks == 0 && !g.Bonus.live && g.Bonus.introIRQTimer == 0 {
			g.Bonus.introExpiredFrozen = true
		}
		g.endSoundFrame()
		return
	}
	g.PhaseTicks++
	if g.OriginalMode() != 8 {
		if g.pauseCooldown != 0 {
			g.pauseCooldown--
		}
		g.inputRepeatCounter++
		g.tickSoundTimers()
		g.displayElapsed++
		g.tickReflection()
		g.tickAmbient()
		g.tickFighterTimers()
		if g.fighterEffectPreparation != nil {
			g.advanceTimedFighterEffects(g.judgeMaintenanceWork())
		}
		g.tickOwnerIndicators()
		if g.OriginalMode() == 1 && g.Fighters[0].Action != 28 && g.hint.Timer != 0 {
			g.hint.Timer--
		}
		g.tickDisplayCounter()
	}
	if pictureMode != 5 && pictureMode != 7 {
		g.updateSoundEffects()
	}
	if g.roundPreparation != 0 {
		g.tickRoundPreparation()
	} else {
		switch g.Phase {
		case Attract, Ready, Fight:
			g.tickFight(inputs)
		case Judge:
			g.tickJudge()
		case PauseDemo:
			g.tickPauseDemo()
		case BallBonus:
			g.tickBalls(inputs)
		case BombBonus:
			g.tickBombs(inputs)
		case BonusEnd:
			if g.pacedBallPreparation() {
				g.tickPacedBalls(inputs)
			} else {
				g.tickBonusResult(inputs)
			}
		case GameOver:
			// Presentation owns high-score entry and the request to start a match.
		}
	}
	if g.inArena() && !g.fightPrepared {
		g.prepareFightPicture(inputs)
	}
	g.prepareBallWaitByte()
	if g.Phase == BallBonus || g.Phase == BombBonus || g.Phase == BonusEnd {
		// The main thread paints after a completed wait, while IRQ-only VBLs
		// retain the existing screen, including its sampled effect timers.
		// $C882 returns after the final failed bomb wait without another
		// actor preparation. IRQ erasure of that retained actor must survive.
		finishedFailedBomb := picturePhase == BombBonus && g.Phase == BonusEnd && !g.Bonus.Success
		retainedBombResult := g.Phase == BonusEnd && g.Bonus.resultRetainPicture
		if !g.pacedBallPreparation() && (!g.pacedBombPreparation() || g.Bonus.clearTicks > 0) && !finishedFailedBomb && !retainedBombResult && (g.displayCounter != pictureCounter || g.OriginalMode() != pictureMode) {
			g.rememberPicture()
		}
		if g.Phase == BonusEnd && g.Bonus.resultStage == bonusResultFinalFrame {
			g.clearArenaPictures()
		}
	}
	g.endSoundFrame()
}

func (g *Game) anyWinner() bool {
	for _, f := range g.Fighters {
		if f.Points >= 6 {
			return true
		}
	}
	return false
}

func (g *Game) endRound() {
	g.beginJudge()
	g.emit(RoundEvent, -1, -1, 0, 0, g.Round)
}

func (g *Game) afterJudge() {
	if g.judge.active && !g.judge.complete {
		return
	}
	if g.demonstrating {
		// The frontend owns $A110 and $24AC before the actual $77AA reset.
		g.Phase, g.PhaseTicks = GameOver, 0
		g.rememberFightBoundary()
		return
	}
	var survivors []int
	// $B730/$C5E2 start at red and then decrement to white.
	for i := 1; i >= 0; i-- {
		if g.Fighters[i].Human {
			survivors = append(survivors, i)
		}
	}
	if len(survivors) == 0 || g.judge.active && g.judge.ending {
		g.Phase, g.PhaseTicks, g.Message = GameOver, 0, "GAME OVER"
		g.emit(GameOverEvent, -1, -1, 0, 0, 0)
		// The original returns to the dispatcher before entering scores.
		g.rememberFightBoundary()
		return
	}
	if g.palFrameProfile.FrameClocks() == 227*313 {
		g.roundPreparation = roundPreparationBonus
		if g.roundCycle+1 < 3 {
			g.roundPreparation = roundPreparationFighters
		}
		return
	}
	g.advanceRound()
}

func (g *Game) nextRandom() uint64 {
	return uint64(g.originalRandomByte())
}

func (g *Game) randomInt(n int) int { return int(g.nextRandom() % uint64(n)) }

func normalize(i Input) Input {
	i.X, i.Y = sign(i.X), sign(i.Y)
	return i
}

func sign(i int) int {
	if i < 0 {
		return -1
	}
	if i > 0 {
		return 1
	}
	return 0
}

func side(x float64) int {
	if x < 0 {
		return -1
	}
	return 1
}

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }

func (g *Game) emit(kind EventKind, fighter, target int, x, y float64, value int) {
	move := Idle
	if fighter >= 0 && fighter < len(g.Fighters) {
		move = g.Fighters[fighter].Move
	}
	g.Events = append(g.Events, Event{Kind: kind, Move: move, Fighter: fighter, Target: target, X: x, Y: y, Value: value})
}

// SelectMove reproduces the manual's two eight-direction joystick maps.
// A backwards walk becomes Block only when an opponent attacks from the front.
func SelectMove(input Input, facing int) Move {
	i := normalize(input)
	x, y := i.X*facing, i.Y
	if x == 0 && y == 0 {
		return Idle
	}
	if !i.Fire {
		switch {
		case x == 0 && y < 0:
			return Jump
		case x > 0 && y < 0:
			return FrontPunch
		case x > 0 && y == 0:
			return WalkForward
		case x > 0 && y > 0:
			return ShinKick
		case x == 0 && y > 0:
			return Sweep
		case x < 0 && y > 0:
			return CrouchPunch
		case x < 0 && y == 0:
			return WalkBackward
		default:
			return ReversePunch
		}
	}
	switch {
	case x == 0 && y < 0:
		return FlyingKick
	case x > 0 && y < 0:
		return Headbutt
	case x > 0 && y == 0:
		return StomachKick
	case x > 0 && y > 0:
		return FaceKick
	case x == 0 && y > 0:
		return ReverseSweep
	case x < 0 && y > 0:
		return BackstepKick
	case x < 0 && y == 0:
		return Backflip
	default:
		return DoubleKick
	}
}
