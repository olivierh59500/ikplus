package presentation

import "github.com/olivierh59500/ikplus/internal/game"

// highScoreSequence owns scrolling scores, actor animation and scenery.
// It advances through native updates and shares typed scene state.
type highScoreSequence struct {
	planes *sceneryPlanes

	actors      *game.HighScoreActors
	ambient     game.AmbientState
	random      uint64
	phase       int
	ended       bool
	mainWait    game.WaitPoint
	preparation *scorePreparation
	updateWork  int
}

func initializeHighScoreActors(p *sceneryPlanes) *highScoreSequence {
	h := newPresentationClock(p)
	var initial [3]game.TitleActorState
	for id := range initial {
		initial[id] = readPresentationActor(p, id)
	}
	h.actors = game.NewHighScoreActors(initial, p.state.Round, p.state.Difficulty)
	h.publishActors()
	clear(p.state.SoundDown[0:3])
	p.state.ScoreForeground = byte(h.randomWord()) & 1
	h.actors.HoldWhite()
	h.publishActors()
	p.state.Elapsed, p.state.FirstScore = 10, 49
	p.state.Points[0], p.state.Points[1], p.state.Points[2] = 0, 1, 1
	p.state.TargetPolicy, p.state.Defence, p.state.Activity = 6, 2, 5
	p.state.Message = 255
	return h
}

func newHighScoreSequence(p *sceneryPlanes) *highScoreSequence {
	h := initializeHighScoreActors(p)
	for _, bank := range p.screens() {
		h.restoreBottom(bank)
	}
	p.state.ScoresVisible, p.state.FrameWraps = 1, 0
	p.state.Frame &= 15
	h.paint()
	h.resume()
	return h
}

func newPresentationClock(p *sceneryPlanes) *highScoreSequence {
	return &highScoreSequence{planes: p, mainWait: game.WaitFightPicture, ambient: presentationAmbient(p), random: uint64(p.state.RandomCounter)<<48 | uint64(p.state.RandomMultiplier)<<32 | uint64(p.state.RandomAddition)<<16 | uint64(p.state.RandomXor)}
}

func readPresentationActor(p *sceneryPlanes, id int) game.TitleActorState { return p.state.Actors[id] }
func (h *highScoreSequence) publishActors()                               { h.planes.state.Actors = h.actors.States() }

func (h *highScoreSequence) randomWord() uint16 {
	p := h.planes
	if io := p.presentation; io != nil && io.randomWord != nil {
		value := io.randomWord()
		h.random = uint64(p.state.RandomCounter)<<48 | uint64(p.state.RandomMultiplier)<<32 | uint64(p.state.RandomAddition)<<16 | uint64(p.state.RandomXor)
		return value
	}
	value, next := game.RandomWord(h.random, p.state.Frame, p.state.SampleRandom)
	h.random = next
	p.state.SetRandomState(next)
	return value
}

func (h *highScoreSequence) restoreBottom(bank *packedBitmap) {
	copy(bank.pixels[117*160:200*160], h.planes.artwork.Background[117*160:200*160])
}

func (h *highScoreSequence) paint() {
	p := h.planes
	bank := p.pictureBuffer(p.state.Drawing)
	h.restoreBottom(bank)
	painter := originalScene{planes: p}
	draw := func(id int) {
		painter.frameAt(bank, id, int(p.state.Actors[id].Frame), int(p.state.Actors[id].Position), p.state.Actors[id].Facing != 0, nil, true)
	}
	draw(1)
	draw(0)
	if p.state.ScoreForeground != 0 {
		highScoreTable{p}.draw(bank)
	}
	draw(2)
	painter.projectShadowsAt(bank)
	if p.state.ScoreForeground == 0 {
		highScoreTable{p}.draw(bank)
	}
}

// resume advances retained native preparation or the atomic supplied-wait
// comparison profile. Live drawing capacity is independent of host speed.
func (h *highScoreSequence) resume() {
	if h.preparation != nil {
		h.preparation.advance()
		return
	}
	p := h.planes
	for !h.ended && p.state.Elapsed > p.artwork.Score.SpeedDelay[p.state.Speed+p.state.Fast] {
		h.completeWait()
		if !h.advanceScoreRow() {
			return
		}
		copy(h.actors.Wait[:], p.state.Recovery[0:3])
		copy(h.actors.Rest[:], p.state.Rest[0:3])
		h.actors.Step(func() byte { return byte(h.randomWord()) }, p.state.Keys[0])
		h.publishActors()
		h.paint()
	}
}

func (h *highScoreSequence) advanceScoreRow() bool {
	p := h.planes
	if h.phase == 0 {
		p.state.Elapsed = 6
		h.phase = 1
	} else if h.phase == 1 {
		if p.state.FirstScore == 0 || p.state.Keys[0] == 0x48 || p.state.Keys[0] == 0x50 {
			h.phase = 2
			if p.state.FrameWraps >= 5 {
				h.finish()
				return false
			}
		} else {
			p.state.FirstScore--
			if p.state.FrameWraps >= 4 {
				h.finish()
				return false
			}
		}
	} else {
		if p.state.Keys[0] == 0x48 && p.state.FirstScore != 0 {
			p.state.FirstScore--
		}
		if p.state.Keys[0] == 0x50 && p.state.FirstScore < 49 {
			p.state.FirstScore++
		}
		if p.state.FrameWraps >= 5 {
			h.finish()
			return false
		}
	}
	return !h.ended
}

func (h *highScoreSequence) completeWait() {
	h.beginDisplayWait()
	h.finishDisplayWait()
}

func (h *highScoreSequence) beginDisplayWait() {
	p := h.planes
	if io := p.presentation; io != nil && io.beforeWait != nil {
		io.beforeWait()
	}
	p.state.DisplayCounter++
}

func (h *highScoreSequence) finishDisplayWait() {
	p := h.planes
	p.state.Elapsed = 0
	p.state.BlackScreen = 0
	p.state.BufferPhase = (p.state.BufferPhase ^ 4) & 4
	index := int(p.state.BufferPhase)
	if index == 0 {
		p.state.Drawing, p.state.Background = game.PictureFirst, game.PictureSecond
		p.state.CollisionRead, p.state.CollisionReadMirror = game.CoverageFirst, game.CoverageFirstMirrored
		p.state.CollisionWrite, p.state.CollisionWriteMirror = game.CoverageSecond, game.CoverageSecondMirrored
	} else {
		p.state.Drawing, p.state.Background = game.PictureSecond, game.PictureFirst
		p.state.CollisionRead, p.state.CollisionReadMirror = game.CoverageSecond, game.CoverageSecondMirrored
		p.state.CollisionWrite, p.state.CollisionWriteMirror = game.CoverageFirst, game.CoverageFirstMirrored
	}
	p.state.EraseStart = game.RestorationCursor{Buffer: p.state.Drawing}
	p.state.EraseCursor = p.state.EraseStart

	h.clock()
	h.beltColors()
}

func (h *highScoreSequence) clock() {
	p := h.planes
	bcd := func(v byte) int { return int(v>>4)*10 + int(v&15) }
	s := hudState{Round: bcd(p.state.Round), Seconds: bcd(p.state.Seconds), Mode: int(p.state.Mode), VBL: int(p.state.Frame), Counter: int(p.state.DisplayCounter)}
	if p.state.Humans[0]|p.state.Humans[1] != 0 {
		s.Humans = 1
	}
	for _, draw := range hudLayout(s)[36:] {
		if draw.Image != "" {
			continue // $6AD8 refreshes glyphs; speed and owner sprites survive.
		}
		if p.state.German != 0 && s.Humans != 0 && s.Mode != 6 && draw.Y == 0 && draw.X >= 32*8 && draw.X < 37*8 {
			draw.Glyph = int(byte(p.artwork.Score.GermanClock[draw.X/8-32]-0x30)) & 63
		}
		for row := 0; row < 8; row++ {
			mask := p.artwork.HUDFont[draw.Glyph][row]
			for plane := 0; plane < 4; plane++ {
				value := byte(0)
				if draw.Color&(1<<plane) != 0 {
					value = mask
				}
				for _, bank := range p.screens() {
					bank.pixels[(draw.Y+row)*160+draw.X/8+plane*40] = value
				}
			}
		}
	}
	p.state.ClockGlyph = 6
}

func (h *highScoreSequence) finish() {
	h.planes.state.ScoresVisible = 0
	if h.preparation != nil {
		w := h.preparation
		w.clearBank(&h.planes.pictures[0], nil)
		w.clearBank(&h.planes.pictures[1], func() {
			h.ended = true
		})
		return
	}
	for _, bank := range h.planes.screens() {
		h.restoreBottom(bank)
	}
	h.ended = true
}

// Step supplies one VBL, then resumes the blocked source main thread. The
// shared audio owner will schedule music around this IRQ in the frontend.
func (h *highScoreSequence) Step(key byte) {
	if h.ended {
		return
	}
	p := h.planes
	p.state.Keys[0] = key
	h.vbl()
	if p.restartRequested() || p.state.Frozen != 0 {
		return
	}
	h.resume()
}

func (h *highScoreSequence) vbl() {
	p := h.planes
	p.setMainWait(h.mainWait)
	p.state.Frame++
	if p.state.Frame == 0 {
		p.state.FrameWraps++
	}
	if p.state.MessageTimer != 0 {
		p.state.MessageTimer--
	}
	// $FD2/$EC4 publish the retained screen and colors before main-thread
	// drawing can change the next picture's belt palette.
	p.publishDisplay()
	if io := p.presentation; io != nil && io.beginUpdate != nil {
		io.beginUpdate()
	}
	if p.restartRequested() || p.state.Frozen != 0 {
		return
	}
	if p.state.PauseCooldown != 0 {
		p.state.PauseCooldown--
	}
	for i, v := range p.state.SoundCooldown {
		if v != 0 {
			p.state.SoundCooldown[i]--
		}
	}
	p.state.Elapsed++
	p.state.MotionTimer++
	p.state.InputRepeat++
	for id := 0; id < 3; id++ {
		if p.state.Stun[id] != 0 {
			p.state.Stun[id]--
		}
		if p.state.Recovery[id] != 0 {
			p.state.Recovery[id]--
		} else if p.state.Rest[id] != 0 {
			p.state.Rest[id]--
		}
	}
	before := h.ambient
	h.sceneryVBL()
	h.updateWork = scoreSceneUpdateWork(before, h.ambient, p.state.Frame, p.state.Frame&1 == 0 && p.waterUpdates == 0)
	p.nameBorder()
	if p.state.Frame&3 == 0 {
		for id := 1; id >= 0; id-- {
			if p.state.Humans[id] != 0 && p.state.OwnerFlash[id] != 0 {
				p.state.OwnerFlash[id]--
				h.hand(id, p.state.OwnerFlash[id]&4 == 0)
			}
		}
	}
	p.publishScoreFrame()
	if h.preparation != nil {
		h.preparation.beginUpdate()
	}
}

func (h *highScoreSequence) hand(id int, active bool) {
	p := h.planes
	pose := 0
	if active {
		pose = 1
	}
	for _, bank := range p.screens() {
		p.spriteAt(bank, p.artwork.HandX[id], 0, 15, p.artwork.Sprites[p.artwork.Hands[pose][id]], false, false, false)
	}
}

func (h *highScoreSequence) beltColors() {
	p := h.planes
	p.state.BeltColors[2] = 0xcc0
	for id := 0; id < 2; id++ {
		color := []uint16{0xc00, 0xe6e}[id]
		if p.state.Mode != 6 && p.state.Humans[id] != 0 {
			score := uint16(p.state.ScoreDigits[id*3])<<8 | uint16(p.state.ScoreDigits[1+(id*3)])
			index := 0
			for index < 23 && int16(score) >= int16(p.artwork.Score.BeltThresholds[index]) {
				index++
			}
			color = p.artwork.Score.BeltColors[id][index]
		}
		p.state.BeltColors[id] = color
	}
}
