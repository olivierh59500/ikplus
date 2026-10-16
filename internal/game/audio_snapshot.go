package game

import "github.com/olivierh59500/ikplus/internal/audio"

// AudioSnapshot copies musical sequences and effect selection for comparisons.
// Inspecting a snapshot never advances playback or the shared gameplay RNG.
type AudioSnapshot struct {
	Control  audio.ControlState
	Music    audio.MusicState
	Reserved audio.Channel
	Scratch  uint16
	Work     audio.MusicWork
}

func (g *Game) AudioSnapshot() (AudioSnapshot, bool) {
	e := g.audioEngine
	if e == nil {
		return AudioSnapshot{}, false
	}
	s := AudioSnapshot{Control: e.Control.State, Reserved: e.ReservedVoice(), Music: e.Music.Snapshot(), Scratch: e.PitchScratch(), Work: e.Music.Work()}
	return s, true
}
