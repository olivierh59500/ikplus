package sound

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/olivierh59500/ikplus/internal/audio"
	"github.com/olivierh59500/ikplus/internal/game"
)

// Offline mixes completed game commands on four sample voices.
// It neither dispatches effects nor consumes the shared gameplay RNG.
type Offline struct {
	renderer   *audio.Renderer
	SampleRate int
}

func NewOffline(data *audio.Data, rate int) (*Offline, error) {
	if data == nil {
		return nil, fmt.Errorf("native audio data is missing")
	}
	renderer, err := audio.NewRenderer(data.Samples, rate)
	if err != nil {
		return nil, err
	}
	return &Offline{renderer: renderer, SampleRate: rate}, nil
}

// NewA500 applies the original fixed and switchable low-pass response.
// NewOffline produces the unfiltered four-voice mix.
func NewA500(data *audio.Data, rate int) (*Offline, error) {
	mixer, err := NewOffline(data, rate)
	if err != nil {
		return nil, err
	}
	mixer.renderer.SetFiltered(true)
	return mixer, nil
}

// Tick follows an audio-enabled Game.Tick. PCM length follows the rational
// update duration, so consecutive frames can contain different sample counts.
func (m *Offline) Tick(g *game.Game) ([]byte, error) {
	frame, ok := g.AudioFrame()
	if !ok || frame.End <= frame.Begin {
		return nil, fmt.Errorf("advance an audio-enabled game before rendering PCM")
	}
	return m.renderer.Render(frame)
}

// WritePCMHeader writes a PCM16 stereo header for the actual rendered frames.
// Exporters can rewrite a placeholder header after the final game journal.
func WritePCMHeader(w io.Writer, rate int, frames uint64) error {
	if rate < 1 || uint64(rate) > 0xffffffff/4 || frames > (0xffffffff-36)/4 {
		return fmt.Errorf("invalid PCM rate or RIFF size")
	}
	size := frames * 4
	var header [44]byte
	copy(header[:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(size)+36)
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 2)
	binary.LittleEndian.PutUint32(header[24:], uint32(rate))
	binary.LittleEndian.PutUint32(header[28:], uint32(rate)*4)
	binary.LittleEndian.PutUint16(header[32:], 4)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(size))
	n, err := w.Write(header[:])
	if err == nil && n != len(header) {
		return io.ErrShortWrite
	}
	return err
}

// WriteWAVHeader retains nominal 50 Hz duration for legacy diagnostic callers.
// Game exports use WritePCMHeader with the actual rendered frame count.
func WriteWAVHeader(w io.Writer, rate, ticks int) error {
	if ticks < 0 || rate < 1 || rate%game.TickRate != 0 {
		return fmt.Errorf("invalid WAV duration or sample rate")
	}
	frames := uint64(ticks) * uint64(rate/game.TickRate)
	return WritePCMHeader(w, rate, frames)
}
