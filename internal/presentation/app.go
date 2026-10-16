// Package presentation connects the original artwork and native match to
// Ebitengine. Desktop and Android use the same update and rendering paths.
package presentation

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/ikplus/assets"
	"github.com/olivierh59500/ikplus/internal/audio"
	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/pilot"
)

// Options configures the shared application; verification runs are reproducible
// and can hold their final frame on Android for external screen capture.
type Options struct {
	Scale                     int
	Seed                      uint64
	Muted, Mobile, Fullscreen bool
	DataDir                   string
	SmokeTicks                int
	StartBonus                int
	FreezeCheck               bool
	Capture                   string
	Attract                   bool
	ScoreMenu                 bool
	Autoplay                  bool
	Record                    string
	Replay                    string
	AudioOptions              bool
	PALFrameLines             int
}

type view uint8

// UpdateRate leaves UI polling independent of the exact PAL simulation rate.
const UpdateRate = 60

const (
	titleView view = iota
	matchView
	helpView
	scoresView
)

// App owns resources that must be created after the platform graphics context.
type App struct {
	options                          Options
	match                            *game.Game
	view, returnView                 view
	players, selection               int
	paused, muted                    bool
	assetCount                       int
	controls                         controls
	saved                            savedData
	saveError                        error
	scoreRecorded                    bool
	eliminatedScores                 [2]*score
	ticks                            int
	palPhase                         uint64
	verificationDone, captureWritten bool
	captureError                     error
	lastPhase                        game.Phase
	animations                       []animation
	sound                            *soundPlayer
	ambient                          ambientState
	scene                            originalScene
	pilot                            *pilot.Pilot
	recorder                         *frameRecorder
	presentation                     *presentationCycle
	scrollKey                        byte
	replay                           *inputReplay
	replayOrigin                     uint64
	audioOptions                     bool
	titleColors                      [16]uint16
	handoffPixels                    []byte
}

// New decodes the embedded disk-derived assets once; playing never needs the
// ADF, Kickstart ROM, Python, or an Amiga emulator.
func New(options Options) (*App, error) {
	return newApp(options, true)
}

// Offline exports use the same startup and updates without opening a sound
// device. Images remain software-composed until Draw uploads them to the GPU.
func newApp(options Options, soundDevice bool) (*App, error) {
	var replay *inputReplay
	if options.Replay != "" {
		var err error
		replay, err = loadInputReplay(options.Replay)
		if err != nil {
			return nil, err
		}
		options.Seed = replay.Seed
	}
	cold := coldStartupRequested(options, replay)
	var match *game.Game
	if cold {
		match = game.NewCold(options.Seed)
	} else {
		if options.Seed == 0 {
			options.Seed = 1988 // Existing direct-game verification profile.
		}
		match = game.New(options.Seed)
	}
	a := &App{options: options, match: match, players: 1, audioOptions: options.AudioOptions}
	profile := audio.Supplied313
	if replay != nil && replay.PALFrameLines != 313 {
		// Historical input references declare a controlled 312-line field.
		profile = audio.Controlled312
	}
	switch options.PALFrameLines {
	case 0:
	case 312:
		profile = audio.Controlled312
	case 313:
		profile = audio.Supplied313
	default:
		return nil, fmt.Errorf("PAL field must contain 312 or 313 lines")
	}
	if err := a.match.SetPALFrameProfile(profile); err != nil {
		return nil, err
	}
	a.match.UseNativePlayback()
	a.replay = replay
	a.pilot = pilot.New(0)
	a.saved = loadData(options.DataDir)
	a.muted = options.Muted || a.saved.Muted
	a.CancelInput()
	names, err := assets.Names()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if strings.HasSuffix(name, ".png") {
			a.assetCount++
		}
	}
	if raw, err := assets.Read("original/animations.json"); err == nil {
		if err := json.Unmarshal(raw, &a.animations); err != nil {
			return nil, err
		}
	}
	if len(a.animations) < 41 {
		return nil, fmt.Errorf("missing original animation sequences; run make assets")
	}
	if err := a.ambient.load(options.Seed); err != nil {
		return nil, err
	}
	initializeScoreHistory(a.ambient.planes)
	a.match.SetBootMarker(0x01010202)
	a.sound, err = newSoundPlayer(a.match, a.muted, soundDevice)
	if err != nil {
		return nil, err
	}
	if cold {
		a.coldStart()
	}
	if replay == nil && (options.Autoplay || options.SmokeTicks > 0 && !options.Attract || options.StartBonus > 0) {
		a.start(1)
	}
	if replay != nil && !replay.Demo {
		a.start(replay.Players)
	}
	// Replay input zero belongs to the first requested update. A physical
	// cold pending IRQ has already run before that update starts.
	a.replayOrigin = a.match.Frame
	if options.Record != "" {
		width, height := a.Layout(0, 0)
		a.recorder, err = newFrameRecorder(options.Record, width, height, a.match.PALFrameClocks())
		if err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *App) start(players int) {
	a.handoffPixels = nil
	a.endPresentation(false)
	a.players, a.view, a.paused, a.scoreRecorded = players, matchView, false, false
	a.eliminatedScores = [2]*score{}
	a.pilot = pilot.New(0)
	if a.options.Autoplay || a.options.SmokeTicks > 0 || a.options.StartBonus != 0 || a.replay != nil {
		a.match.Start(players)
	} else {
		a.match.Join(players)
	}
	if a.options.StartBonus != 0 {
		a.match.StartBonus(a.options.StartBonus-1, 0)
	}
	a.CancelInput()
}

func (a *App) Update() error {
	if a.recorder != nil {
		if a.recorder.err != nil {
			return a.recorder.err
		}
		// Offline recording requires exactly one rendered frame per PAL tick.
		if a.ticks > a.recorder.frames {
			return nil
		}
	}
	if a.verificationDone {
		if a.captureError != nil {
			return fmt.Errorf("capture verification frame: %w", a.captureError)
		}
		if a.options.Capture != "" && !a.captureWritten {
			return nil
		}
		if !a.options.FreezeCheck {
			return ebiten.Termination
		}
		return nil
	}
	if a.options.SmokeTicks == 0 {
		frozen := a.match.SourceFrozen()
		if just(ebiten.KeyF11) {
			ebiten.SetFullscreen(!ebiten.IsFullscreen())
		}
		if (!frozen || a.view == helpView || a.paused) && just(ebiten.KeyEscape) {
			a.Back()
		}
		ambientKey := a.ambient.pollKeys(func(code byte) {
			// S is the red joystick's down direction during two-player play.
			// Ctrl+S keeps the original floor option available in that layout.
			fastWord := a.ambient.history&0xffff7f == 0xa19e1f
			if code&0x7f == 0x1f && !frozen && a.match.Fighters[1].Human && !ebiten.IsKeyPressed(ebiten.KeyControl) && !fastWord {
				return
			}
			a.match.QueueSourceKey(code)
		}, frozen)
		if !frozen {
			if just(ebiten.KeyM) && a.ambient.history != 0xa2929332 {
				a.toggleSound()
			}
			if just(ebiten.KeyH) && !ambientKey {
				if a.view == helpView {
					a.Back()
				} else {
					a.openHelp()
				}
			}
			if just(ebiten.KeyP) && !ambientKey {
				if a.match.Phase != game.PauseDemo {
					a.togglePause()
				}
			}
			if just(ebiten.KeyPause) {
				a.togglePause()
			}
			if just(ebiten.KeyQ) && !ambientKey && a.AtTitle() && !a.options.Mobile {
				return ebiten.Termination
			}
		}
		a.menuInput()
	}
	var inputs [2]game.Input
	if a.options.SmokeTicks == 0 {
		inputs = a.readInput()
	}
	if a.replay != nil {
		frame := a.match.Frame - a.replayOrigin
		if frame < uint64(len(a.replay.Inputs)) {
			inputs = replayJoystick(a.replay.Inputs[frame])
		} else {
			inputs = [2]game.Input{}
		}
		if frame < uint64(len(a.replay.Keys)) && a.replay.Keys[frame] != 0 {
			a.match.QueueSourceKey(a.replay.Keys[frame])
		}
	}
	if a.replay == nil && a.view == matchView && (a.options.Autoplay || a.options.SmokeTicks > 0 && !a.options.Attract) {
		inputs = [2]game.Input{a.pilot.Input(a.match), {}}
	}
	if (a.view == matchView && !a.paused || a.view == titleView || a.view == scoresView && a.presentation != nil) && a.palTickDue() {
		a.handoffPixels = nil
		if a.presentation != nil {
			a.tickPresentation(inputs)
		} else {
			before := a.match.Ambient()
			drawing, _ := a.match.SceneryBuffers()
			a.match.Tick(inputs)
			if a.match.RestartRequested() {
				a.warmRestart()
			} else if players := a.match.JoinRequested(); players != 0 {
				a.sourceJoin(players)
			} else {
				a.captureEliminatedScores()
				a.ambient.syncSource(before, a.match.Ambient(), a.match.Frame, a.match.OriginalVBL(), a.match.ReflectionPhases(), drawing, a.match.SourceFrozen())
				if a.options.ScoreMenu && a.match.Frame == 60 {
					a.openScores()
				} else if a.match.Phase == game.GameOver && !a.options.Autoplay && (a.options.SmokeTicks == 0 || a.options.Attract || a.replay != nil) {
					if a.view == matchView {
						a.recordScores()
					}
					a.beginPresentation()
				}
			}
		}
		if err := a.sound.update(a.match); err != nil {
			return err
		}
	}
	if a.match.Phase == game.GameOver && a.view == matchView {
		a.recordScores()
	}
	a.sound.setPaused(a.paused || a.view == helpView || a.view == scoresView && a.presentation == nil)
	a.lastPhase = a.match.Phase
	a.ticks++
	if a.options.SmokeTicks > 0 && a.ticks >= a.options.SmokeTicks {
		a.verificationDone = true
	}
	return nil
}

func (a *App) palTickDue() bool {
	if a.options.SmokeTicks > 0 || a.recorder != nil {
		return true
	}
	a.palPhase += audio.PALClock
	frameClocks := audio.PALFrameClocks
	if a.match != nil {
		frameClocks = a.match.PALFrameClocks()
	}
	period := uint64(UpdateRate) * frameClocks
	if a.palPhase < period {
		return false
	}
	a.palPhase -= period
	return true
}

func (a *App) menuInput() {
	x, y, click := pointerPress()
	offset := a.offset()
	frozen := a.match.SourceFrozen()
	if a.options.Mobile && click {
		if x > 404 && x < 476 {
			if a.audioOptions {
				if y >= 90 && y <= 122 {
					a.toggleSound()
				}
				if y >= 214 && y <= 246 {
					a.setAudioOptions(false)
				}
				return
			}
			switch {
			case y >= 10 && y <= 42 && a.view == matchView && a.presentation == nil:
				a.togglePause()
				return
			case y >= 50 && y <= 82:
				a.Back()
				return
			case y >= 90 && y <= 122:
				a.setAudioOptions(true)
				return
			}
		}
	}
	switch a.view {
	case titleView:
		if a.enteringName() {
			return
		}
		if !frozen && (a.presentation == nil || a.presentation.flow == nil) && just(ebiten.KeyUp, ebiten.KeyDown) {
			a.selection = (a.selection + 1) % 2
		}
		if !frozen && just(ebiten.KeyEnter) {
			a.requestHumanStart(a.selection + 1)
		}
		if action, ok := titleActionAt(a.options.Mobile, x, y); click && ok {
			switch action {
			case titleOnePlayer:
				a.requestHumanStart(1)
			case titleTwoPlayers:
				a.requestHumanStart(2)
			case titleMoves:
				a.openHelp()
			case titleScores:
				a.openScores()
			}
		}
	case matchView:
		if a.presentation != nil {
			return
		} else if a.match.Phase == game.GameOver {
			if !frozen && just(ebiten.KeyEnter, ebiten.KeySpace) || click && x >= offset && x < offset+320 {
				a.Back()
			}
		} else if a.paused {
			if just(ebiten.KeyEnter, ebiten.KeySpace) || click && x >= offset+40 && x <= offset+280 && y >= 114 && y <= 150 {
				a.togglePause()
			}
			if click && x >= offset+40 && x <= offset+280 && y >= 160 && y <= 192 {
				a.Back()
			}
		}
	case scoresView:
		if !frozen && just(ebiten.KeyEnter) || click && x >= offset && x < offset+320 && y > 230 {
			a.Back()
		}
	case helpView:
		if just(ebiten.KeyEnter, ebiten.KeySpace) || click && x >= offset && x < offset+320 && y > 230 {
			a.Back()
		}
	}
}

func (a *App) openHelp() { a.returnView = a.view; a.view = helpView; a.CancelInput() }

func (a *App) togglePause() {
	if a.view != matchView || a.match.Phase == game.GameOver {
		return
	}
	if a.paused {
		a.paused = false
	} else if a.match.OriginalMode() == 1 || a.match.OriginalMode() == 6 {
		a.match.RequestPause()
	} else {
		a.paused = true
	}
	a.CancelInput()
}

func (a *App) toggleSound() { a.muted = !a.muted; a.sound.setMuted(a.muted); a.save() }

// Suspend is called on Android focus loss; only the update thread mutates state.
func (a *App) Suspend() {
	if a.view == matchView && a.match.Phase != game.GameOver {
		a.paused = true
	}
	a.CancelInput()
	a.sound.setPaused(true)
	a.save()
}

// Back implements contextual navigation for Escape and Android's Back gesture.
func (a *App) Back() {
	if a.audioOptions {
		a.setAudioOptions(false)
		return
	}
	if a.presentation != nil && (a.view == matchView || a.view == titleView || a.view == scoresView) {
		a.endPresentation(true)
		a.CancelInput()
		return
	}
	switch a.view {
	case helpView, scoresView:
		a.view = a.returnView
	case matchView:
		if !a.paused && !a.match.PauseRequested() && a.match.Phase != game.GameOver && a.match.Phase != game.PauseDemo {
			a.togglePause()
		} else {
			a.recordScores()
			a.view, a.paused = titleView, false
			a.match.ReturnToDemo()
		}
	}
	a.CancelInput()
}

func (a *App) AtTitle() bool          { return a.view == titleView }
func (a *App) VerificationDone() bool { return a.verificationDone }
func (a *App) Layout(_, _ int) (int, int) {
	if a.options.Mobile {
		return 480, 256
	}
	return 320, 256
}
func (a *App) offset() int {
	if a.options.Mobile {
		return 80
	}
	return 0
}

func (a *App) CheckReport() string {
	f := a.match.Fighters[0]
	return fmt.Sprintf("ticks=%d phase=%s round=%d points=%d score=%d assets=%d touch=%t", a.ticks, a.match.Phase, a.match.Round, f.Points, f.Score, a.assetCount, a.options.Mobile)
}

func (a *App) capture(screen *ebiten.Image) {
	if !a.verificationDone || a.captureWritten || a.options.Capture == "" {
		return
	}
	a.captureWritten = true
	if err := os.MkdirAll(filepath.Dir(a.options.Capture), 0o755); err != nil {
		a.captureError = err
		return
	}
	file, err := os.Create(a.options.Capture)
	if err != nil {
		a.captureError = err
		return
	}
	// ReadPixels captures the same framebuffer used by both platform renderers.
	bounds := screen.Bounds()
	pixels := make([]byte, 4*bounds.Dx()*bounds.Dy())
	screen.ReadPixels(pixels)
	a.captureError = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 4 * bounds.Dx(), Rect: bounds})
	if err := file.Close(); a.captureError == nil {
		a.captureError = err
	}
}

func (a *App) Close() {
	_ = a.FinishRecording()
	a.save()
	if a.sound != nil {
		a.sound.close()
	}
}
