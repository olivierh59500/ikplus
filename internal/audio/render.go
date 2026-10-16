package audio

import (
	"encoding/binary"
	"fmt"
)

type sampleVoice struct {
	channel  Channel
	offset   uint32
	length   int
	position float64
	oneShot  bool
}

// Renderer mixes four sample voices. It knows sample positions, pitch, volume
// and looping; no hardware registers, DMA slots or interrupts are involved.
type Renderer struct {
	samples            []byte
	rate               uint64
	clock, sampleIndex uint64
	voices             [4]sampleVoice
	buffer             []byte
	filtered, led      bool
	filter             a500Filter
}

func NewRenderer(samples []byte, rate int) (*Renderer, error) {
	if rate < 8000 || rate > 192000 {
		return nil, fmt.Errorf("audio sample rate must be between 8000 and 192000 Hz")
	}
	return &Renderer{samples: samples, rate: uint64(rate), filter: newA500Filter(rate)}, nil
}

func (r *Renderer) SetFiltered(enabled bool) { r.filtered = enabled }

func (r *Renderer) advance(at uint64) {
	elapsed := at - r.clock
	for index := range r.voices {
		v := &r.voices[index]
		if !v.channel.Enabled || v.channel.Period == 0 {
			continue
		}
		v.position += float64(elapsed) / float64(v.channel.Period)
		for v.position >= float64(v.length) && v.length != 0 {
			if v.oneShot {
				v.channel.Enabled = false
				break
			}
			v.position -= float64(v.length)
			v.offset, v.length = v.channel.Offset, sampleLength(v.channel.Length)
		}
	}
	r.clock = at
}

func (r *Renderer) apply(event Event) {
	if event.Filter {
		r.led = event.FilterEnabled
		return
	}
	v := &r.voices[event.Voice]
	if event.Restart || event.Channel.Enabled && !v.channel.Enabled {
		v.offset, v.length, v.position = event.Channel.Offset, sampleLength(event.Channel.Length), 0
	}
	v.channel, v.oneShot = event.Channel, event.OneShot
}

// Render returns a reused PCM16 stereo buffer. Consecutive frames share sample
// phase and exact rational duration, including rates not divisible by 50.
func (r *Renderer) Render(frame Frame) ([]byte, error) {
	if frame.Begin != r.clock || frame.End < frame.Begin {
		return nil, fmt.Errorf("nonconsecutive audio frame %d..%d after %d", frame.Begin, frame.End, r.clock)
	}
	previous := frame.Begin
	for _, event := range frame.Events {
		if event.At < previous || event.At > frame.End || !event.Filter && (event.Voice < 0 || event.Voice >= 4) {
			return nil, fmt.Errorf("invalid audio event at %d", event.At)
		}
		previous = event.At
	}
	r.buffer = r.buffer[:0]
	next := 0
	for r.sampleIndex*PALClock < frame.End*r.rate {
		at := r.sampleIndex * PALClock / r.rate
		for next < len(frame.Events) && frame.Events[next].At <= at {
			r.advance(frame.Events[next].At)
			r.apply(frame.Events[next])
			next++
		}
		r.advance(at)
		var stereo [2]int
		for index, v := range r.voices {
			if !v.channel.Enabled || v.channel.Period == 0 || v.length == 0 {
				continue
			}
			offset := int(v.offset) + int(v.position)
			if offset < 0 || offset >= len(r.samples) {
				return nil, fmt.Errorf("PCM offset %d is outside the sample bank", offset)
			}
			amplitude := int(int8(r.samples[offset])) * int(min(v.channel.Volume, 64)) * 2
			if index == 0 || index == 3 {
				stereo[0] += amplitude
			} else {
				stereo[1] += amplitude
			}
		}
		for index := range stereo {
			if r.filtered {
				stereo[index] = r.filter.sample(stereo[index], index, r.led)
			}
			value := int16(max(-32768, min(32767, stereo[index])))
			r.buffer = binary.LittleEndian.AppendUint16(r.buffer, uint16(value))
		}
		r.sampleIndex++
	}
	for ; next < len(frame.Events); next++ {
		r.advance(frame.Events[next].At)
		r.apply(frame.Events[next])
	}
	r.advance(frame.End)
	return r.buffer, nil
}
