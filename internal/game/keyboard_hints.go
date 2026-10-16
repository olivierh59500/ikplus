package game

// HintState retains the keyboard-selected $952/$954 controller and its
// separate $628 repeat counter. The main picture owns the sampled timer.
type HintState struct {
	Timer, Index, Count uint16
}

func (g *Game) clearHintArena() {
	g.hint.Timer, g.hint.Index = 0, 0
	g.hintPosition, g.hintOffset, g.hintLine, g.hintGlyphCounter, g.hintColor = 0, 0, 0, 0, 0
}

func (g *Game) readHintState(state *PresentationState) {
	holder := state
	g.hint = HintState{holder.HintTimer, holder.HintIndex, holder.HintCount}
	g.hintOutline = holder.FontOutline
	g.hintPosition, g.hintOffset, g.hintLine = holder.FontPosition, holder.FontOffset, holder.FontLine
	g.hintGlyphCounter, g.hintColor = holder.GlyphCounter, holder.FontColor
}

// prepareHintText advances the source font's working words on the game thread.
// The renderer reads the prepared picture and cannot mutate simulation state.
func (g *Game) prepareHintText(direct bool) {
	if !direct {
		g.hintOutline = 0
		if int16(g.hint.Timer) <= 20 {
			return
		}
	}
	index := int(g.hint.Index & 31)
	g.hintColor = fighterArtwork.HintColors[index]
	if !direct && g.hintColor != 15 {
		g.hintOutline = 0xffff
	}
	language := 0
	if g.germanText {
		language = 1
	}
	hint := fighterArtwork.Score.Hints[language][index]
	if !hint.Valid {
		panic("hint entry is outside the verified native text paths")
	}
	if len(hint.Text) == 0 {
		g.hint.Timer, g.hint.Index = 0, 0
		return
	}
	g.hintPosition, g.hintOffset, g.hintLine = 0x258, 160, 0x258
	for _, ch := range hint.Text {
		switch {
		case ch == 0x9b:
			return
		case ch == 0x9c:
			g.hintPosition = g.hintLine + 40
			g.hintLine = g.hintPosition
		case ch <= 20 || ch == 32:
			// Spaces/control advances do not call $A5E8.
		default:
			g.hintGlyphCounter = 0xffff
		}
	}
}

func (g *Game) Hint() HintState {
	if holder := g.presentationState; holder != nil {
		return HintState{holder.HintTimer, holder.HintIndex, holder.HintCount}
	}
	return g.hint
}

// IsHintSequence recognizes only the original twenty authored signatures.
// Frontend shortcuts can use this to avoid interrupting a typed game word.
func IsHintSequence(history uint32) bool {
	for index := 0; index < 20; index++ {
		if nativeGameplay.Input.HintSignatures[index] == history {
			return true
		}
	}
	return false
}

func (g *Game) applyAmbientKey(history uint32) bool {
	if history&0xffffff != 0x999e2e && history != 0xa1979f23 && history != 0xb0979320 && history != 0x99929397 {
		return false
	}
	g.ambient.KeySequence(history)
	if holder := g.presentationState; holder != nil {
		index := 0
		switch history {
		case 0xa1979f23:
			index = 1
		case 0xb0979320:
			index = 2
		case 0x99929397:
			index = 3
		}
		holder.AmbientForces[index] = 0xffff
	}
	return true
}

func (g *Game) applyHintKey(key, mode byte) bool {
	h := g.Hint()
	if h.Timer != 0 || mode != 1 && mode != 3 {
		return false
	}
	index := -1
	if key == 0x62 {
		index = 0
		for id := 0; id < 2; id++ {
			if g.Fighters[id].Human {
				index++
			}
		}
		if g.Fighters[1].Human {
			index++
		}
		if holder := g.presentationState; holder != nil {
			index = int(holder.HumanCount) + int(holder.Humans[1])
		}
	} else {
		for number := 0; number < 20; number++ {
			if nativeGameplay.Input.HintSignatures[number] == g.speedKeyHistory {
				index = number + 4
				break
			}
		}
	}
	if index < 0 {
		return false
	}
	h.Index = uint16(index)
	h.Count += uint16(nativeGameplay.Input.HintCounts[index])
	g.hint = h
	if holder := g.presentationState; holder != nil {
		holder.HintIndex = h.Index
		holder.HintCount = h.Count
	}
	if int16(h.Count) > 1 {
		g.restartRequested = true
		return true
	}
	h.Timer = 250
	g.hint = h
	if holder := g.presentationState; holder != nil {
		holder.HintTimer = h.Timer
	}
	g.speedValue = 3
	g.displaySpeedValue = g.speedValue - g.OriginalFast()
	if g.presentationSpeedChanged != nil {
		g.presentationSpeedChanged()
	}
	return true
}
