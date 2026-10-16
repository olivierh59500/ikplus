package presentation

import "github.com/olivierh59500/ikplus/internal/game"

// hintBacking reconstructs only surviving old font bits. Historical actor
// erasures can remove outline pixels; they must not erase unrelated scenery
// that the IRQ has painted since that main-thread preparation.
func (s *originalScene) hintBacking(picture game.PreparedPicture) {
	if s.hintBase == nil {
		s.hintBase, s.hintMask = make([]byte, 32000), make([]byte, 32000)
	}
	working := s.planes.working.pixels[:32000]
	copy(s.hintBase, working)
	clear(s.hintMask)
	backing := picture.HintBacking
	font := game.PreparedPicture{Hint: backing.State, HintOutline: backing.Outline, HintGerman: backing.German}
	font.Fighters[0].Score, font.Fighters[1].Score = backing.Scores[0], backing.Scores[1]
	previous := s.planes.hintStencil
	s.planes.hintStencil = s.hintMask
	s.hint(font)
	s.planes.hintStencil = previous
	for _, erased := range picture.HintErasures {
		s.erasePreviousPicture(erased)
	}
	for i, mask := range s.hintMask {
		working[i] = s.hintBase[i]&^mask | working[i]&mask
	}
}

// hint paints the state sampled for this bank's prepared picture. Font work
// belongs to the game thread; composing an older bank cannot rewrite live RAM.
func (s *originalScene) hint(picture game.PreparedPicture) {
	p := s.planes
	saved := p.state
	defer func() { p.state = saved }()
	p.state.HintTimer = picture.Hint.Timer
	p.state.HintIndex = picture.Hint.Index
	p.state.German = 0
	if picture.HintGerman {
		p.state.German = 1
	}
	p.state.FontOutline = picture.HintOutline
	for id := 0; id < 2; id++ {
		for digit, divisor := range []int{10000, 100, 1} {
			value := picture.Fighters[id].Score / divisor % 100
			p.state.ScoreDigits[id*3+digit] = byte(value/10*16 + value%10)
		}
	}
	highScoreTable{p}.scoreHint(&p.working)
}
