package presentation

import (
	"github.com/olivierh59500/ikplus/internal/game"
)

const (
	// Preparation continues for seven updates after the two-frame entry wait.
	titlePreparationFrames = 7
	// A common denominator carries progress between five- and six-frame
	// animation intervals without rounding each phase change to a full frame.
	titlePictureUnits = 30
	// Authored row equivalents available to actor preparation on each update.
	titleActorPreparationQuota = 358
	titleScrollingActorQuota   = 313
)

// titleSequence owns the title animation, particles and retained actor pictures.
// Ordinary Go phases and bounded preparation work determine publication.
type titleSequence struct {
	effects        titleEffects
	actors         *game.TitleActors
	rgba           []byte
	random         uint64
	frame          byte
	mainFrame      byte
	entry          int
	running, ended bool
	// Measured title motion uses five- and six-update picture intervals.
	// Historical wait-only fixtures retain their atomic update mode.
	paced                bool
	initializationWait   int
	pictureProgress      int
	picturePending       bool
	exitWait             int
	particleIndex        int
	particleOrigin       int
	particleX            uint16
	particleHalf         bool
	particleWordPhase    int
	actorWork            *game.TitleActorWork
	nextActor            int
	actorPaintDebt       int
	actorPreparationDone bool
	actorStartFrame      byte
}

func newTitleSequenceFromPlanes(p *sceneryPlanes, seed uint64, frame byte) *titleSequence {
	t := &titleSequence{random: seed, frame: frame, mainFrame: frame, effects: titleEffects{planes: p}}
	p.state.Mode, p.state.Frame, p.state.AmbientMode = 8, frame, 0
	t.effects.random = t.randomWord
	p.state.TitleStage, p.state.TitleAdvance, p.state.TitleEnding = 0, 0, 0
	p.state.TitleActorsEnabled, p.state.TitleExitTimer, p.state.TitlePicture = 0, 0, 0
	for i := range p.state.TitlePalette {
		p.state.TitlePalette[i] = p.artwork.Title.Black[i]
	}
	return t
}

func (t *titleSequence) randomWord() uint16 {
	if io := t.effects.planes.presentation; io != nil && io.randomWord != nil {
		value := io.randomWord()
		p := t.effects.planes
		t.random = uint64(p.state.RandomCounter)<<48 | uint64(p.state.RandomMultiplier)<<32 | uint64(p.state.RandomAddition)<<16 | uint64(p.state.RandomXor)
		return value
	}
	value, next := game.RandomWord(t.random, t.frame, t.effects.planes.state.SampleRandom)
	t.random = next
	p := t.effects.planes
	p.state.RandomCounter = uint16(next >> 48)
	p.state.RandomMultiplier = uint16(next >> 32)
	p.state.RandomAddition = uint16(next >> 16)
	p.state.RandomXor = uint16(next)
	return value
}

func (t *titleSequence) initializePictures() {
	p := t.effects.planes
	// The source clears exactly 32000 bytes in each picture, retaining the
	// following 768 bytes. The mask loop includes its final longword.
	p.clearTitlePictures()
	t.beginParticlePreparation()
	for t.particleIndex < 300 {
		t.prepareParticleWords(264)
	}
	t.finishPicturePreparation()
}

func (t *titleSequence) beginParticlePreparation() {
	p := t.effects.planes
	selector := p.state.ParticleOrigin
	p.state.ParticleOrigin = selector + 1
	t.particleOrigin = int(selector & 3)
	t.particleIndex, t.particleWordPhase, t.particleHalf = 0, 0, false
}

// prepareParticleWords is an ordinary bounded preparation queue. Retaining a
// candidate's first word lets a frame boundary fall between its X and Y draws.
// No instruction costs or interrupt state participate in this operation.
func (t *titleSequence) prepareParticleWords(limit int) {
	p := t.effects.planes
	signed := func(value uint16) (int16, uint16) {
		if int16(value) < 0 {
			value |= 0xff00
			return int16(value), uint16(-int16(value))
		}
		value &= 255
		return int16(value), value
	}
	for word := 0; word < limit && t.particleIndex < 300; word++ {
		particle := &p.state.Particles[t.particleIndex]
		if !t.particleHalf {
			offset := (299 - t.particleIndex) & 6
			origin := p.artwork.Title.ParticleOrigins[t.particleOrigin]
			particle.X = origin[offset/2]
			particle.Y = origin[offset/2+4]
			t.particleX = t.randomWord()
			t.particleHalf = true
			continue
		}
		y := t.randomWord()
		t.particleHalf = false
		vx, ax := signed(t.particleX)
		vy, ay := signed(y)
		// Acceptance uses the signed low word of the original product.
		if ax|ay < 32 || int16(ax*ay) > 0x2000 {
			continue
		}
		particle.VX = uint16(vx >> 1)
		particle.VY = uint16(vy >> 1)
		particle.PreviousX = 0
		particle.PreviousY = 0
		t.particleIndex++
	}
}

func (t *titleSequence) finishPicturePreparation() {
	p := t.effects.planes
	for i := range p.state.TitlePalette {
		p.state.TitlePalette[i] = p.artwork.Title.Black[i]
	}
	p.titleLogo()
	p.titleRemoveFourthPlane()
	p.state.TitleAdvance = 1
}

// Step advances one supplied VBL and then lets the blocked original main
// thread continue. The owner retains exact PAL cadence and audio scheduling.
func (t *titleSequence) Step() {
	if t.ended {
		return
	}
	t.frame++
	p := t.effects.planes
	wait := game.WaitNextFrame
	if t.entry >= 2 && !t.running {
		wait = game.WaitTitleActors
	}
	p.setMainWait(wait)
	p.state.Frame = t.frame
	if t.frame == 0 {
		p.state.FrameWraps++
	}
	if p.state.MessageTimer != 0 {
		p.state.MessageTimer--
	}
	p.publishDisplay()
	if io := p.presentation; io != nil && io.beginUpdate != nil {
		io.beginUpdate()
	}
	if p.restartRequested() {
		return // $667A branches to $17A0 before the old title IRQ resumes.
	}
	t.effects.paletteMode = paletteTitle // The surrounding title VBL installs its base list.
	if p.state.BlackScreen != 0 {
		// $FE2 publishes the black palette and leaves the IRQ before the
		// title effects. Its main thread still observes the changed VBL.
		for index := range t.effects.hardware {
			t.effects.hardware[index] = p.artwork.Title.Black[index]
		}
	} else {
		t.effects.tick(t.frame)
	}
	if p.state.Frozen != 0 {
		return
	}
	if (t.entry < 2 || t.running) && t.frame == t.mainFrame {
		return // $774A compares the live byte, including a complete frozen wrap.
	}
	if t.entry < 2 {
		t.mainFrame = t.frame
		t.entry++
		if t.entry == 2 {
			if t.paced {
				p.clearTitlePictures()
				t.beginParticlePreparation()
				t.initializationWait = titlePreparationFrames
			} else {
				t.initializePictures()
			}
		}
		return
	}
	if t.initializationWait != 0 {
		t.initializationWait--
		switch t.initializationWait {
		case 5:
			// The first partial preparation frame contains 146 word draws.
			t.prepareParticleWords(146)
		case 1, 2, 3, 4:
			// The observed full-frame preparation rate is 790 words over
			// three updates. Carry its fraction rather than rounding it.
			t.particleWordPhase += 790
			words := t.particleWordPhase / 3
			t.particleWordPhase %= 3
			t.prepareParticleWords(words)
		case 0:
			if t.particleIndex < 300 {
				t.prepareParticleWords(264)
				t.initializationWait = 1
			} else {
				t.finishPicturePreparation()
			}
		}
		return
	}
	if t.exitWait != 0 {
		t.exitWait--
		t.ended = t.exitWait == 0
		return
	}
	if t.picturePending {
		if !t.actorPreparationDone {
			t.advanceActorPreparation()
		}
		if t.ended || t.exitWait != 0 {
			t.picturePending = false
			return
		}
		t.pictureProgress += titlePictureUnits / t.effects.pictureFrames()
		if t.actorPreparationDone && t.pictureProgress >= t.effects.pictureThreshold() {
			t.finishActorPicture()
			t.pictureProgress -= titlePictureUnits
			t.picturePending = false
			t.mainFrame = t.frame
		}
		return
	}
	if !t.running {
		if p.state.TitleActorsEnabled == 0 {
			return
		}
		var positions, facing [3]byte
		for id := range positions {
			positions[id] = p.state.Actors[id].Position
			facing[id] = p.state.Actors[id].Facing
		}
		t.actors = game.NewTitleActors(positions, facing, p.state.Round)
		for id, actor := range t.actors.States() {
			t.publishActor(id, actor)
		}
		t.running = true
	}
	if t.paced {
		if t.beginPacedActorPicture() {
			t.picturePending = true
			t.pictureProgress += titlePictureUnits / t.effects.pictureFrames()
		}
	} else {
		t.prepareActors()
	}
	t.mainFrame = t.frame
}

func (t *titleSequence) publishActor(id int, actor game.TitleActorState) {
	p := t.effects.planes
	p.state.Actors[id] = actor
}

func (t *titleSequence) prepareActors() {
	if t.beginActorPicture() {
		t.finishActorPicture()
	}
}

// prepareActorHeader supplies shared strategy inputs and clears the next
// working picture before any actor starts its decision.
func (t *titleSequence) prepareActorHeader() {
	p := t.effects.planes
	// $2890 publishes the title's AI parameters before actor decisions. They
	// remain live RAM as well as arguments to the translated actor controller.
	p.state.Difficulty, p.state.Activity, p.state.TargetPolicy = 5, 5, 6
	if t.frame&8 != 0 {
		p.state.TargetPolicy = 0
	}
	p.state.Defence = 1
	if t.frame&64 != 0 {
		p.state.Defence = 5
	}
	p.state.ActorCursor = 2
	clear(p.titleMask.pixels[112*40 : 200*40])
	t.actorStartFrame = t.frame
}

func (t *titleSequence) beginActorPicture() bool {
	t.prepareActorHeader()
	p := t.effects.planes
	t.actors.StepWithPublication(t.frame, func() byte { return byte(t.randomWord()) }, t.paintTitleActor, p.state.Keys[0])
	return t.finishActorPreparation()
}

func (t *titleSequence) paintTitleActor(id int, actor game.TitleActorState) {
	p := t.effects.planes
	painter := originalScene{planes: p}

	t.publishActor(id, actor)
	coverage := true
	if p.state.TitleEnding != 0 {
		coverage = false
	} else if n := p.state.TitlePicture >> 4; n < 4 {
		coverage = id <= int(n&3)
	}
	var mask *packedBitmap
	p.state.CollisionEnabled = 0
	if coverage {
		mask = &p.titleMask
		p.state.CollisionEnabled = 1
	}
	painter.frameAt(&p.pictures[0], id, int(actor.Frame), int(actor.Position), actor.Facing != 0, mask, false)
	p.state.ActorCursor-- // $2670 descends through blue, red and white.
}

func (t *titleSequence) finishActorPreparation() bool {
	p := t.effects.planes

	if p.state.TitleEnding != 0 {
		p.state.TitleExitTimer = p.state.TitleExitTimer + 1
		if p.state.TitleExitTimer > 18 {
			if t.paced && t.actorStartFrame == t.frame {
				t.exitWait = 1
			} else {
				t.ended = true
			}
			return false
		}
	}
	return true
}

// beginPacedActorPicture prepares work in blue/red/white order. Drawing
// capacity, rather than CPU instructions or interrupts, controls when a new
// random request can join the current update.
func (t *titleSequence) beginPacedActorPicture() bool {
	t.prepareActorHeader()
	t.nextActor, t.actorPaintDebt = 2, 0
	t.actorWork, t.actorPreparationDone = nil, false
	t.advanceActorPreparation()
	return !t.ended && t.exitWait == 0
}

func (t *titleSequence) advanceActorPreparation() {
	p := t.effects.planes
	capacity := titleActorPreparationQuota
	if p.state.TitleStage == 27 {
		capacity = titleScrollingActorQuota
	}
	if t.actorPaintDebt != 0 {
		spent := min(capacity, t.actorPaintDebt)
		capacity -= spent
		t.actorPaintDebt -= spent
		if t.actorPaintDebt != 0 {
			return
		}
	}
	for t.nextActor >= 0 {
		if t.actorWork == nil {
			if capacity < 4 {
				return
			}
			capacity -= 4
			t.actorWork = t.actors.PrepareActor(t.nextActor, t.actorStartFrame, p.state.Keys[0])
		}
		ready := t.actorWork.Advance(func() (byte, bool) {
			if capacity < 2 {
				return 0, false
			}
			capacity -= 2
			return byte(t.randomWord()), true
		})
		if !ready {
			return
		}
		id, actor := t.actors.CommitActor(t.actorWork)
		t.paintTitleActor(id, actor)
		capacity -= 2 + t.actorPictureWork(int(actor.Frame), int(actor.Position))
		t.actorWork = nil
		t.nextActor--
		if capacity < 0 {
			t.actorPaintDebt = -capacity
			return
		}
	}
	t.actorPreparationDone = true
	t.finishActorPreparation()
}

// actorPictureWork counts authored row spans, run setup and horizontal word
// alignment. The measured quotas are independent of host rendering speed.
func (t *titleSequence) actorPictureWork(frame, rawX int) int {
	if frame < 0 || frame >= 88 {
		return 0
	}
	work, totalRows := 0, 0
	for _, run := range t.effects.planes.artwork.Fighters[frame].Runs {
		rows := len(run.Rows)
		work += rows + 2
		totalRows += rows
	}
	return work + totalRows*(rawX&7)/32
}

func (t *titleSequence) finishActorPicture() {
	p := t.effects.planes
	index := int(p.state.TitlePicture & 15)
	preview := &p.titleHistory[index]
	reduceTitleField(p.titleMask.pixels[112*40:200*40], preview.Large[:], p.mirroredTitleField(index, false), false)
	reduceTitleField(preview.Large[:], preview.Small[:], p.mirroredTitleField(index, true), true)
	for _, slot := range p.artwork.Title.Slots {
		frame := int(uint16(p.state.TitlePicture+uint16(slot.Delay)) & 15)
		source := p.titleHistory[frame].Small[:]
		if slot.Rows == 40 {
			source = p.titleHistory[frame].Large[:]
		}
		if slot.Mirrored {
			source = p.mirroredTitleField(frame, slot.Rows != 40)
		}
		copyTitleField(source, p.pictures[1].pixels[slot.Offset:], slot.Rows)
	}
	p.state.TitlePicture = p.state.TitlePicture + 1
}

func (t *titleSequence) stateWords() [8]uint16 {
	s := &t.effects.planes.state
	return [8]uint16{s.TitleStage, s.TitleAdvance, s.TitleEnding, s.TitleTimer, s.TitleExitTimer, s.TitleActorsEnabled, s.TitlePicture, s.ColorSequence}
}
