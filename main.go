// IK+ runs the native Go port of the supplied Amiga game.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/ikplus/internal/presentation"
)

func main() {
	options := presentation.Options{}
	flag.IntVar(&options.Scale, "scale", 3, "initial integer window scale")
	flag.Uint64Var(&options.Seed, "seed", 0, "original cold-start RNG words; direct-game verification defaults to 1988")
	flag.IntVar(&options.PALFrameLines, "pal-lines", 0, "PAL field lines: 313 normally, 312 for historical controlled references")
	flag.BoolVar(&options.Muted, "mute", false, "start with audio muted")
	flag.BoolVar(&options.Mobile, "touch", false, "preview Android touch controls")
	flag.BoolVar(&options.AudioOptions, "audio-options", false, "show the touch audio sidebar for verification")
	flag.BoolVar(&options.Fullscreen, "fullscreen", false, "start in fullscreen")
	flag.IntVar(&options.SmokeTicks, "smoke", 0, "run deterministic input and exit after N updates")
	flag.IntVar(&options.StartBonus, "bonus", 0, "start a bonus round: 1 balls, 2 bombs")
	flag.BoolVar(&options.Attract, "attract", false, "keep the bot demonstration running during a smoke capture")
	flag.BoolVar(&options.ScoreMenu, "score-menu", false, "open the original score display after sixty demo PAL frames")
	flag.BoolVar(&options.Autoplay, "autoplay", false, "watch a simulated human play with ordinary joystick inputs")
	flag.StringVar(&options.Record, "record", "", "record each simulated PAL frame to lossless video using ffmpeg")
	flag.StringVar(&options.Replay, "replay", "", "replay legal joystick and supported source keyboard input from JSON")
	flag.StringVar(&options.Capture, "capture", "", "save the final verification frame as PNG")
	flag.StringVar(&options.DataDir, "data-dir", "", "high score and preference directory")
	flag.Parse()
	if options.AudioOptions {
		options.Mobile = true
	}
	if options.Scale < 1 || options.Scale > 8 || options.SmokeTicks < 0 || options.StartBonus < 0 || options.StartBonus > 2 {
		log.Fatal("scale must be 1-8, smoke must be nonnegative, and bonus must be 0-2")
	}
	if options.Capture != "" && options.SmokeTicks == 0 {
		log.Fatal("capture requires a positive smoke update count")
	}
	if options.ScoreMenu {
		options.Attract = true
	}
	if options.Replay != "" && (options.Autoplay || options.StartBonus != 0 || options.ScoreMenu) {
		log.Fatal("input replay cannot be combined with autoplay, bonus practice or score-menu capture")
	}
	if options.Attract && (options.Autoplay || options.StartBonus > 0) {
		log.Fatal("attract cannot be combined with autoplay or bonus practice")
	}
	if options.Record != "" && (options.SmokeTicks <= 0 || options.Mobile) {
		log.Fatal("recording requires positive -smoke ticks and the desktop layout")
	}
	if options.Record != "" {
		options.Muted = true
	}
	if options.DataDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		options.DataDir = filepath.Join(base, "ikplus")
	}
	app, err := presentation.New(options)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()
	ebiten.SetWindowTitle("IK+ — Go / Ebitengine")
	width, height := app.Layout(0, 0)
	ebiten.SetWindowSize(width*options.Scale, height*options.Scale)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(options.Fullscreen)
	ebiten.SetScreenFilterEnabled(false)
	ebiten.SetTPS(presentation.UpdateRate)
	if options.Record != "" {
		ebiten.SetTPS(ebiten.SyncWithFPS)
		ebiten.SetVsyncEnabled(false)
	}
	if options.Record != "" || options.SmokeTicks > 0 {
		// Automated captures must continue when another window takes focus.
		ebiten.SetRunnableOnUnfocused(true)
	}
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
	if err := app.FinishRecording(); err != nil {
		log.Fatal(err)
	}
	if options.SmokeTicks > 0 {
		if !app.VerificationDone() {
			log.Fatalf("verification interrupted: %s", app.CheckReport())
		}
		fmt.Println(app.CheckReport())
	}
}
