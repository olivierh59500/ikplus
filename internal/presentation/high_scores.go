package presentation

// highScoreTable owns the six parallel, 51-byte fields at $98E. The original
// ranks fifty entries and uses the final slot for a candidate score/name.
type highScoreTable struct{ planes *sceneryPlanes }

// sort translates the stable descending bubble pass at $AD16. Equal scores
// keep their existing order, including their names and candidate flags.
func (h highScoreTable) sort() {
	s := &h.planes.state
	s.SortRemaining = 50
	for count := 50; count > 0; count-- {
		s.RowCursor = 0
		for a := 0; a < count; a++ {
			b := a + 1
			if s.ScoreHigh[a] < s.ScoreHigh[b] || s.ScoreHigh[a] == s.ScoreHigh[b] && s.ScoreLow[a] < s.ScoreLow[b] {
				h.swapRows(a, b)
			}
			s.RowCursor++
		}
		s.SortRemaining--
	}
}

func (h highScoreTable) swapRows(left, right int) {
	s := &h.planes.state
	for _, field := range []*[51]byte{&s.ScoreHigh, &s.ScoreLow, &s.ScoreCandidate, &s.NameFirst, &s.NameSecond, &s.NameThird} {
		field[left], field[right] = field[right], field[left]
	}
	s.DigitBlank = 0
}

// glyph translates $A5E8/$A5D6's five-row font. The outline is ORed into all
// four planes, while only the authored glyph bits replace existing artwork.
func (h highScoreTable) glyph(bank *packedBitmap, glyph, column int, both bool) {
	for row := 0; row < 5; row++ {
		h.glyphRow(bank, glyph, column, row, both)
	}
	h.planes.state.GlyphCounter = 0xffff
}

// glyphRow paints one authored font row without advancing the caller's
// layout. A retained table preparation can stop between any two rows.
func (h highScoreTable) glyphRow(bank *packedBitmap, glyph, column, row int, both bool) {
	p := h.planes
	position := (int(p.state.FontPosition) + (column & 255)) & 1023
	offset := position/40*1280 + position%40 + int(int16(p.state.FontOffset))
	color := p.state.FontColor & 15
	ink := p.artwork.ScoreFont[glyph&63][row]
	outline := (ink>>1 | ink<<1 | ink) ^ ink
	outline &= byte(p.state.FontOutline)
	if p.hintStencil != nil {
		for plane := 0; plane < 4; plane++ {
			p.hintStencil[offset+row*160+plane*40] |= ink | outline
		}
	}
	for plane := 0; plane < 4; plane++ {
		at := offset + row*160 + plane*40
		value := bank.pixel(at)&^ink | outline
		if color&(1<<plane) != 0 {
			value |= ink
		}
		bank.putPixel(at, value)
		if both {
			p.pictures[1].putPixel(offset+row*160+plane*40, value)
		}
	}
}

// draw translates $AB94's six-row window, header and belt labels. Its row
// selector is zero based; the final window can contain fewer than six rows.
func (h highScoreTable) draw(bank *packedBitmap) {
	plan := h.prepare()
	plan.advance(h.planes, bank, len(plan.glyphs)*5)
}

func (h highScoreTable) walk(draw func(glyph, column int)) {
	p := h.planes
	p.state.FontOutline = 0xffff
	p.state.FontOffset = 0
	p.state.FontPosition = 0x28a
	p.state.FontColor = 4
	language := 0
	if p.state.German != 0 {
		language = 1
	}
	for column := 0; column < 27; column++ {
		if ch := p.artwork.Score.Headers[language][column]; ch != 0x40 {
			draw(int(byte(ch-0x30)), column)
		}
	}
	p.state.FontColor = 9
	p.state.FontPosition = 0x2b1
	first := int(p.state.FirstScore)
	p.state.RowCursor, p.state.RankIndex = 0, byte(first)
	p.state.RankDigits = byte(first/10*16 + first%10)
	for line := 0; line < 6 && first+line < 50; line++ {
		index := first + line
		column, blank := 1, 16
		p.state.DigitBlank = 16
		digits := func(value byte) {
			p.state.DigitValue = value
			for _, digit := range []byte{value >> 4, value & 15} {
				glyph := int(digit)
				if digit == 0 {
					glyph = blank
				} else {
					blank = 0
					p.state.DigitBlank = 0
				}
				draw(glyph, column)
				column++
			}
		}
		rank := index + 1
		digits(byte(rank/10*16 + rank%10))
		for letter, field := range []*[51]byte{&p.state.NameFirst, &p.state.NameSecond, &p.state.NameThird} {
			draw(int(field[index]), 7+letter)
		}
		score := uint16(p.state.ScoreHigh[index])<<8 | uint16(p.state.ScoreLow[index])
		belt := 0
		for belt < 23 && int16(score) >= int16(p.artwork.Score.BeltThresholds[belt]) {
			belt++
		}
		for letter := 0; letter < 8; letter++ {
			draw(int(byte(p.artwork.Score.BeltLabels[language][belt][letter]-0x30)), 12+letter)
		}
		column, blank = 21, 16
		p.state.DigitBlank = 16
		digits(p.state.ScoreHigh[index])
		digits(p.state.ScoreLow[index])
		digits(0)
		draw(0, column-1)
		p.state.FontPosition = p.state.FontPosition + 40
		p.state.RowCursor++
		p.state.RankIndex++
		p.state.RankDigits = byte(rank/10*16 + rank%10)
	}
}
