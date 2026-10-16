package presentation

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/olivierh59500/ikplus/internal/audio"
	"github.com/olivierh59500/ikplus/internal/sound"
)

// SoftwareRecording describes native scene pixels before GPU upload and
// application overlays. It never represents a capture of a platform window.
type SoftwareRecording struct {
	Renderer                 string
	Width, Height, Frames    int
	PALClock, FrameClocks    uint64
	AudioRate                int
	AudioSamples             uint64
	VideoSHA256, AudioSHA256 string
	ReplaySHA256             string `json:",omitempty"`
	SourceRevision           string `json:",omitempty"`
	SourceModified           bool
	FrameSHA256              []string
	Milestones               []RecordingMilestone
	Final                    string
}

// RecordingMilestone identifies unedited source frames for later composition.
type RecordingMilestone struct {
	Scene, WhiteBelt                   string
	Mode                               byte
	Points                             [3]int
	VideoFrame                         int
	GameFrame                          uint64
	Phase                              string
	Round, WhiteScore                  int
	BonusNumber, BonusTotal, Deflected int
	BonusSuccess                       bool
}

type softwarePCMWriter struct {
	output io.Writer
	bytes  uint64
}

func (w *softwarePCMWriter) Write(p []byte) (int, error) {
	n, err := w.output.Write(p)
	w.bytes += uint64(n)
	return n, err
}

// RecordSoftware exports the shared 320x200 compositor and synchronous native
// PCM without opening a window or sound device. SmokeTicks selects the exact
// frame count; normal replay/autoplay startup and Update remain authoritative.
func RecordSoftware(options Options, videoPath, audioPath string) (SoftwareRecording, error) {
	var report SoftwareRecording
	if options.SmokeTicks < 1 || videoPath == "" {
		return report, fmt.Errorf("software recording requires a video path and positive frame count")
	}
	if options.Record != "" || options.Capture != "" {
		return report, fmt.Errorf("software recording cannot also request a GPU recording or screenshot")
	}
	if audioPath != "" {
		video, err := filepath.Abs(videoPath)
		if err != nil {
			return report, err
		}
		audio, err := filepath.Abs(audioPath)
		if err != nil {
			return report, err
		}
		if video == audio {
			return report, fmt.Errorf("video and audio outputs must be different files")
		}
	}
	if options.DataDir == "" {
		options.DataDir = filepath.Join(filepath.Dir(videoPath), "profile")
	}
	options.Muted = true
	a, err := newApp(options, false)
	if err != nil {
		return report, err
	}
	defer a.Close()
	a.recorder, err = newFrameRecorder(videoPath, 320, 200, a.match.PALFrameClocks())
	if err != nil {
		return report, err
	}
	var pcmFile *os.File
	var pcmBuffer *bufio.Writer
	var pcm softwarePCMWriter
	if audioPath != "" {
		if err := os.MkdirAll(filepath.Dir(audioPath), 0755); err != nil {
			return report, err
		}
		pcmFile, err = os.Create(audioPath)
		if err != nil {
			return report, err
		}
		defer pcmFile.Close()
		if err := sound.WritePCMHeader(pcmFile, audioRate, 0); err != nil {
			return report, err
		}
		pcmBuffer = bufio.NewWriterSize(pcmFile, 1<<20)
		pcm.output = pcmBuffer
		a.sound.capture = &pcm
	}
	report.Renderer = "Shared native scene compositor; no GPU scaling or application overlays"
	report.Width, report.Height = 320, 200
	report.PALClock, report.FrameClocks, report.AudioRate = audio.PALClock, a.match.PALFrameClocks(), audioRate
	if build, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				report.SourceRevision = setting.Value
			case "vcs.modified":
				report.SourceModified = setting.Value == "true"
			}
		}
	}
	if options.Replay != "" {
		report.ReplaySHA256, err = recordingFileHash(options.Replay)
		if err != nil {
			return report, err
		}
	}
	previousPhase, previousScene, previousRound := "", "", -1
	for frame := 0; frame < options.SmokeTicks; frame++ {
		if err := a.Update(); err != nil {
			return report, fmt.Errorf("update %d: %w", frame, err)
		}
		pixels := a.scenePixels()
		a.recorder.writePixels(pixels)
		if a.recorder.err != nil {
			return report, a.recorder.err
		}
		report.FrameSHA256 = append(report.FrameSHA256, fmt.Sprintf("%x", sha256.Sum256(pixels)))
		g := a.match
		phase := g.Phase.String()
		scene := a.recordingScene()
		if phase != previousPhase || scene != previousScene || g.Round != previousRound {
			report.Milestones = append(report.Milestones, RecordingMilestone{
				VideoFrame: frame, GameFrame: g.Frame, Phase: phase, Round: g.Round, WhiteScore: g.Fighters[0].Score,
				Scene: scene, Mode: g.OriginalMode(), WhiteBelt: strings.TrimSpace(g.Fighters[0].Belt()), Points: [3]int{g.Fighters[0].Points, g.Fighters[1].Points, g.Fighters[2].Points},
				BonusNumber: g.Bonus.Number, BonusTotal: g.Bonus.Total, Deflected: g.Bonus.Deflected, BonusSuccess: g.Bonus.Success,
			})
			previousPhase, previousScene, previousRound = phase, scene, g.Round
		}
	}
	if err := a.FinishRecording(); err != nil {
		return report, err
	}
	report.Frames, report.AudioSamples, report.Final = a.recorder.frames, pcm.bytes/4, a.CheckReport()
	if pcmFile != nil {
		if err := pcmBuffer.Flush(); err != nil {
			return report, err
		}
		if _, err := pcmFile.Seek(0, io.SeekStart); err != nil {
			return report, err
		}
		if err := sound.WritePCMHeader(pcmFile, audioRate, report.AudioSamples); err != nil {
			return report, err
		}
		if err := pcmFile.Close(); err != nil {
			return report, err
		}
		report.AudioSHA256, err = recordingFileHash(audioPath)
		if err != nil {
			return report, err
		}
	}
	report.VideoSHA256, err = recordingFileHash(videoPath)
	return report, err
}

func (a *App) recordingScene() string {
	if a.presentation == nil {
		return "arena"
	}
	if a.presentation.title != nil {
		return "title"
	}
	if a.presentation.flow != nil {
		return "scores"
	}
	if a.presentation.humanJoin != nil {
		return "join"
	}
	return "arena-transition"
}

func recordingFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
