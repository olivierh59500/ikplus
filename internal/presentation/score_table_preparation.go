package presentation

// scoreFontState retains only the painter's layout. Live frame, scenery,
// input and palette state continue independently while a picture is prepared.
type scoreFontState struct {
	position, offset, outline, counter     uint16
	color, row, rank, digits, blank, value byte
}

func readScoreFont(p *sceneryPlanes) scoreFontState {
	s := &p.state
	return scoreFontState{s.FontPosition, s.FontOffset, s.FontOutline, s.GlyphCounter,
		s.FontColor, s.RowCursor, s.RankIndex, s.RankDigits, s.DigitBlank, s.DigitValue}
}

func (font scoreFontState) apply(p *sceneryPlanes) {
	s := &p.state
	s.FontPosition, s.FontOffset, s.FontOutline, s.GlyphCounter = font.position, font.offset, font.outline, font.counter
	s.FontColor, s.RowCursor, s.RankIndex = font.color, font.row, font.rank
	s.RankDigits, s.DigitBlank, s.DigitValue = font.digits, font.blank, font.value
}

type scoreGlyph struct {
	font          scoreFontState
	glyph, column int
}

// scoreTablePreparation owns an ordered native drawing list. Row capacity
// controls progress without a processor, source memory or host render timing.
type scoreTablePreparation struct {
	glyphs            []scoreGlyph
	final             scoreFontState
	index, row        int
	started, finished bool
}

func (h highScoreTable) prepare() *scoreTablePreparation {
	// Plan on a value copy so assembling the drawing list cannot change live
	// painter state or touch either retained bitmap.
	context := *h.planes
	plan := &scoreTablePreparation{}
	(highScoreTable{&context}).walk(func(glyph, column int) {
		plan.glyphs = append(plan.glyphs, scoreGlyph{readScoreFont(&context), glyph, column})
		context.state.GlyphCounter = 0xffff
	})
	plan.final = readScoreFont(&context)
	return plan
}

func (plan *scoreTablePreparation) advance(p *sceneryPlanes, bank *packedBitmap, rows int) bool {
	if plan.finished {
		return true
	}
	for rows > 0 && plan.index < len(plan.glyphs) {
		glyph := plan.glyphs[plan.index]
		if !plan.started {
			glyph.font.apply(p)
			p.state.GlyphCounter = 4
			plan.started = true
		}
		(highScoreTable{p}).glyphRow(bank, glyph.glyph, glyph.column, plan.row, false)
		p.state.GlyphCounter--
		plan.row++
		rows--
		if plan.row == 5 {
			plan.index++
			plan.row, plan.started = 0, false
		}
	}
	if plan.index == len(plan.glyphs) {
		plan.final.apply(p)
		plan.finished = true
	}
	return plan.finished
}
