package audio

// musicSample is a named field requested by the recovered random vocabulary.
// Its position in that vocabulary is unrelated to the track's Go layout.
type musicSample struct {
	Voice int
	Field string
	Shift uint8
}

// RandomByte samples semantic track state without publishing an address space.
func (m *Music) RandomByte(index int) byte {
	request := m.data.Entropy.Requests[index]
	s := m.tracks[request.Voice]
	var value uint16
	var symbol [4]byte
	switch request.Field {
	case "disabled":
		value = s.Disabled
	case "duration":
		value = s.Duration
	case "mode":
		value = s.InstrumentMode
	case "period":
		value = s.Period
	case "slide":
		value = s.Slide
	case "slide-step":
		value = s.SlideStep
	case "slide-target":
		value = s.SlideTarget
	case "slide-delay":
		value = s.SlideDelay
	case "vibrato":
		value = s.Vibrato
	case "envelope":
		symbol = m.data.Entropy.Envelopes[s.Envelope.ID][s.Envelope.Step]
	case "sequence":
		symbol = m.data.Entropy.Patterns[s.Sequence.ID][s.Sequence.Step]
	case "order":
		symbol = m.data.Entropy.Orders[s.Order.ID][s.Order.Step]
	case "repeat":
		symbol = m.data.Entropy.Orders[s.Repeat.ID][s.Repeat.Step]
	case "template":
		symbol = m.data.Entropy.Envelopes[s.EnvelopeTemplate][0]
	case "attack":
		symbol = m.data.sampleEntropy[s.Attack.Offset]
	case "attack-length":
		value = s.Attack.Length
	case "loop":
		symbol = m.data.sampleEntropy[s.Loop.Offset]
	case "loop-length":
		value = s.Loop.Length
	case "vibrato-delay":
		value = s.VibratoDelay
	case "vibrato-interval":
		value = s.VibratoInterval
	case "vibrato-fall":
		value = s.VibratoFall
	case "vibrato-rise":
		value = s.VibratoRise
	case "vibrato-extra":
		value = s.VibratoExtra
	case "progress":
		value = s.Progress
	}
	switch request.Field {
	case "envelope", "sequence", "order", "repeat", "template", "attack", "loop":
		return symbol[3-int(request.Shift)/8]
	}
	return byte(value >> request.Shift)
}
