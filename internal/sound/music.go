// Package sound connects native sample playback to Ebitengine and export.
package sound

import "github.com/olivierh59500/ikplus/internal/audio"

// Music reads the original game-specific note and instrument sequences.
type Music = audio.Music

func NewMusic(data *audio.Data, sampleRate int) (*Music, error) {
	return audio.NewMusic(data, sampleRate)
}
