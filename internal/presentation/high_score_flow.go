package presentation

import "github.com/olivierh59500/ikplus/internal/game"

type scoreFlowPart byte

const (
	scoreQualifiedMessage scoreFlowPart = iota
	scoreRejectedMessage
	scorePromptMessage
	scoreEntryPicture
	scoreName
	scoreConfirmationPicture
	scoreConfirmationHold
	scoreReturnPicture
	scoreReturnHold
	scoreOverflowWait
	scoreDemo
	scoreSorting
)

// highScoreFlow translates $A110's caption/name/pose/return control flow. Its
// caption and live-hint paths share the same native renderer and PAL state.
// The original destructive code-writing branch remains outside its scope.
type highScoreFlow struct {
	clock     *highScoreSequence
	caption   game.CaptionController
	painter   originalScene
	previous  [2]game.PictureEraseState
	name      *nameEntry
	part      scoreFlowPart
	frames    int
	waiting   bool
	ended     bool
	holdFrame byte
	sorting   *scoreSortPreparation
}

func newHighScoreFlow(p *sceneryPlanes) *highScoreFlow {
	return newHighScoreFlowWithHistory(p, nil)
}

func newHighScoreFlowWithHistory(p *sceneryPlanes, history *[2]game.PictureEraseState) *highScoreFlow {
	f := &highScoreFlow{clock: newPresentationClock(p)}
	f.painter = originalScene{planes: p}
	f.painter.initialize()
	f.caption = game.NewCaptionController(presentationCaption(p))
	for i := range f.previous {
		f.previous[i] = game.PictureEraseState{Valid: true, Mode: 4, Active: -1, Fighters: f.fighters(), Caption: f.caption.Snapshot(), CaptionText: f.caption.Snapshot().Visible}
	}
	if history != nil {
		f.previous = *history
	}
	if p.presentation != nil && p.presentation.pacedScores && int8(p.state.CandidateOwner) < 0 {
		f.beginScoreSort()
		return f
	}
	switch (highScoreTable{p}).prepareCandidate() {
	case scoreDisplayOnly:
		f.beginDemo(false)
	case scoreEmpty:
		f.clearHumans()
		f.beginCaption(20, 150, scorePromptMessage)
	case scoreRejected:
		f.beginCaption(19, 200, scoreRejectedMessage)
	case scoreQualified:
		f.beginCaption(18, 200, scoreQualifiedMessage)
	}
	f.resume()
	return f
}

func presentationCaption(p *sceneryPlanes) game.CaptionState {
	s := game.CaptionState{Message: p.state.Message, Previous: p.state.PreviousMessage, Width: p.state.CaptionWidth, Visible: p.state.CaptionVisible != 0, Anchor: p.state.CaptionAnchor, Column: p.state.CaptionColumn, Budget: p.state.CaptionBudget, Alternate: p.state.German != 0, Amount: p.state.BonusAmount}
	copy(s.Order[:], p.state.Order[0:3])
	copy(s.Counters[:], p.state.SparkleCounters[0:8])
	copy(s.X[:], p.state.SparkleX[0:8])
	copy(s.Y[:], p.state.SparkleY[0:8])
	return s
}

func (f *highScoreFlow) storeCaption() {
	p := f.clock.planes
	s := f.caption.Snapshot()
	p.state.Message, p.state.PreviousMessage, p.state.CaptionWidth = s.Message, s.Previous, s.Width
	p.state.CaptionVisible = 0
	if s.Visible {
		p.state.CaptionVisible = 1
	}
	p.state.CaptionAnchor, p.state.CaptionColumn, p.state.CaptionBudget = s.Anchor, s.Column, s.Budget
	copy(p.state.SparkleCounters[0:8], s.Counters[:])
	copy(p.state.SparkleX[0:8], s.X[:])
	copy(p.state.SparkleY[0:8], s.Y[:])
}

func (f *highScoreFlow) captionContext() game.CaptionContext {
	p := f.clock.planes
	ctx := game.CaptionContext{Mode: p.state.Mode, Counter: p.state.DisplayCounter, Elapsed: p.state.Elapsed, Variant: p.state.Referee}
	for id, actor := range p.state.Actors {
		ctx.Positions[id], ctx.Facing[id], ctx.Frames[id] = actor.Position, actor.Facing, actor.Frame
	}
	return ctx
}

func (f *highScoreFlow) fighters() [3]game.Fighter {
	p := f.clock.planes
	var fighters [3]game.Fighter
	for id := range fighters {
		facing := 1
		if p.state.Actors[id].Facing != 0 {
			facing = -1
		}
		fighters[id] = game.Fighter{ID: id, X: float64(p.state.Actors[id].Position) * 2, Y: game.GroundY, Facing: facing, SpriteFrame: int(p.state.Actors[id].Frame)}
	}
	return fighters
}

func (f *highScoreFlow) eraseCaptionBoth(anchor byte) {
	p := f.clock.planes
	for _, bank := range p.screens() {
		copy(p.working.pixels[:32000], bank.pixels[:32000])
		f.painter.eraseCaption(anchor)
		copy(bank.pixels[:32000], p.working.pixels[:32000])
	}
}

func (f *highScoreFlow) hideCaption() {
	f.caption.SetAlternate(f.clock.planes.state.German != 0)
	before := f.caption.Snapshot().Previous
	f.caption.Prepare(255, f.captionContext())
	f.storeCaption()
	if before != 255 {
		f.eraseCaptionBoth(f.caption.Snapshot().Anchor)
	}
}

func (f *highScoreFlow) prepareCaptionPicture() {
	p := f.clock.planes
	f.caption.SetAlternate(p.state.German != 0)
	hint := p.state.HintTimer != 0
	bank, index := p.pictureBuffer(p.state.Drawing), 0
	if bank == &p.pictures[1] {
		index = 1
	}
	copy(p.working.pixels[:32000], bank.pixels[:32000])
	if hint {
		f.clock.restoreBottom(&p.working)
	} else {
		f.painter.erasePreviousPicture(f.previous[index])
	}
	for row := 192; row < 200; row++ {
		clear(p.working.pixels[row*160 : row*160+40])
	}
	fighters := f.fighters()
	for _, id := range []int{2, int(p.state.DrawPriority&1) ^ 1, int(p.state.DrawPriority & 1)} {
		a := fighters[id]
		f.painter.frame(id, a.SpriteFrame, int(a.X/2), a.Facing < 0)
	}
	previous := f.caption.Snapshot().Previous
	f.caption.Update(p.state.Message, f.captionContext(), func() byte {
		f.storeCaption()
		return byte(f.clock.randomWord())
	})
	f.storeCaption()
	state := f.caption.Snapshot()
	if previous != state.Previous {
		copy(bank.pixels[:32000], p.working.pixels[:32000])
		f.eraseCaptionBoth(state.Anchor)
		copy(p.working.pixels[:32000], bank.pixels[:32000])
	}
	f.painter.caption(state, true)
	if hint {
		p.state.FontOutline = 0
		if int16(p.state.HintTimer) <= 20 {
			// $1EDA branches directly to $1F0E: retained short hints still
			// select the alternate picture path, but their text is not drawn.
			f.painter.projectShadows()
		} else if p.artwork.HintColors[p.state.HintIndex] != 15 {
			p.state.FontOutline = 0xffff
			f.painter.projectShadows()
			highScoreTable{p}.scoreHint(&p.working)
		} else {
			highScoreTable{p}.scoreHint(&p.working)
			f.painter.projectShadows()
		}
	} else {
		f.painter.projectShadows()
	}
	copy(bank.pixels[:32000], p.working.pixels[:32000])
	f.previous[index] = game.PictureEraseState{Valid: true, Mode: 4, Active: -1, Fighters: fighters, Caption: state, CaptionText: state.Visible}
	f.waiting = true
}

func (f *highScoreFlow) beginCaption(message, timer byte, part scoreFlowPart) {
	f.clock.planes.state.Message, f.clock.planes.state.MessageTimer = message, timer
	f.part, f.waiting = part, false
}

func (f *highScoreFlow) clearHumans() {
	p := f.clock.planes
	p.state.Humans[0], p.state.Humans[1] = 0, 0
	f.clock.hand(1, false)
	f.clock.hand(0, false)
}

func (f *highScoreFlow) beginDemo(sort bool) {
	p := f.clock.planes
	if p.presentation != nil && p.presentation.pacedScores {
		f.beginScoreSort()
		return
	}
	if sort {
		highScoreTable{p}.sort()
	}
	f.hideCaption()
	f.clock = newHighScoreSequence(p)
	f.part, f.waiting = scoreDemo, true
}

func (f *highScoreFlow) startScorePicture(part scoreFlowPart) {
	p := f.clock.planes
	p.state.ScoresVisible, p.state.FrameWraps = 1, 0
	p.state.Frame &= 15
	f.clock.paint()
	f.holdFrame = p.state.Frame
	f.part, f.waiting = part, true
}

func (f *highScoreFlow) finish() {
	p := f.clock.planes
	p.state.Mode = 8
	clear(p.state.Humans[0:3])
	p.state.ScoresVisible = 0
	f.clock.hand(0, false)
	f.clock.hand(1, false)
	f.ended = true
}

func (f *highScoreFlow) resume() {
	p := f.clock.planes
	for !f.ended {
		switch f.part {
		case scoreQualifiedMessage, scoreRejectedMessage, scorePromptMessage:
			if !f.waiting {
				f.prepareCaptionPicture()
			}
			if p.state.Elapsed <= p.artwork.Score.SpeedDelay[p.state.Speed+p.state.Fast] {
				return
			}
			f.clock.completeWait()
			f.waiting = false
			if p.state.MessageTimer != 0 {
				continue
			}
			if f.part == scoreRejectedMessage {
				f.clearHumans()
				f.beginCaption(20, 150, scorePromptMessage)
			} else if f.part == scorePromptMessage {
				f.beginDemo(true)
			} else {
				selected := int(p.state.Order[2])
				p.state.Actors[selected].Facing, p.state.Actors[selected].Position, p.state.Actors[selected].Frame = 0, 20, 48
				p.state.Actors[int(p.state.Order[0])].Frame, p.state.Actors[int(p.state.Order[1])].Frame = 87, 87
				p.state.BorderEnabled = 1
				p.state.BorderOffset = 0
				f.hideCaption()
				for _, bank := range p.screens() {
					f.clock.restoreBottom(bank)
				}
				f.startScorePicture(scoreEntryPicture)
			}
		case scoreEntryPicture, scoreConfirmationPicture, scoreReturnPicture:
			if p.state.Elapsed <= p.artwork.Score.SpeedDelay[p.state.Speed+p.state.Fast] {
				return
			}
			f.clock.completeWait()
			f.waiting = false
			if f.part == scoreEntryPicture {
				f.name = newNameEntry(p)
				f.clock, f.part = f.name.clock, scoreName
			} else if f.part == scoreConfirmationPicture {
				f.part, f.frames = scoreConfirmationHold, 40
			} else {
				f.part, f.frames = scoreReturnHold, 100
			}
			return
		case scoreName:
			if !f.name.ended {
				return
			}
			p.state.Actors[int(p.state.Order[2])].Frame = 49
			f.startScorePicture(scoreConfirmationPicture)
		case scoreConfirmationHold:
			if f.frames != 0 {
				return
			}
			p.state.Actors[int(p.state.Order[2])].Frame = 48
			f.startScorePicture(scoreReturnPicture)
		case scoreReturnHold:
			if f.frames != 0 {
				return
			}
			selected := int(p.state.Order[2])
			p.state.Humans[selected] = 0
			f.clock.hand(selected, false)
			f.part = scoreOverflowWait
		case scoreOverflowWait:
			if p.state.FrameWraps < 2 {
				return
			}
			p.state.BorderEnabled = 0
			p.state.ScoresVisible = 0
			for _, bank := range p.screens() {
				f.clock.restoreBottom(bank)
			}
			f.finish()
		case scoreDemo:
			if !f.clock.ended {
				return
			}
			f.finish()
		case scoreSorting:
			return
		}
	}
}

func (f *highScoreFlow) Step(input [2]byte) {
	if f.ended {
		return
	}
	p := f.clock.planes
	copy(p.state.Joystick[0:2], input[:])
	f.clock.mainWait = game.WaitFightPicture
	if f.part == scoreName || f.part == scoreConfirmationHold || f.part == scoreReturnHold {
		f.clock.mainWait = game.WaitNextFrame
	} else if f.part == scoreOverflowWait {
		f.clock.mainWait = game.WaitScoresExit
	}
	if f.part == scoreName {
		f.name.Step(input[p.state.CandidateOwner])
	} else if f.part == scoreDemo {
		f.clock.Step(p.state.Keys[0])
	} else {
		f.clock.vbl()
		if f.part == scoreSorting && !p.restartRequested() && p.state.Frozen == 0 {
			f.sorting.position = f.clock.updateWork
			f.sorting.advance()
		}
		if !p.restartRequested() && p.state.Frozen == 0 && p.state.Frame != f.holdFrame && (f.part == scoreConfirmationHold || f.part == scoreReturnHold) {
			f.frames--
			f.holdFrame = p.state.Frame
		}
	}
	if p.restartRequested() || p.state.Frozen != 0 {
		return
	}
	f.resume()
}
