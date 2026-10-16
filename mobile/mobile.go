//go:build android

// Package mobile connects the shared Go game to Android's Ebitengine view.
package mobile

import (
	"log"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/olivierh59500/ikplus/internal/presentation"
)

// Commands cross the Android UI thread boundary; the update thread owns the app.
type commandKind uint8

const (
	configure commandKind = iota
	verify
	mute
	pause
	back
	cancelInput
)

type command struct {
	kind       commandKind
	directory  string
	ticks      int
	startBonus int
	muted      bool
}

type host struct {
	app          *presentation.App
	commands     chan command
	directory    string
	configured   bool
	verifyTicks  int
	startBonus   int
	muted        bool
	pausePending bool
	reported     bool
	atTitle      atomic.Bool
}

var gameHost = &host{commands: make(chan command, 32)}

func init() {
	// Presentation schedules the rational PAL rate from fixed UI updates.
	ebiten.SetTPS(presentation.UpdateRate)
	ebiten.SetScreenFilterEnabled(false)
	gameHost.atTitle.Store(true)
	enginemobile.SetGame(gameHost)
}

// Configure supplies private storage before Android initializes the game view.
func Configure(directory string) { gameHost.enqueue(command{kind: configure, directory: directory}) }

// ConfigureVerification advances deterministic gameplay and holds its final frame.
func ConfigureVerification(ticks, startBonus int) {
	gameHost.enqueue(command{kind: verify, ticks: ticks, startBonus: startBonus})
}

// ConfigureMuted starts a session silently without changing device volume.
func ConfigureMuted(muted bool) { gameHost.enqueue(command{kind: mute, muted: muted}) }

// RequestPause keeps an interrupted round paused until the player resumes it.
func RequestPause() { gameHost.enqueue(command{kind: pause}) }

// RequestBack routes Android Back through the game's contextual navigation.
func RequestBack() { gameHost.enqueue(command{kind: back}) }

// CancelInput releases unfinished gestures after focus or surface changes.
func CancelInput() { gameHost.enqueue(command{kind: cancelInput}) }

// IsAtTitle lets Android close normally when Back is pressed on the title screen.
func IsAtTitle() bool { return gameHost.atTitle.Load() }

// Dummy provides a stable exported symbol for the Android binding generator.
func Dummy() {}

func (h *host) enqueue(request command) {
	select {
	case h.commands <- request:
	default:
		// Lifecycle callbacks must not block Android's UI thread.
	}
}

func (h *host) Update() error {
	for {
		select {
		case request := <-h.commands:
			switch request.kind {
			case configure:
				if h.app == nil {
					h.directory = request.directory
					h.configured = true
				}
			case verify:
				if h.app == nil {
					h.verifyTicks = max(0, min(request.ticks, 50000))
					h.startBonus = max(0, min(request.startBonus, 2))
				}
			case mute:
				if h.app == nil {
					h.muted = request.muted
				}
			case pause:
				h.pausePending = true
			case back:
				if h.app != nil {
					h.app.Back()
				}
			case cancelInput:
				if h.app != nil {
					h.app.CancelInput()
				}
			}
		default:
			goto ready
		}
	}
ready:
	if h.app == nil {
		if !h.configured {
			return nil
		}
		// Android's JNI context must exist before graphics or audio are created.
		var err error
		h.app, err = presentation.New(presentation.Options{
			Scale: 4, Mobile: true, DataDir: h.directory, Muted: h.muted,
			SmokeTicks: h.verifyTicks, FreezeCheck: true, StartBonus: h.startBonus,
		})
		if err != nil {
			return err
		}
	}
	if h.pausePending {
		h.app.Suspend()
		h.pausePending = false
	}
	if err := h.app.Update(); err != nil {
		return err
	}
	h.atTitle.Store(h.app.AtTitle())
	if h.verifyTicks > 0 && h.app.VerificationDone() && !h.reported {
		log.Printf("ikplus_android_check %s", h.app.CheckReport())
		h.reported = true
	}
	return nil
}

func (h *host) Draw(screen *ebiten.Image) {
	if h.app != nil {
		h.app.Draw(screen)
	}
}

func (h *host) Layout(width, height int) (int, int) {
	if h.app != nil {
		return h.app.Layout(width, height)
	}
	return max(1, width), max(1, height)
}
