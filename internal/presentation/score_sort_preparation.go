package presentation

// scoreSortPreparation retains the stable table pass while scene updates
// continue. Comparisons and swaps consume native preparation capacity.
type scoreSortPreparation struct {
	flow                             *highScoreFlow
	remaining, index, debt, position int
}

func (f *highScoreFlow) beginScoreSort() {
	f.part = scoreSorting
	f.sorting = &scoreSortPreparation{flow: f, remaining: 50, position: 259 * scorePreparationQuota / 313}
	f.clock.planes.state.SortRemaining, f.clock.planes.state.RowCursor = 50, 0
	f.sorting.advance()
}

func (w *scoreSortPreparation) advance() {
	p := w.flow.clock.planes
	for w.remaining > 0 && w.position < scorePreparationQuota {
		left, right := w.index, w.index+1
		swap := p.state.ScoreHigh[left] < p.state.ScoreHigh[right] ||
			p.state.ScoreHigh[left] == p.state.ScoreHigh[right] && p.state.ScoreLow[left] < p.state.ScoreLow[right]
		if w.debt == 0 {
			w.debt = 110
			if p.state.ScoreHigh[left] != p.state.ScoreHigh[right] {
				w.debt = 86
			}
			if swap {
				w.debt += 500
			}
		}
		used := min(w.debt, scorePreparationQuota-w.position)
		w.position += used
		w.debt -= used
		if w.debt != 0 {
			return
		}
		if swap {
			(highScoreTable{p}).swapRows(left, right)
		}
		w.index++
		p.state.RowCursor = byte(w.index)
		if w.index == w.remaining {
			w.remaining--
			p.state.SortRemaining = byte(w.remaining)
			if w.remaining != 0 {
				w.index, p.state.RowCursor = 0, 0
			}
		}
	}
	if w.remaining == 0 {
		f := w.flow
		f.hideCaption()
		f.clock = newPacedHighScoreSequence(p, w.position)
		f.part, f.waiting, f.sorting = scoreDemo, true, nil
	}
}
