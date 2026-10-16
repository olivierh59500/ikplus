package presentation

// scoreHint translates $1F1C's authored multiline hints, including the two
// live belt-name substitutions. It writes through the verified five-row font.
func (h highScoreTable) scoreHint(bank *packedBitmap) {
	p := h.planes
	index := int(p.state.HintIndex & 31)
	p.state.FontColor = p.artwork.HintColors[index]
	if index == 24 && int16(p.state.HintTimer) <= 50 {
		// The supplied executable enters a destructive code-writing loop
		// here ($1F46, counter at $1366). Its boot-time reachability needs
		// independent evidence before a native equivalent can be selected.
		panic("original hint code-writing loop is outside the verified presentation path")
	}
	language := 0
	if p.state.German != 0 {
		language = 1
	}
	hint := p.artwork.Score.Hints[language][index]
	if !hint.Valid {
		panic("hint entry is outside the verified native text paths")
	}
	if len(hint.Text) == 0 {
		p.state.HintTimer = 0
		p.state.HintIndex = 0
		return
	}
	p.state.FontPosition = 0x258
	p.state.FontOffset = 160
	p.state.FontLine = p.state.FontPosition
	column := 0
	for _, ch := range hint.Text {
		switch {
		case ch == 0x9b:
			return
		case ch == 0x9c:
			p.state.FontPosition = p.state.FontLine + 40
			p.state.FontLine = p.state.FontPosition
			column = 0
		case ch <= 20:
			column = (column + int(ch)) & 0xffff
		case ch == 0x9f || ch == 0xa0:
			actor := 0
			if ch == 0xa0 {
				actor = 1
			}
			score := uint16(p.state.ScoreDigits[actor*3])<<8 | uint16(p.state.ScoreDigits[1+(actor*3)])
			belt := 0
			for belt < 23 && int16(score) >= int16(p.artwork.Score.BeltThresholds[belt]) {
				belt++
			}
			for letter := 0; letter < 8; letter++ {
				h.glyph(bank, int(p.artwork.Score.BeltLabels[language][belt][letter])-0x30, column+letter, false)
			}
			column = (column + 8) & 0xffff
		default:
			if ch != 32 {
				h.glyph(bank, int(ch)-0x30, column, false)
			}
			column = (column + 1) & 0xffff
		}
	}
}
