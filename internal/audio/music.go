package audio

import (
	"encoding/binary"
	"fmt"
	"io"
)

// TrackState is one musical voice's independent native controller.
// Durations, envelopes and cursors are ordinary Go values, never RAM fields.
type TrackState struct {
	Disabled                                                uint16
	Duration, InstrumentMode, Period                        uint16
	Slide, SlideStep, SlideTarget, SlideDelay               uint16
	Vibrato                                                 uint16
	Envelope, Sequence, Order, Repeat                       Cursor
	EnvelopeTemplate                                        int
	Attack, Loop                                            Sample
	VibratoDelay, VibratoInterval, VibratoFall, VibratoRise uint16
	VibratoExtra, Progress                                  uint16
}

// MusicState is an immutable observation of the four sequence controllers.
type MusicState struct{ Tracks [4]TrackState }

// MusicWork counts logical operations in one native sequence update. It is
// independent of device reads, host execution time and sample output size.
type MusicWork struct {
	Voices, ReservedVoices, Commands, Orders, Instruments     int
	Notes, Rests, Stops, Settings, Releases                   int
	Envelopes, EnvelopeHolds, Loops, PitchSteps, PitchEffects int
}

// Music advances decoded notes, instruments, envelopes and pitch effects.
// Reader output is PCM16 stereo; game updates and device reads stay separate.
type Music struct {
	data                       *Data
	tracks                     [4]TrackState
	channels                   [4]voice
	sampleRate, tickPhase      int
	tail                       [4]byte
	tailPosition               int
	err                        error
	filter, effectBusy         bool
	dummy                      voice
	onVoice                    func(int, Channel, bool)
	onFilter                   func(bool)
	fireButtons                [2]bool
	masterVolume, pitchChanges uint16
	work                       MusicWork
}

type voice struct {
	offset                 uint32
	length, period, volume uint16
	enabled                bool
	activeOffset           uint32
	activeLength           int
	position               float64
}

// Channel selects a PCM span, its playback pitch and its volume.
type Channel struct {
	Offset                 uint32
	Length, Period, Volume uint16
	Enabled                bool
}

func (m *Music) Registers() [4]Channel {
	var channels [4]Channel
	for i, c := range m.channels {
		channels[i] = Channel{c.offset, c.length, c.period, c.volume, c.enabled}
	}
	return channels
}
func (m *Music) Snapshot() MusicState { return MusicState{m.tracks} }
func (m *Music) Work() MusicWork      { return m.work }

// Restore restores sequence controls for deterministic replay. It does not
// change sample phases, audio time or playback voices.
func (m *Music) Restore(state MusicState) { m.tracks = state.Tracks }
func (m *Music) Err() error               { return m.err }
func (m *Music) Tick()                    { m.tick() }

var _ io.Reader = (*Music)(nil)

func NewMusic(data *Data, sampleRate int) (*Music, error) {
	if sampleRate < 8000 || sampleRate > 192000 {
		return nil, fmt.Errorf("music sample rate must be between 8000 and 192000 Hz")
	}
	if data == nil || data.Version != 1 || len(data.Samples) == 0 {
		return nil, fmt.Errorf("native music data is missing")
	}
	return &Music{data: data, tracks: data.Tracks, sampleRate: sampleRate, tailPosition: 4, tickPhase: sampleRate, masterVolume: 15}, nil
}

func (m *Music) Read(out []byte) (int, error) {
	n := 0
	for n < len(out) {
		if m.err != nil {
			return n, m.err
		}
		if m.tailPosition == 4 {
			m.renderFrame(m.tail[:])
			m.tailPosition = 0
		}
		copied := copy(out[n:], m.tail[m.tailPosition:])
		m.tailPosition += copied
		n += copied
	}
	return n, nil
}

func (m *Music) renderFrame(out []byte) {
	if m.tickPhase >= m.sampleRate {
		m.tickPhase -= m.sampleRate
		m.tick()
	}
	m.tickPhase += 50
	var left, right int
	for i := range m.channels {
		c := &m.channels[i]
		if !c.enabled || c.period == 0 || c.activeLength == 0 {
			continue
		}
		for c.position >= float64(c.activeLength) {
			c.position -= float64(c.activeLength)
			c.activeOffset = c.offset
			c.activeLength = sampleLength(c.length)
		}
		at := int(c.activeOffset) + int(c.position)
		if at < 0 || at >= len(m.data.Samples) {
			m.fail("PCM offset %d is outside the sample bank", at)
			continue
		}
		amplitude := int(int8(m.data.Samples[at])) * int(min(c.volume, 64)) * 2
		if i == 0 || i == 3 {
			left += amplitude
		} else {
			right += amplitude
		}
		c.position += float64(PALClock) / float64(c.period) / float64(m.sampleRate)
	}
	binary.LittleEndian.PutUint16(out, uint16(int16(max(-32768, min(32767, left)))))
	binary.LittleEndian.PutUint16(out[2:], uint16(int16(max(-32768, min(32767, right)))))
}

func sampleLength(words uint16) int {
	if words == 0 {
		return 131072
	}
	return int(words) * 2
}

func (m *Music) tick() {
	m.work = MusicWork{}
	for index := range m.channels {
		channel := &m.channels[index]
		if index == 3 && m.effectBusy {
			channel = &m.dummy
			m.work.ReservedVoices++
		}
		m.updateVoice(index, channel)
		if m.err != nil {
			return
		}
	}
}

func (m *Music) updateVoice(index int, c *voice) {
	s := &m.tracks[index]
	if s.Disabled != 0 {
		return
	}
	m.work.Voices++
	switch s.Duration {
	case 0:
		m.sequence(index, c, s.Sequence)
		return
	case 1:
		if s.InstrumentMode != 0 {
			m.work.Releases++
			m.enableVoice(c, false)
		}
		s.Duration--
	default:
		if s.InstrumentMode == 3 {
			m.work.Loops++
			c.offset, c.length = s.Loop.Offset, s.Loop.Length
			m.publishVoice(c, false)
			s.InstrumentMode = 1
		}
		s.Duration--
	}
	m.pitchChanges = 0
	period, changed := s.Period, false
	if s.Slide != 0 {
		m.work.PitchEffects++
		if s.SlideDelay != 0 {
			s.SlideDelay--
		} else {
			changed = true
			m.pitchChanges++
			if s.Slide == 1 {
				period += s.SlideStep
				if period > s.SlideTarget {
					period = s.SlideTarget
					s.Slide = 0
				}
			} else {
				period -= s.SlideStep
				if period <= s.SlideTarget {
					period = s.SlideTarget
					s.Slide = 0
				}
			}
		}
	}
	if s.Vibrato != 0 {
		m.work.PitchEffects++
		if s.VibratoDelay != 0 {
			s.VibratoDelay--
		} else {
			s.VibratoDelay = s.VibratoInterval
			changed = true
			m.pitchChanges++
			if s.Vibrato < 3 {
				period -= s.VibratoRise
				s.Vibrato++
			} else {
				period += s.VibratoFall
				s.Vibrato++
				if s.Vibrato == 5 {
					s.Vibrato = 1
				}
			}
		}
	}
	if changed {
		m.work.PitchSteps += int(m.pitchChanges)
		s.Period = period
		c.period = period
		m.publishVoice(c, false)
	}
	m.envelope(c, s)
}

func (m *Music) envelope(c *voice, s *TrackState) {
	m.work.Envelopes++
	cursor := &s.Envelope
	if cursor.ID < 0 || cursor.ID >= len(m.data.Envelopes) {
		m.fail("invalid envelope %d", cursor.ID)
		return
	}
	values := m.data.Envelopes[cursor.ID]
	if len(values) == 0 {
		m.fail("missing volume envelope")
		return
	}
	value := values[min(cursor.Step, len(values)-1)]
	if cursor.Step >= len(values) {
		m.work.EnvelopeHolds++
	}
	if cursor.Step < len(values) {
		cursor.Step++
	}
	// The authored volume calculation retains its 16-bit product.
	c.volume = ((value * m.masterVolume) >> 4) & 127
	m.publishVoice(c, false)
}

func (m *Music) sequence(index int, c *voice, cursor Cursor) {
	s := &m.tracks[index]
	for count := 0; count < 1024 && m.err == nil; count++ {
		if cursor.ID <= 0 || cursor.ID >= len(m.data.Patterns) || cursor.Step >= len(m.data.Patterns[cursor.ID]) {
			m.fail("invalid music pattern cursor %+v", cursor)
			return
		}
		command := m.data.Patterns[cursor.ID][cursor.Step]
		m.work.Commands++
		cursor.Step++
		switch command.Kind {
		case "stop":
			m.work.Stops++
			return
		case "note":
			m.work.Notes++
			s.Period = command.Period
			c.period = command.Period
			m.publishVoice(c, false)
			m.startNote(index, c, cursor, command.Duration)
			return
		case "instrument":
			m.work.Instruments++
			instrument := m.data.Instruments[command.Instrument]
			s.InstrumentMode = instrument.Mode
			if instrument.Mode == 0 {
				c.offset, c.length = instrument.Attack.Offset, instrument.Attack.Length
				m.publishVoice(c, false)
			} else {
				s.Attack, s.Loop = instrument.Attack, instrument.Loop
			}
		case "pattern":
			m.work.Orders++
			order := s.Order
			if order.Step >= len(m.data.Orders[order.ID]) {
				order = s.Repeat
				s.Order = order
			}
			if order.Step >= len(m.data.Orders[order.ID]) {
				m.fail("empty song order")
				return
			}
			s.Order.Step++
			cursor = Cursor{ID: m.data.Orders[order.ID][order.Step]}
		case "envelope":
			m.work.Settings++
			s.EnvelopeTemplate = command.Envelope
		case "slide":
			m.work.Settings++
			s.Slide, s.SlideStep, s.SlideTarget, s.SlideDelay = command.Slide, command.Step, command.Target, command.Delay
		case "vibrato":
			m.work.Settings++
			s.Vibrato = 1
			s.VibratoDelay, s.VibratoInterval = command.Delay, command.Interval
			s.VibratoFall, s.VibratoRise, s.VibratoExtra, s.Progress = command.Fall, command.Rise, command.Extra, command.Progress
		case "slide-off":
			m.work.Settings++
			s.Slide = 0
		case "vibrato-off":
			m.work.Settings++
			s.Vibrato = 0
		case "rest":
			m.work.Rests++
			s.Duration = command.Duration - 1
			s.Sequence = cursor
			s.Envelope = Cursor{ID: m.data.SilentEnvelope}
			s.Progress = m.nextTrackDisabled(index)
			c.volume = 0
			m.publishVoice(c, false)
			return
		case "filter":
			m.work.Settings++
			m.SetFilter(command.Enabled)
		default:
			m.fail("unsupported decoded music command %q", command.Kind)
		}
	}
	if m.err == nil {
		m.fail("music command loop exceeded its limit")
	}
}

func (m *Music) nextTrackDisabled(index int) uint16 {
	if index+1 < len(m.tracks) {
		return m.tracks[index+1].Disabled
	}
	return m.data.AfterTracks
}

func (m *Music) startNote(index int, c *voice, cursor Cursor, duration uint16) {
	s := &m.tracks[index]
	s.Duration = duration - 1
	s.Sequence = cursor
	s.Envelope = Cursor{ID: s.EnvelopeTemplate}
	s.Progress = m.nextTrackDisabled(index)
	if s.Vibrato != 0 {
		s.Vibrato = 1
	}
	if s.InstrumentMode != 0 {
		s.InstrumentMode = 3
		c.offset, c.length = s.Attack.Offset, s.Attack.Length
		m.publishVoice(c, false)
	}
	m.enableVoice(c, true)
	m.envelope(c, s)
}

func (m *Music) SetFilter(enabled bool) {
	m.filter = enabled
	if m.onFilter != nil {
		m.onFilter(enabled)
	}
}
func (m *Music) publishVoice(c *voice, restart bool) {
	for i := range m.channels {
		if c == &m.channels[i] && m.onVoice != nil {
			m.onVoice(i, Channel{c.offset, c.length, c.period, c.volume, c.enabled}, restart)
			return
		}
	}
}
func (m *Music) enableVoice(c *voice, enabled bool) {
	restart := enabled && !c.enabled
	if restart {
		c.activeOffset, c.activeLength, c.position = c.offset, sampleLength(c.length), 0
	}
	c.enabled = enabled
	m.publishVoice(c, restart)
}
func (m *Music) fail(format string, values ...any) {
	if m.err == nil {
		m.err = fmt.Errorf(format, values...)
	}
}
