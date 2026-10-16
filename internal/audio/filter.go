package audio

import "math"

type filterState struct{ rc1, rc2, rc3, rc4, rc5 float32 }

type a500Filter struct {
	fixed1, fixed2, led float32
	states              [2]filterState
}

// rcCoefficient preserves the single-precision boundaries of the independent
// reference's bilinear correction. Cutoffs at Nyquist bypass their stage.
func rcCoefficient(rate, cutoff int) float32 {
	if cutoff >= rate/2 {
		return 1
	}
	omega := float32(2 * math.Pi * float64(cutoff) / float64(rate))
	omega = float32(math.Tan(float64(omega/2)) * 2)
	return 1 / (1 + 1/omega)
}

func newA500Filter(rate int) a500Filter {
	return a500Filter{fixed1: rcCoefficient(rate, 6200), fixed2: rcCoefficient(rate, 20000), led: rcCoefficient(rate, 7000)}
}

func rcStage(coefficient, input, previous float32) float32 {
	// Explicit rounding also prevents a platform-dependent fused multiply-add.
	fresh := float32(coefficient * input)
	held := float32((1 - coefficient) * previous)
	return float32(fresh + held)
}

func (f *a500Filter) sample(input int, channel int, enabled bool) int {
	s := &f.states[channel]
	value := float32(int16(input))
	s.rc1 = float32(float64(rcStage(f.fixed1, value, s.rc1)) + 1e-10)
	s.rc2 = rcStage(f.fixed2, s.rc1, s.rc2)
	s.rc3 = rcStage(f.led, s.rc2, s.rc3)
	s.rc4 = rcStage(f.led, s.rc3, s.rc4)
	s.rc5 = rcStage(f.led, s.rc4, s.rc5)
	output := s.rc2
	if enabled {
		output = s.rc5
	}
	return max(-32768, min(32767, int(output)))
}
