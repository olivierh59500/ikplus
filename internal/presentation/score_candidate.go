package presentation

type scoreCandidate byte

const (
	scoreDisplayOnly scoreCandidate = iota
	scoreEmpty
	scoreRejected
	scoreQualified
)

// prepareCandidate translates $A110..$A234 up to the next caption/display
// call. It preserves the source's byte-sum zero test and strict last-place
// comparison, then locates the candidate flag after the stable sort.
func (h highScoreTable) prepareCandidate() scoreCandidate {
	p := h.planes
	if int8(p.state.CandidateOwner) < 0 {
		h.sort()
		return scoreDisplayOnly
	}
	p.state.OwnerFlash[0], p.state.OwnerFlash[1] = 30, 30
	clear(p.state.ScoreCandidate[0:50])
	p.state.ScoreCandidate[50] = 1
	offset := 0
	if p.state.Order[2] != 0 {
		offset = 3
	}
	p.state.ScoreHigh[50], p.state.ScoreLow[50] = p.state.ScoreDigits[offset], p.state.ScoreDigits[1+(offset)]
	if p.state.ScoreHigh[50]+p.state.ScoreLow[50] == 0 {
		return scoreEmpty
	}
	p.state.NameFirst[50], p.state.NameSecond[50], p.state.NameThird[50] = 12, 12, 12
	high, low := p.state.ScoreHigh[49], p.state.ScoreLow[49]
	if high > p.state.ScoreHigh[50] || high == p.state.ScoreHigh[50] && low >= p.state.ScoreLow[50] {
		return scoreRejected
	}
	h.sort()
	index := 0
	for index < 50 && p.state.ScoreCandidate[index] == 0 {
		index++
	}
	if index == 50 {
		return scoreRejected
	}
	p.state.CandidateIndex = byte(index)
	row, first := index, 0
	if index >= 6 {
		row, first = 0, index
	}
	if index >= 45 {
		row, first = index-45, 45
	}
	p.state.CandidateRow, p.state.FirstScore = byte(row), byte(first)
	return scoreQualified
}
