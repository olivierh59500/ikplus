package game

// Preparation uses fixed work units per game update. The queue retains ordinary
// drawing tasks; it has no processor, hardware registers or interrupt service.
type ballWorkStage byte

// Window selection follows its own setup, before the retained header work.
// Multi-projectile dispatch and score publication stay as named native work;
// the branch totals below supply their variable geometry.
const (
	ballDisplayWaitSetupWork     = 77
	ballThreeObjectDispatchWork  = 1630
	ballFourObjectDispatchWork   = 1418
	ballLaunchDispatchWork       = 690
	ballScorePublicationWork     = 6120
	ballSingleObjectWindowMax    = 53320
	ballMultipleObjectWindowMax  = 55000
	ballSingleObjectLongPass     = 16820
	ballThreeObjectLongPass      = 40000
	ballFourObjectLongPass       = 50000
	ballEarlyThreeObjectMaxSpeed = 240
	ballRippleTwoObjectWindowMax = 57000
	ballRippleTwoObjectLongPass  = 25000
	ballRippleTwoObjectPassMax   = 28000
)

const (
	ballWorkEntry ballWorkStage = iota
	ballWorkRestore
	ballWorkCaption
	ballWorkReferee
	ballWorkSparkles
	ballWorkControl
	ballWorkActor
	ballWorkShield
	ballWorkShadows
	ballWorkWindow
	ballWorkHeader
	ballWorkInitialize
	ballWorkActivate
	ballWorkHold
	ballWorkExit
	ballWorkExitHold
	ballWorkDeath
	ballWorkWaitSetup
)

type ballPreparationWork struct {
	stage                        ballWorkStage
	debt                         int
	waitNext                     bool
	finalHolds                   bool
	sparkles                     captionSparkleWork
	slotDebt                     int
	copyRemainder                int
	objectsReady, firstObjects   bool
	randomFrames                 []byte
	trace                        *BallPreparationTask
	objectWork                   int
	objectPassWork               int
	effectWork                   int
	restoreAlignment             int
	paintAlignment               int
	launches                     int
	scorePaints                  int
	packedInstructionCycles      int
	restoreInstructionCycles     int
	paintInstructionCycles       int
	contactInstructionCycles     int
	motionInstructionCycles      int
	spinInstructionCycles        int
	groundInstructionCycles      int
	integrations                 int
	shieldPicture                shieldPicture
	actorWork                    int
	actorErases                  []BallActorErase
	objectsStopped               bool
	bodyContacts, shieldContacts [4]bool
}

// A falling shield keeps moving while a picture is prepared. Its drawing
// uses the position selected when painting starts, not its later live motion.
type shieldPicture struct {
	x, y               float64
	kind               int
	alternate, visible bool
}

// BallPreparedPicture records completed main-picture preparation for reference
// comparison. Gameplay and the pilot never consult these observations.
type BallPreparedPicture struct {
	Frame        uint64
	Progress     int
	Counter      byte
	Random       uint64
	Fighter      Fighter
	Caption      CaptionState
	HoldTimer    byte
	ShieldFlags  byte
	RandomFrames []byte
	ControlState byte
}

func (g *Game) BallPreparedPictures() []BallPreparedPicture { return g.ballPreparedPictures }

type BallPreparationTask struct {
	Stage                                byte
	Frame, EndFrame                      uint64
	Position, EndPosition                int
	Counter                              byte
	Pose, Work, Rows, Runs               int
	PaintingRotations, PaintingAlignment int
}

// BallMaintenanceSnapshot records native work inputs for development analysis.
// The preparation queue never consults this journal or an original reference.
type BallMaintenanceSnapshot struct {
	Frame                                                            uint64
	Work, ObjectWork, ObjectPassWork, Round                          int
	EffectWork, EffectID                                             int
	VBL                                                              byte
	Before, After                                                    BonusSnapshot
	Ambient                                                          AmbientState
	RestoreRows, PaintRows, ContactRows, Integrations, FloorContacts int
	RestoreAlignment, PaintAlignment, Launches, ScorePaints          int
	PackedInstructionCycles                                          int
	RestoreInstructionCycles, PaintInstructionCycles                 int
	ContactInstructionCycles, GroundInstructionCycles                int
	MotionInstructionCycles, SpinInstructionCycles                   int
	MusicDuration                                                    [4]uint16
	MusicSequence                                                    [4][2]int
	MusicEnvelope                                                    [4][2]int
	BodyContacts, ShieldContacts                                     [4]bool
	ContactCalls, ContactWork                                        int
}

func (g *Game) BallMaintenanceSnapshots() []BallMaintenanceSnapshot {
	return g.ballMaintenanceSnapshots
}

// TraceBallPreparation records task boundaries for independent development
// comparison. The game never reads the resulting journal.
func (g *Game) TraceBallPreparation() { g.ballWorkTraceEnabled = true }

func (g *Game) BallPreparationTasks() []BallPreparationTask { return g.ballPreparationTasks }

func (g *Game) observeBallPreparationTask(position int) {
	if !g.ballWorkTraceEnabled {
		return
	}
	w := &g.Bonus.preparation
	if w.trace != nil && w.trace.Stage == byte(w.stage) {
		return
	}
	if w.trace != nil {
		w.trace.EndFrame, w.trace.EndPosition = g.Frame, position
		g.ballPreparationTasks = append(g.ballPreparationTasks, *w.trace)
	}
	_, bank := g.SceneryBuffers()
	p := g.Picture(bank)
	w.trace = &BallPreparationTask{Stage: byte(w.stage), Frame: g.Frame, Position: position,
		Counter: g.displayCounter, Pose: g.Fighters[g.Bonus.Fighter].SpriteFrame,
		Work: w.debt, Rows: p.restoreRows, Runs: p.restoreRuns}
	switch w.stage {
	case ballWorkActor:
		f := g.Fighters[g.Bonus.Fighter]
		w.trace.PaintingRotations = actorRotationWords(f.SpriteFrame, true)
		w.trace.PaintingAlignment = int(f.X/2) & 7
	case ballWorkReferee:
		caption := g.Caption()
		w.trace.PaintingRotations = actorRotationWords(int(caption.RefereeFrame), true)
		w.trace.PaintingAlignment = int(caption.RefereeX) & 7
	}
}

// UseNativeBallPreparation retains drawing work across ordinary game updates.
// Intermediate picture timing remains independently checked by the full gates.
func (g *Game) UseNativeBallPreparation() { g.ballPreparationEnabled = true }

func (g *Game) pacedBallPreparation() bool {
	return g.ballPreparationEnabled && g.palFrameProfile.FrameClocks() == 227*313 &&
		(g.Phase == BallBonus || g.Phase == BonusEnd && g.Bonus.kind == 0)
}

func (g *Game) ballRestoreWork() int {
	_, bank := g.SceneryBuffers()
	p := g.Picture(bank)
	rowWork := 100
	halfRows := p.restoreRows / 2
	if g.Bonus.clearTicks > 0 && g.Bonus.turn == 1 {
		// Initial clearing precedes activation of both collision-mask planes.
		rowWork = 79
		halfRows = 0
	}
	base := 1046
	if p.restoreRuns != 0 {
		base += 142
	}
	return base + p.restoreRows*rowWork + halfRows + p.restoreRuns*181
}

func (g *Game) ballBackgroundWork() int {
	work := 7300 + g.ambientFinishWork
	if g.OriginalVBL()&3 == 2 {
		work += 1000
	}
	if g.ambient.Mode == 1 && g.ambient.Counter&3 == 0 {
		work += 1800
	}
	if g.ambient.Mode == 3 {
		work += ripplePreparationWork(g.ambient)
	}
	if g.OriginalVBL()&3 == 0 {
		work += 1000 + 6900*g.ownerIndicatorPaints
	}
	return work
}

func (g *Game) ballMaintenanceWork() int {
	w := &g.Bonus.preparation
	return g.ballBackgroundWork() + w.objectPassWork + w.objectWork + w.effectWork
}

func (g *Game) tickPacedBalls(inputs [2]Input) {
	b := &g.Bonus
	var beforeState BonusSnapshot
	if g.ballWorkTraceEnabled {
		beforeState = g.OriginalBonusSnapshot()
	}
	b.Ticks++
	b.preparation.firstObjects = false
	b.preparation.objectWork = 0
	b.preparation.objectPassWork = 0
	b.preparation.effectWork = 0
	b.preparation.restoreAlignment = 0
	b.preparation.paintAlignment = 0
	b.preparation.launches = 0
	b.preparation.scorePaints = 0
	b.preparation.packedInstructionCycles = 0
	b.preparation.restoreInstructionCycles = 0
	b.preparation.paintInstructionCycles = 0
	b.preparation.contactInstructionCycles = 0
	b.preparation.motionInstructionCycles = 0
	b.preparation.spinInstructionCycles = 0
	b.preparation.groundInstructionCycles = 0
	b.preparation.integrations = 0
	b.preparation.bodyContacts, b.preparation.shieldContacts = [4]bool{}, [4]bool{}
	var effectID int
	if g.audioControl != nil && g.audioControl.State.Active != 0 {
		// Starting a queued effect contributes ordinary sample handoff work.
		// Passive source painting intervals include this additional work;
		// the native player still emits commands without hardware emulation.
		b.preparation.effectWork = 600
		effectID = int(g.audioControl.State.ID)
	}
	var restoreRows, paintRows, contactRows, integrations, floorContacts int
	var contactCalls, contactWork int
	if !b.preparation.objectsStopped && (b.live || b.turn > 1 && b.clearTicks == 0) {
		b.preparation.firstObjects = !b.preparation.objectsReady
		b.preparation.objectsReady = true
		// The object pass visits every saved ball even after the dropped
		// shield leaves the arena. Its dispatch and drawing setup belong
		// to the pass, rather than to the shield's active-motion flag.
		b.preparation.objectPassWork = 4620
		if b.preparation.firstObjects {
			b.preparation.objectPassWork = 3000
		}
		if g.OriginalVBL()&3 == 2 {
			b.preparation.objectPassWork += 450
		}
		mask := b.contactFighter
		if !b.maskReady {
			mask = g.Fighters[b.Fighter]
		}
		if !b.ShieldVisible {
			mask.ShieldX = -10000
		}
		var before [4]Ball
		copy(before[:], b.Balls)
		number := b.Number
		deflected := b.Deflected
		// All overlap scans finish before an impact can change this flag.
		failedBeforeContacts := b.failed
		g.tickOriginalBallObjects(mask)
		b.preparation.scorePaints = b.Deflected - deflected
		if b.live || b.introRemaining == 0 {
			b.preparation.groundInstructionCycles = ballGroundInstructionCycles(b.Balls)
		}
		for id, previous := range before {
			previousRows := ballPictureRows(previous)
			if previous.drawn {
				_, cycles := ballPackedOperation(ballPackedRestore, previous.source)
				b.preparation.packedInstructionCycles += cycles
				b.preparation.restoreInstructionCycles += cycles
			}
			if previous.drawn && previousRows != 0 {
				restoreRows += previousRows
				b.preparation.restoreAlignment += previousRows * (int(previous.source.X>>7) & 15)
				b.preparation.objectWork += 2000 * previousRows / 11
			}
			if previous.source.Flags != 0 {
				integrations++
				if previous.source.Bounces != b.Balls[id].source.Bounces {
					floorContacts++
				}
				b.preparation.objectWork += 400
			}
			// C2C6 paints every saved slot, including inactive retained cells.
			if b.live || b.introRemaining == 0 {
				rows, cycles := ballPackedOperation(ballPackedPaint, b.Balls[id].source)
				b.preparation.packedInstructionCycles += cycles
				b.preparation.paintInstructionCycles += cycles
				if rows != 0 {
					paintRows += rows
					b.preparation.paintAlignment += rows * (int(b.Balls[id].source.X>>7) & 15)
					b.preparation.objectWork += 3200 * rows / 11
				}
			}
			x, _ := originalBallPosition(previous.source)
			if previous.drawn && (failedBeforeContacts || int(x)+24 >= 119 && int(x)+24 < 270) {
				rows, work := ballContactPreparationWork(previous.source)
				_, cycles := ballPackedOperation(ballPackedContact, previous.source)
				b.preparation.packedInstructionCycles += cycles
				b.preparation.contactInstructionCycles += cycles
				contactRows += rows
				contactCalls++
				contactWork += work
				b.preparation.objectWork += work
			}
		}
		if b.Number != number {
			// Launching follows the drawing pass. The new slot contributes
			// integration and picture work only on the following update.
			b.preparation.objectWork += 600
			b.preparation.launches++
		}
		b.preparation.integrations = integrations
		if integrations == 4 || (b.Deflected > 0 && integrations == 3) {
			cycles := b.preparation.packedInstructionCycles + b.preparation.motionInstructionCycles +
				b.preparation.spinInstructionCycles + b.preparation.groundInstructionCycles
			overhead := ballFourObjectDispatchWork
			if integrations == 3 {
				overhead = ballThreeObjectDispatchWork
			}
			objectWork := cycles/2 + overhead + ballLaunchDispatchWork*b.preparation.launches +
				ballScorePublicationWork*b.preparation.scorePaints
			b.preparation.objectWork = objectWork - b.preparation.objectPassWork
		}
	}
	g.updateSoundEffects()
	work := g.ballMaintenanceWork()
	if g.ballWorkTraceEnabled {
		snapshot := BallMaintenanceSnapshot{
			Frame: g.Frame, Work: work, ObjectWork: b.preparation.objectWork, Round: g.Round,
			ObjectPassWork: work - g.ballBackgroundWork() - b.preparation.objectWork - b.preparation.effectWork,
			VBL:            g.OriginalVBL(), Before: beforeState, After: g.OriginalBonusSnapshot(), Ambient: g.Ambient(),
			EffectWork: b.preparation.effectWork, EffectID: effectID,
			BodyContacts: b.preparation.bodyContacts, ShieldContacts: b.preparation.shieldContacts,
			ContactCalls: contactCalls, ContactWork: contactWork,
			RestoreRows: restoreRows, PaintRows: paintRows, ContactRows: contactRows,
			Integrations: integrations, FloorContacts: floorContacts,
			RestoreAlignment: b.preparation.restoreAlignment, PaintAlignment: b.preparation.paintAlignment,
			Launches: b.preparation.launches, ScorePaints: b.preparation.scorePaints,
			PackedInstructionCycles:  b.preparation.packedInstructionCycles,
			RestoreInstructionCycles: b.preparation.restoreInstructionCycles,
			PaintInstructionCycles:   b.preparation.paintInstructionCycles,
			ContactInstructionCycles: b.preparation.contactInstructionCycles,
			GroundInstructionCycles:  b.preparation.groundInstructionCycles,
			MotionInstructionCycles:  b.preparation.motionInstructionCycles,
			SpinInstructionCycles:    b.preparation.spinInstructionCycles}
		if g.audioEngine != nil {
			for voice, track := range g.audioEngine.Music.Snapshot().Tracks {
				snapshot.MusicDuration[voice] = track.Duration
				snapshot.MusicSequence[voice] = [2]int{track.Sequence.ID, track.Sequence.Step}
				snapshot.MusicEnvelope[voice] = [2]int{track.Envelope.ID, track.Envelope.Step}
			}
		}
		g.ballMaintenanceSnapshots = append(g.ballMaintenanceSnapshots, snapshot)
		g.ballPreparationTasks = append(g.ballPreparationTasks, BallPreparationTask{Stage: 255,
			Frame: g.Frame, EndFrame: g.Frame, Position: work, EndPosition: work,
			Counter: g.displayCounter, Work: b.preparation.objectWork})
	}
	g.advanceBallPreparation(inputs, work)
}

// The original packed painter clips its eleven rows at the bottom edge.
// Painting and restoration share this geometry; contact clipping is separate.
func ballPictureRows(ball Ball) int {
	x := int(ball.source.X>>8) & 248
	y := int(ball.source.Y>>8) + 96
	if x < 8 || x >= 176 || y >= Height {
		return 0
	}
	return min(11, Height-y)
}

type ballPackedOperationKind byte

const (
	ballPackedRestore ballPackedOperationKind = iota
	ballPackedPaint
	ballPackedContact
)

// ballPackedOperation translates the complete packed-loop branch geometry.
// Its nominal cycles come from unchanged original instructions and remain
// separate from the physical preparation allowance.
func ballPackedOperation(kind ballPackedOperationKind, state originalBallState) (rows, cycles int) {
	top := int(state.Y>>8) + 96
	if kind == ballPackedContact {
		top &= 255
	}
	empty := [...]int{356, 384, 374}
	outside := [...]int{656, 654, 644}
	if top >= Height {
		return 0, empty[kind]
	}
	rows = min(11, Height-top)
	aligned := int(state.X>>8) & 248
	if aligned < 8 || aligned >= 176 {
		cycles = outside[kind]
		if top+11 > Height {
			cycles += 14
		}
		return 0, cycles
	}
	shift := int(state.X>>7) & 15
	switch kind {
	case ballPackedRestore:
		cycles = 674 + rows*380 + rows*shift*2
	case ballPackedPaint:
		cycles = 742 + rows*416 + rows*shift*8
	case ballPackedContact:
		cycles = 976 + rows*182 + rows*shift*2
	}
	if rows == 11 {
		cycles -= 14
	}
	if aligned == 8 {
		if kind == ballPackedRestore {
			cycles += 34
		} else {
			cycles += 26
		}
	} else if aligned == 168 {
		if kind == ballPackedRestore {
			cycles += 42
		} else {
			cycles += 44
		}
	}
	return rows, cycles
}

// ballMotionInstructionCycles describes the nominal branch work of one
// movement operation. The values are retained as preparation evidence; they
// do not advance a processor or act as elapsed display clocks.
func ballMotionInstructionCycles(state originalBallState, soundCooldown [2]byte) int {
	velocityY := state.VY + int16(state.Gravity)
	positionY := state.Y + uint16(velocityY)
	bounced := byte(positionY>>8) >= state.Floor
	if !bounced && byte(positionY>>8) >= 104 {
		return 410
	}

	velocityX := state.VX
	cycles := 466
	if bounced {
		// A zero reflected velocity skips the bounce counter and sound gate.
		cycles = 852
		if velocityY != 0 {
			switch {
			case soundCooldown[0] != 0:
				cycles = 902
			case soundCooldown[1] != 0:
				cycles = 926
			default:
				// This includes the enabled floor-sound request measured by the
				// physical operation capture.
				cycles = 1108
			}
		}
		if state.Flags&8 != 0 {
			cycles += 126
		}
		if state.Flags&32 != 0 {
			if state.Flags&16 != 0 {
				cycles += 176
			} else {
				cycles += 140
			}
		} else if state.Flags&16 != 0 {
			cycles += 170
		}
		if state.Flags&4 != 0 {
			cycles += 138
			velocityX >>= 1
		}
	}

	positionX := state.X + uint16(velocityX)
	if integerX := byte(positionX >> 8); integerX >= 183 {
		cycles += 60
	} else if integerX < 3 {
		cycles += 76
	}
	return cycles
}

// ballSpinInstructionCycles retains the two directional byte-range branches
// and the optional flashing-color update as ordinary typed work.
func ballSpinInstructionCycles(frame int, velocity int16, flags, vbl byte) int {
	value := byte(frame)
	cycles := 162
	if velocity < 0 {
		value--
		cycles = 146
		if int8(value) < 0 {
			cycles = 176
		}
	} else {
		value++
		if int8(value) >= 9 {
			cycles = 182
		}
	}
	if flags&32 != 0 {
		cycles += 40
		if vbl&7 == 0 {
			cycles += 52
		}
	}
	return cycles
}

// ballGroundInstructionCycles retains the four-slot ground pass and the
// authored eight-row shadow rotation for every visible projectile.
func ballGroundInstructionCycles(balls []Ball) int {
	cycles := 806
	for id := 0; id < min(4, len(balls)); id++ {
		x := byte(balls[id].source.X >> 8)
		y := byte(balls[id].source.Y >> 8)
		if x >= 8 && x < 176 && y >= 78 && y <= 85 {
			cycles += 1272 + 32*int(x&7)
		}
	}
	return cycles
}

// ballDisplayNeedsNextUpdate resolves the narrow upper-window boundary with
// the complete typed object work. Flat row allowances otherwise place several
// distinct passes at the same apparent position around the line edge.
func (g *Game) ballDisplayNeedsNextUpdate(position int) bool {
	limit := 251 * picturePreparationQuota / 313
	if position < limit {
		return false
	}
	w := &g.Bonus.preparation
	cycles := w.packedInstructionCycles + w.motionInstructionCycles +
		w.spinInstructionCycles + w.groundInstructionCycles
	if w.integrations == 4 {
		if position > ballMultipleObjectWindowMax {
			return true
		}
		return cycles < ballFourObjectLongPass
	}
	if w.integrations == 3 && !g.Bonus.failed && g.Bonus.Deflected > 0 && g.Bonus.launchRate < ballEarlyThreeObjectMaxSpeed {
		if position > ballMultipleObjectWindowMax {
			return true
		}
		return cycles >= ballThreeObjectLongPass
	}
	if w.integrations == 2 && g.ambient.Mode == 3 && position <= ballRippleTwoObjectWindowMax &&
		cycles >= ballRippleTwoObjectLongPass && cycles < ballRippleTwoObjectPassMax {
		return false
	}
	if position > ballSingleObjectWindowMax {
		return true
	}
	if g.Bonus.failed && w.contactInstructionCycles >= 9000 {
		return false
	}
	if w.paintInstructionCycles < 3000 && w.motionInstructionCycles >= 900 && g.OriginalVBL()&3 == 0 {
		return false
	}
	return w.paintInstructionCycles < 3000 || cycles >= ballSingleObjectLongPass
}

// Contact scans retain the authored byte-sized vertical coordinate. Even a
// rejected saved picture visits the contact routine after the shield is lost.
// Painting and restoration keep their separate, wider clipping coordinates.
func ballContactPreparationWork(s originalBallState) (rows, work int) {
	rows, _ = ballPackedOperation(ballPackedContact, s)
	top := (int(s.Y>>8) + 96) & 255
	if rows == 0 && top >= Height {
		return 0, 173
	}
	x := int(s.X>>8) & 248
	if x < 8 || x >= 176 {
		return 0, 300
	}
	rows = min(11, Height-top)
	shift := int(s.X>>7) & 15
	return rows, 450 + rows*85 + rows*shift*picturePreparationQuota/(227*313)
}

func (g *Game) beginBallRestore() {
	g.Bonus.preparation.stage = ballWorkRestore
	g.Bonus.preparation.debt = g.ballRestoreWork()
	g.Bonus.preparation.randomFrames = nil
	g.Bonus.preparation.actorErases = nil
}

// Body strips precede the final ground-shadow work. Retain their completed
// rows before the next projectile restoration. Duration weights remain
// provisional and are checked by the complete picture-comparison gates.
func (g *Game) ballActorRowsPainted() int {
	w := &g.Bonus.preparation
	f := g.Fighters[g.Bonus.Fighter]
	rows, _ := authoredPictureRows(f.SpriteFrame)
	if w.stage > ballWorkActor {
		return rows
	}
	work := w.actorWork - w.debt - (actorPreparationSetupWork - actorGroundShadowWork)
	painted := 0
	alignment := int(f.X/2) & 7
	for _, run := range fighterArtwork.Fighters[f.SpriteFrame].Runs {
		work -= 100
		for range run.Rows {
			rowWork := 150 + ((painted+1)*3/2 - painted*3/2) +
				ballActorRotationWork((painted+1)*3, alignment) - ballActorRotationWork(painted*3, alignment)
			if work < rowWork {
				return painted
			}
			work -= rowWork
			painted++
		}
	}
	return painted
}

// Body rows rotate three words; the eight shadow rows rotate forty more.
// Keep their alignment work in the queue's ordinary fixed update scale.
func ballActorRotationWork(rotations, alignment int) int {
	return rotations * alignment * 2 * picturePreparationQuota / (227 * 313)
}

func (g *Game) rememberBallActorErase(ball Ball, bank int) {
	if !g.pacedBallPreparation() || ballPictureRows(ball) == 0 {
		return
	}
	w := &g.Bonus.preparation
	_, preparing := g.SceneryBuffers()
	if bank != preparing || w.stage < ballWorkActor || w.stage > ballWorkShadows {
		return
	}
	w.actorErases = append(w.actorErases, BallActorErase{ball, g.ballActorRowsPainted()})
}

func (g *Game) ballControlWork() int {
	if g.Bonus.failed {
		// C3EC returns immediately after testing the failed-bonus flag.
		return 34
	}
	return 280
}

// Each original shield board submits 23 rows; the neutral pose submits two.
func shieldPictureRows(x, y uint16) int {
	aligned := int(x>>8) & 248
	top := int(y>>8) + 96
	if aligned < 8 || aligned >= 176 || top >= Height {
		return 0
	}
	return min(23, Height-top)
}

// Work follows the actual packed rows and horizontal rotation. The calibrated
// rates remain provisional until both complete picture timelines pass.
func (g *Game) ballShieldWork() int {
	b := &g.Bonus
	if !b.ShieldVisible || b.shieldMotion.Flags == 0 {
		return 40
	}
	work := 140
	boards := 1
	if g.Fighters[b.Fighter].ShieldType == 1 {
		boards = 2
	}
	for board := 0; board < boards; board++ {
		x := b.shieldMotion.X + uint16(board*8<<8)
		rows := shieldPictureRows(x, b.shieldMotion.Y)
		if rows == 0 {
			work += 160
			continue
		}
		shift := int(x>>7) & 15
		work += 565 + rows*205 + rows*shift*37/10
	}
	return work
}

// Retain fractional copy credit across regions, updates and drawing tasks.
// Splitting a task must not repeatedly round away completed work.
func (w *ballPreparationWork) advanceDebt(position, limit, rate int) int {
	return advancePreparationDebt(&w.debt, &w.copyRemainder, position, limit, rate)
}

func (g *Game) advanceBallPreparation(inputs [2]Input, position int) {
	b := &g.Bonus
	w := &b.preparation
	for position < picturePreparationQuota && g.pacedBallPreparation() {
		g.observeBallPreparationTask(position)
		if w.debt > 0 {
			for w.debt > 0 && position < picturePreparationQuota {
				drawingRate := 0
				if w.stage == ballWorkActor || w.stage == ballWorkReferee || w.stage == ballWorkRestore || w.stage == ballWorkShield || w.stage == ballWorkExit {
					drawingRate = 1055
					if w.stage == ballWorkRestore || w.stage == ballWorkExit {
						drawingRate = 1070
					}
				}
				limit, rate := preparationWorkRate(position, drawingRate)
				position = w.advanceDebt(position, limit, rate)
			}
			if w.debt != 0 {
				return
			}
		}
		switch w.stage {
		case ballWorkEntry:
			g.beginBallRestore()
		case ballWorkRestore:
			w.stage = ballWorkCaption
			g.caption.state.Alternate = g.GermanText()
			previous := g.caption.state.Previous
			visible := g.caption.state.Visible
			g.caption.prepare(g.caption.state.Message, g.captionContext())
			if previous != g.caption.state.Previous {
				g.eraseCaptionPictures(g.caption.state.Anchor)
				w.debt = 18800
				if !visible && g.caption.state.Visible {
					w.debt += 4600
				}
			}
			w.debt += captionTextPreparationWork(g.Caption())
		case ballWorkCaption:
			if !g.caption.beginUpdate(g.caption.state.Message, g.captionContext()) {
				w.stage, w.debt = ballWorkControl, g.ballControlWork()
				continue
			}
			w.stage = ballWorkReferee
			w.debt = actorPreparationWork(int(g.caption.state.RefereeFrame), int(g.caption.state.RefereeX))
		case ballWorkReferee:
			w.stage = ballWorkSparkles
			w.sparkles = g.caption.prepareSparkles()
		case ballWorkSparkles:
			done := w.sparkles.advanceSlots(&g.caption, func() (byte, bool) {
				if position+captionSparkleRandomWork > picturePreparationQuota {
					return 0, false
				}
				position += captionSparkleRandomWork
				w.randomFrames = append(w.randomFrames, g.OriginalVBL())
				return g.originalRandomByte(), true
			}, func(slot int, draw bool) bool {
				if w.slotDebt == 0 {
					w.slotDebt = g.caption.sparkleSlotWork(slot, draw)
				}
				used := min(w.slotDebt, picturePreparationQuota-position)
				position += used
				w.slotDebt -= used
				return w.slotDebt == 0
			})
			if !done {
				return
			}
			w.stage, w.debt = ballWorkControl, g.ballControlWork()
		case ballWorkControl:
			f := &g.Fighters[b.Fighter]
			if b.clearTicks > 0 && b.turn == 1 && !b.failed {
				// Actor-free clearing still prepares the hidden board position.
				// It follows restoration rather than being set at scene entry.
				g.prepareClearingShieldPosition()
			}
			if b.clearTicks == 0 && !b.failed {
				// Successful final clearing still publishes the selected actor
				// through live shield control, although the board is hidden.
				f.Age++
				b.contactFighter, b.maskReady = *f, true
				g.tickShield(b.Fighter, g.bonusInput(inputs))
				g.syncShieldMotionFromPose()
			}
			w.stage, w.debt = ballWorkActor, actorPreparationWork(f.SpriteFrame, int(f.X/2))
			rows, _ := authoredPictureRows(f.SpriteFrame)
			alignment := int(f.X/2) & 7
			w.debt += ballActorRotationWork(actorRotationWords(f.SpriteFrame, true), alignment) - rows*alignment*4
			if f.SpriteFrame >= 36 && f.SpriteFrame <= 47 && f.Stun == 0 && f.recoveryWait == 0 && !f.Puff {
				// The source effect painters return when their timers are empty.
				w.debt -= 1250
			}
			if f.SpriteFrame != 87 {
				w.debt += 180
			}
			w.actorWork = w.debt
		case ballWorkActor:
			f := g.Fighters[b.Fighter]
			w.shieldPicture = shieldPicture{f.ShieldX, f.ShieldY, f.ShieldType, f.ShieldAlternate, b.ShieldVisible}
			w.stage, w.debt = ballWorkShield, g.ballShieldWork()
		case ballWorkShield:
			w.stage = ballWorkShadows
			if g.caption.state.Visible {
				w.debt = 3200
			}
		case ballWorkShadows:
			g.rememberPicture()
			_, bank := g.SceneryBuffers()
			picture := &g.pictures[bank]
			f := &picture.Fighters[picture.Active]
			f.ShieldX, f.ShieldY = w.shieldPicture.x, w.shieldPicture.y
			f.ShieldType, f.ShieldAlternate = w.shieldPicture.kind, w.shieldPicture.alternate
			picture.ShieldVisible = w.shieldPicture.visible
			picture.BallActorErases = append([]BallActorErase(nil), w.actorErases...)
			picture.restoreRows, picture.restoreRuns = pictureRestorationGeometry(*picture)
			timer := b.introIRQTimer
			if g.Phase == BonusEnd {
				timer = b.resultIRQTimer
			}
			g.ballPreparedPictures = append(g.ballPreparedPictures, BallPreparedPicture{g.Frame, position, g.displayCounter, g.random, g.Fighters[b.Fighter], g.Caption(), timer, b.shieldMotion.Flags, append([]byte(nil), w.randomFrames...), g.Fighters[b.Fighter].computerAnimation.Pending})
			g.displayElapsed = 20
			w.stage, w.debt = ballWorkWaitSetup, ballDisplayWaitSetupWork
		case ballWorkWaitSetup:
			w.stage = ballWorkWindow
			// The source accepts every position on line250, not just its
			// first instant. The following update is needed from line251.
			w.waitNext = g.ballDisplayNeedsNextUpdate(position)
		case ballWorkWindow:
			if w.waitNext {
				w.waitNext = false
				return
			}
			position = max(position, 210*picturePreparationQuota/313)
			w.stage, w.debt = ballWorkHeader, 10450-ballDisplayWaitSetupWork
		case ballWorkHeader:
			g.completeDisplayWait()
			b.contactFighter, b.maskReady = g.Fighters[b.Fighter], true
			if b.clearTicks > 0 {
				b.clearTicks--
				if b.clearTicks == 0 {
					w.stage, w.debt = ballWorkInitialize, 50900
				} else {
					g.beginBallRestore()
				}
				continue
			}
			if b.finalClearing {
				if w.finalHolds {
					b.resultStage = bonusResultScenery
					w.stage, w.debt = ballWorkExit, bonusSceneryClearWork
				} else {
					w.finalHolds = true
					g.beginBallRestore()
				}
				continue
			}
			if !b.live && b.introIRQTimer == 0 {
				w.stage, w.debt = ballWorkActivate, 49800
				continue
			}
			if g.Phase == BonusEnd {
				if b.resultIRQTimer == 0 {
					if b.resultStage == bonusResultMessage {
						b.resultStage, b.resultIRQTimer = bonusResultTail, 50
						g.caption.state.Message = 255
					} else if len(b.queue) > 1 {
						g.afterBonus()
						return
					} else {
						b.startResultFrames(bonusResultHold, 40, g.OriginalVBL())
						w.stage = ballWorkHold
						continue
					}
				}
				g.beginBallRestore()
				continue
			}
			if b.live && b.failed && b.activeCount == 0 && b.deathVisible {
				if len(b.queue) > 1 {
					g.afterBonus()
					return
				}
				g.finishBonus(false)
				w.stage = ballWorkHold
				continue
			}
			if b.live && !b.failed && b.activeCount == 0 && b.spawnDone {
				g.finishBonus(true)
			}
			if b.failed && b.live {
				w.stage, w.debt = ballWorkDeath, g.ballDeathPreparationWork()
				continue
			}
			g.beginBallRestore()
		case ballWorkDeath:
			g.preparePacedBallDeath()
			g.beginBallRestore()
		case ballWorkInitialize:
			g.setupBonusActor()
			g.beginBallRestore()
		case ballWorkActivate:
			b.live, b.introRemaining = true, 0
			g.caption.state.Message = 255
			g.beginBallRestore()
		case ballWorkHold:
			if !b.advanceResultFrames(g.OriginalVBL()) {
				return
			}
			b.finalClearing, b.ShieldVisible = true, false
			g.caption.state.Message = 255
			for id := range g.Fighters {
				f := &g.Fighters[id]
				f.SpriteFrame, f.X = 87, 272
				f.computerAnimation.Frame, f.computerAnimation.Position = 87, 136
			}
			g.beginBallRestore()
		case ballWorkExit:
			// Clear both scene regions and collision layers before stopping
			// projectiles. The final sampled update follows this completed work.
			g.clearArenaPictures()
			w.objectsStopped = true
			b.startResultFrames(bonusResultFinalFrame, 1, g.OriginalVBL())
			w.stage = ballWorkExitHold
		case ballWorkExitHold:
			if !b.advanceResultFrames(g.OriginalVBL()) {
				return
			}
			g.rememberBonusProjectiles()
			// Source scenery resumes during ready preparation before the
			// following update samples fighter placement randomness.
			g.roundPreparation = roundPreparationBonusReturn
			g.Phase, g.PhaseTicks, g.Message = Ready, 0, "READY"
			return
		}
	}
}

// The header already includes the settled fallen-actor path. Advancing the
// authored reaction adds selection, publication, motion and optional speech.
func (g *Game) ballDeathPreparationWork() int {
	b := &g.Bonus
	f := &g.Fighters[b.Fighter]
	s := &f.computerAnimation
	if b.deathStarted && s.Fallen != 0 {
		return 0
	}
	work := 665
	if f.Facing < 0 {
		work += 12
	}
	next := s.Next
	if !b.deathStarted {
		work += 214
		next = nativeGameplay.Animation.Starts[b.failureAction]
	}
	frame := nativeGameplay.Animation.Frames[next]
	if frame == 42 {
		work += 14
	} else if frame == 46 {
		work -= 5
	}
	if nativeGameplay.Animation.Flags[next]&32 != 0 {
		work += 25
	}
	return work
}

func (g *Game) preparePacedBallDeath() {
	b := &g.Bonus
	if !b.deathStarted {
		b.deathStarted = true
		g.soundDown[b.Fighter] = 1
		f := &g.Fighters[b.Fighter]
		f.Action, f.Move, f.reactionAction = b.failureAction, Knockdown, b.failureAction
		g.resetSourceFighter(b.Fighter)
		f.computerAnimation.Pending = byte(b.failureAction)
	}
	g.advanceBonusDeathActor(5)
	b.deathVisible = true
}
