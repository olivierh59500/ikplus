package presentation

import (
	"io"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	nativeaudio "github.com/olivierh59500/ikplus/internal/audio"
	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/sound"
)

const audioRate = 44100

type soundPlayer struct {
	context                *audio.Context
	player                 *audio.Player
	mixer                  *sound.Offline
	queue                  *sound.PCMQueue
	capture                io.Writer
	muted, paused, started bool
}

func newSoundPlayer(match *game.Game, muted, device bool) (*soundPlayer, error) {
	data, err := nativeaudio.LoadData()
	if err != nil {
		return nil, err
	}
	if err = match.EnableAudio(); err != nil {
		return nil, err
	}
	mixer, err := sound.NewA500(data, audioRate)
	if err != nil {
		return nil, err
	}
	s := &soundPlayer{mixer: mixer, muted: muted}
	if !device {
		return s, nil
	}
	s.context, s.queue = audio.NewContext(audioRate), sound.NewPCMQueue(audioRate*4)
	s.player, err = s.context.NewPlayer(s.queue)
	if err != nil {
		return nil, err
	}
	s.player.SetBufferSize(20 * time.Millisecond)
	s.sync()
	return s, nil
}

// update renders exactly the completed game journal. The backend reads copied
// PCM and cannot advance the driver or consume gameplay RNG.
func (s *soundPlayer) update(match *game.Game) error {
	pcm, err := s.mixer.Tick(match)
	if err != nil {
		return err
	}
	if s.capture != nil {
		written, err := s.capture.Write(pcm)
		if err == nil && written != len(pcm) {
			err = io.ErrShortWrite
		}
		return err
	}
	if s.queue == nil {
		return nil
	}
	s.queue.Push(pcm)
	if !s.started && s.queue.Buffered() >= audioRate*4/25 {
		s.started = true
		s.sync()
	}
	return nil
}

func (s *soundPlayer) setMuted(muted bool) { s.muted = muted; s.sync() }
func (s *soundPlayer) setPaused(paused bool) {
	if s.paused != paused {
		s.paused = paused
		s.sync()
	}
}
func (s *soundPlayer) sync() {
	if s.player == nil {
		return
	}
	volume := 1.0
	if s.muted {
		volume = 0
	}
	s.player.SetVolume(volume)
	if s.paused || !s.started {
		s.player.Pause()
	} else {
		s.player.Play()
	}
}
func (s *soundPlayer) close() {
	if s.queue != nil {
		_ = s.queue.Close()
	}
	if s.player != nil {
		_ = s.player.Close()
	}
}
