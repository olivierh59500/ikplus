// Command render-audio exports synchronous original audio for a deterministic
// skilled-player video capture without opening a window or an audio device.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/olivierh59500/ikplus/internal/audio"
	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/pilot"
	"github.com/olivierh59500/ikplus/internal/sound"
)

func main() {
	seed := flag.Uint64("seed", 1988, "deterministic tournament seed")
	ticks := flag.Int("ticks", 30000, "number of PAL frames to render")
	out := flag.String("out", ".cache/video/gameplay.wav", "output PCM16 stereo WAV file")
	musicOut := flag.String("music-out", "", "optional separate 20-second original music WAV without effects")
	lines := flag.Int("pal-lines", 313, "PAL field length; use the same value as the video capture")
	flag.Parse()
	profile := audio.Supplied313
	if *lines == 312 {
		profile = audio.Controlled312
	} else if *lines != 313 {
		fmt.Fprintln(os.Stderr, "PAL field must contain 312 or 313 lines")
		os.Exit(2)
	}
	if err := render(*seed, *ticks, *out, profile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *musicOut != "" {
		if err := renderMusic(*musicOut, profile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// renderMusic supplies source music for title and credit cards. Gameplay uses
// its own uninterrupted music timeline and four sample voices.
func renderMusic(output string, profile audio.PALFrameProfile) error {
	data, err := audio.LoadData()
	if err != nil {
		return err
	}
	mixer, err := sound.NewA500(data, 44100)
	if err != nil {
		return err
	}
	g := game.New(1988)
	if err = g.SetPALFrameProfile(profile); err != nil {
		return err
	}
	if err = g.EnableAudio(); err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()
	w := bufio.NewWriter(file)
	const frames = 20 * 44100
	if err = sound.WritePCMHeader(w, 44100, frames); err != nil {
		return err
	}
	remaining := frames * 4
	for remaining > 0 {
		g.Tick([2]game.Input{})
		pcm, err := mixer.Tick(g)
		if err != nil {
			return err
		}
		if len(pcm) > remaining {
			pcm = pcm[:remaining]
		}
		if _, err = w.Write(pcm); err != nil {
			return err
		}
		remaining -= len(pcm)
	}
	if err = w.Flush(); err != nil {
		return err
	}
	return file.Close()
}

func render(seed uint64, ticks int, output string, profile audio.PALFrameProfile) error {
	if ticks < 1 {
		return fmt.Errorf("ticks must be positive")
	}
	if seed == 0 {
		seed = 1988 // Desktop direct-game captures use this seed for zero.
	}
	data, err := audio.LoadData()
	if err != nil {
		return err
	}
	mixer, err := sound.NewA500(data, 44100)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()
	w := bufio.NewWriterSize(file, 1<<20)
	if err = sound.WritePCMHeader(w, 44100, 0); err != nil {
		return err
	}
	g := game.New(seed)
	// Match the desktop autoplay startup order, including its initial effect
	// stop journal and physical CIA pins, before starting the fresh tournament.
	g.UsePhysicalJoystickPins()
	g.SetBootMarker(0x01010202)
	if err = g.SetPALFrameProfile(profile); err != nil {
		return err
	}
	if err = g.EnableAudio(); err != nil {
		return err
	}
	g.Start(1)
	p := pilot.New(0)
	frames := uint64(0)
	for tick := 0; tick < ticks; tick++ {
		g.Tick([2]game.Input{p.Input(g)})
		pcm, err := mixer.Tick(g)
		if err != nil {
			return fmt.Errorf("tick %d: %w", tick, err)
		}
		if _, err = w.Write(pcm); err != nil {
			return err
		}
		frames += uint64(len(pcm) / 4)
	}
	if err = w.Flush(); err != nil {
		return err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err = sound.WritePCMHeader(file, 44100, frames); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	fmt.Printf("Rendered %.2f seconds of original audio: %s (round %d; %s; score %d)\n", float64(frames)/44100, output, g.Round, g.Phase, g.Fighters[0].Score)
	return nil
}
