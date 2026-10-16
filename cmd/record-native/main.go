// Command record-native exports the shared native scene and synchronous PCM
// without opening a platform window or sound device.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/olivierh59500/ikplus/internal/presentation"
)

func main() {
	output := flag.String("out", ".cache/video/native-current/gameplay.mkv", "lossless FFV1 Matroska output")
	pcm := flag.String("audio", "", "PCM16 stereo WAV output; defaults to the video path plus .wav")
	replay := flag.String("replay", "", "optional ordinary recorded keyboard/joystick input")
	seed := flag.Uint64("seed", 12, "deterministic tournament seed")
	frames := flag.Int("frames", 15000, "exact number of PAL frames to export")
	lines := flag.Int("pal-lines", 313, "PAL field length; 313 is normal native playback")
	attract := flag.Bool("attract", false, "retain the normal native title/demo cycle")
	scores := flag.Bool("scores", false, "open the native high-score display after 60 frames")
	profile := flag.String("data-dir", "", "isolated export profile; defaults to a profile folder beside the video")
	flag.Parse()
	if *pcm == "" {
		*pcm = *output + ".wav"
	}
	report, err := presentation.RecordSoftware(presentation.Options{
		Seed: *seed, Replay: *replay, SmokeTicks: *frames, PALFrameLines: *lines,
		Attract: *attract, ScoreMenu: *scores, Autoplay: *replay == "" && !*attract && !*scores, DataDir: *profile,
	}, *output, *pcm)
	if err == nil {
		var raw []byte
		raw, err = json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(*output+".json", append(raw, '\n'), 0644)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	seconds := float64(report.Frames) * float64(report.FrameClocks) / float64(report.PALClock)
	fmt.Printf("Recorded %d frames (%.3f seconds), %d audio samples\n%s\nVideo: %s\nAudio: %s\nMetadata: %s.json\n", report.Frames, seconds, report.AudioSamples, report.Final, *output, *pcm, *output)
}
