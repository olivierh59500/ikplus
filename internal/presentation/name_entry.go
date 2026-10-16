package presentation

import "github.com/olivierh59500/ikplus/internal/game"

// nameEntry translates $A6EA. Moving left/right with fire released cycles the
// authored space/A..Z glyphs; pressing fire confirms one character.
type nameEntry struct {
	clock     *highScoreSequence
	ended     bool
	mainFrame byte
}

func newNameEntry(p *sceneryPlanes) *nameEntry {
	h := newPresentationClock(p)
	n := &nameEntry{clock: h}
	p.state.FrameWraps, p.state.InputRepeat, p.state.NameColumn = 0, 0, 0
	p.state.Initials[0], p.state.Initials[1], p.state.Initials[2] = 12, 12, 12
	n.poll()
	p.state.PreviousFire = p.state.FireReleased
	p.state.NameGlyph = 17
	p.state.FontPosition = p.artwork.Score.NameRows[p.state.CandidateRow]
	n.resume()
	n.mainFrame = p.state.Frame
	return n
}

func (n *nameEntry) poll() {
	p := n.clock.planes
	input := p.state.Joystick[int(p.state.CandidateOwner)]
	p.state.Directions = input & 15
	p.state.FireReleased = (input>>4)&1 ^ 1
}

func (n *nameEntry) glyph(color byte) {
	p := n.clock.planes
	p.state.FontColor = color
	highScoreTable{p}.glyph(&p.pictures[0], int(p.state.NameGlyph), int(p.state.NameColumn), true)
}

func (n *nameEntry) resume() {
	p := n.clock.planes
	if int8(p.state.FrameWraps) >= 10 {
		n.finish()
		return
	}
	n.poll()
	if p.state.FireReleased != 0 {
		p.state.PreviousFire = 1
		if int8(p.state.InputRepeat) >= 5 {
			p.state.InputRepeat = 0
			if p.state.Directions&4 == 0 {
				n.glyph(15)
				p.state.NameGlyph--
				if p.state.NameGlyph < 16 {
					p.state.NameGlyph = 42
				}
			} else if p.state.Directions&8 == 0 {
				n.glyph(15)
				p.state.NameGlyph++
				if p.state.NameGlyph >= 43 {
					p.state.NameGlyph = 16
				}
			}
		}
	} else if p.state.PreviousFire != 0 {
		n.glyph(4)
		p.state.PreviousFire = 0
		p.state.Initials[int(p.state.NameColumn)] = p.state.NameGlyph
		p.state.NameGlyph, p.state.InputRepeat = 17, 0
		p.state.Frame &= 7
		if int8(p.state.NameColumn) >= 2 {
			n.finish()
			return
		}
		p.state.NameColumn++
		p.state.FrameWraps &= 3
	}
	color := byte(4)
	if p.state.Frame&31 >= 24 {
		color = 8
	}
	n.glyph(color)
}

func (n *nameEntry) finish() {
	p := n.clock.planes
	n.glyph(9)
	index := int(p.state.CandidateIndex)
	for letter, field := range []*[51]byte{&p.state.NameFirst, &p.state.NameSecond, &p.state.NameThird} {
		field[index] = p.state.Initials[letter]
	}
	n.ended = true
}

func (n *nameEntry) Step(input byte) {
	if n.ended {
		return
	}
	p := n.clock.planes
	n.clock.mainWait = game.WaitNextFrame
	p.state.Joystick[int(p.state.CandidateOwner)] = input
	n.clock.vbl()
	if p.restartRequested() || p.state.Frozen != 0 {
		return
	}
	if p.state.Frame == n.mainFrame {
		return
	}
	n.resume()
	n.mainFrame = p.state.Frame
}

// nameBorder translates $A4D0's moving perimeter. The old cells restore
// original scenery through their sprite masks before the next cells appear.
func (p *sceneryPlanes) nameBorder() {
	if p.state.BorderEnabled == 0 {
		return
	}
	paint := func(restore bool) {
		offset := int(p.state.BorderOffset)
		p.state.BorderCell = 0
		cell := func(x, y int) {
			index := (p.state.BorderFrame + p.state.BorderCell) & 7
			p.paintSpriteAt(p.pictureBuffer(p.state.Background), x, y, 3, p.artwork.Sprites[p.artwork.Border[index]], false, true, false, restore)
			p.state.BorderCell = p.state.BorderCell + 1
		}
		for i := 0; i < 4; i++ {
			cell(80+offset+i*64, 120)
		}
		for i := 0; i < 2; i++ {
			cell(336, 120+offset/2+i*32)
		}
		for i := 0; i < 4; i++ {
			cell(336-offset-i*64, 184)
		}
		for i := 0; i < 2; i++ {
			cell(80, 184-offset/2-i*32)
		}
	}
	paint(true)
	p.state.BorderOffset = (p.state.BorderOffset + 2) & 63
	if p.state.BorderOffset == 0 {
		p.state.BorderFrame = p.state.BorderFrame - 1
	}
	p.state.BorderFrame = p.state.BorderFrame & 7
	paint(false)
}
