package audio

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/olivierh59500/ikplus/assets"
)

// Cursor identifies a native pattern, order or envelope and its next element.
type Cursor struct{ ID, Step int }

// Sample describes a span of PCM data in the extracted sample bank.
type Sample struct {
	Offset uint32
	Length uint16
}

// Instrument selects a direct sample or an attack followed by a loop.
type Instrument struct {
	Mode         uint16
	Attack, Loop Sample
}

// Command contains decoded musical content, without source code locations.
type Command struct {
	Kind                                                              string
	Period, Duration                                                  uint16
	Instrument, Envelope                                              int
	Slide, Step, Target, Delay, Interval, Fall, Rise, Extra, Progress uint16
	Enabled                                                           bool
}

type poseTriggers struct {
	Frames, Ordinary, OrdinaryDelays, OrdinaryDelayed []byte
	Hits, HitDelays, HitDelayed, Down                 []byte
}

type sampleEntropy struct {
	Offset uint32
	Value  [4]byte
}
type musicEntropy struct {
	Requests                    []musicSample
	Patterns, Orders, Envelopes [][][4]byte
	Samples                     []sampleEntropy
}

// Data owns immutable authored music commands and sample data.
// Its entropy vocabulary preserves deterministic gameplay randomness while
// track cursors and sample locations use native indices throughout playback.
type Data struct {
	Version        int
	SourceSHA256   string
	Tracks         [4]TrackState
	Patterns       [][]Command
	Orders         [][]int
	Envelopes      [][]uint16
	Instruments    []Instrument
	Effects        []Sample
	Triggers       poseTriggers
	SilentOffset   uint32
	SilentEnvelope int
	AfterTracks    uint16
	Entropy        musicEntropy
	Samples        []byte `json:"-"`
	sampleEntropy  map[uint32][4]byte
}

var nativeData = sync.OnceValues(func() (*Data, error) {
	raw, err := assets.Read("original/audio.json")
	if err != nil {
		return nil, err
	}
	var data Data
	if err = json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	data.Samples, err = assets.Read("original/audio.pcm")
	if err != nil {
		return nil, err
	}
	if data.Version != 1 || len(data.Patterns) < 2 || len(data.Orders) < 2 || len(data.Instruments) < 2 || len(data.Effects) != 18 || len(data.Triggers.Frames) != 17 {
		return nil, fmt.Errorf("invalid extracted audio data")
	}
	data.sampleEntropy = make(map[uint32][4]byte, len(data.Entropy.Samples))
	for _, sample := range data.Entropy.Samples {
		data.sampleEntropy[sample.Offset] = sample.Value
	}
	return &data, nil
})

// LoadData returns the shared, immutable native music and PCM asset bank.
func LoadData() (*Data, error) { return nativeData() }
