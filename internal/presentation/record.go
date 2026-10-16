package presentation

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/ikplus/internal/audio"
)

// frameRecorder streams native pixels to lossless FFV1. Window captures use
// ReadPixels; software exports supply the shared compositor's scene pixels.
// Rational PAL timestamps are independent of offline rendering speed.
type frameRecorder struct {
	command     *exec.Cmd
	input       io.WriteCloser
	log         *os.File
	pixels      []byte
	frames      int
	err         error
	closed      bool
	frameClocks uint64
}

func newFrameRecorder(path string, width, height int, frameClocks uint64) (*frameRecorder, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	log, err := os.Create(path + ".ffmpeg.log")
	if err != nil {
		return nil, err
	}
	frameRate := fmt.Sprintf("%d/%d", audio.PALClock, frameClocks)
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "rawvideo", "-pixel_format", "rgba", "-video_size", fmt.Sprintf("%dx%d", width, height), "-framerate", frameRate, "-i", "pipe:0", "-an", "-c:v", "ffv1", "-level", "3", "-g", "1", "-threads", "2", path)
	cmd.Stderr = log
	input, err := cmd.StdinPipe()
	if err != nil {
		_ = log.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		_ = input.Close()
		_ = log.Close()
		return nil, err
	}
	return &frameRecorder{command: cmd, input: input, log: log, pixels: make([]byte, width*height*4), frameClocks: frameClocks}, nil
}

func (r *frameRecorder) write(screen *ebiten.Image) {
	if r.err != nil {
		return
	}
	screen.ReadPixels(r.pixels)
	r.writePixels(r.pixels)
}

func (r *frameRecorder) writePixels(pixels []byte) {
	if r.err != nil {
		return
	}
	if len(pixels) != len(r.pixels) {
		r.err = fmt.Errorf("record frame %d: received %d pixel bytes, expected %d", r.frames, len(pixels), len(r.pixels))
		return
	}
	written, err := r.input.Write(pixels)
	if err == nil && written != len(pixels) {
		err = io.ErrShortWrite
	}
	if err != nil {
		r.err = fmt.Errorf("record frame %d: %w", r.frames, err)
		return
	}
	r.frames++
	if r.frames%2500 == 0 {
		fmt.Fprintf(os.Stderr, "Recorded %d frames (%.0f seconds of PAL gameplay)\n", r.frames, float64(r.frames)*float64(r.frameClocks)/float64(audio.PALClock))
	}
}

// FinishRecording closes the encoder before a successful export is reported.
func (a *App) FinishRecording() error {
	r := a.recorder
	if r == nil || r.closed {
		return nil
	}
	r.closed = true
	closeErr := r.input.Close()
	encodeErr := r.command.Wait()
	_ = r.log.Close()
	if r.err != nil {
		return r.err
	}
	if closeErr != nil {
		return closeErr
	}
	if encodeErr != nil {
		return fmt.Errorf("finish video encoder: %w", encodeErr)
	}
	if r.frames != a.ticks {
		return fmt.Errorf("recorded %d frames for %d updates", r.frames, a.ticks)
	}
	return nil
}
