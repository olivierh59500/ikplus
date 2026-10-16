package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/ikplus/assets"
	"github.com/olivierh59500/ikplus/internal/game"
)

// ambientState presents the original scenery planes. Runtime logical state
// comes from Game; the local RNG/object adapter supports isolated CPU checks.
type ambientState struct {
	game.AmbientState
	palettes       [][]uint16
	random         uint64
	displayTicks   byte
	history        uint32
	planes         *sceneryPlanes
	lastFrame      uint64
	videoBank      int
	lastSceneClear uint64
}

func (a *ambientState) load(seed uint64) error {
	if err := a.loadData(seed); err != nil {
		return err
	}
	a.planes = newScenePlanes()
	return nil
}

// loadData also supports verification without allocating a graphics resource.
func (a *ambientState) loadData(seed uint64) error {
	raw, err := assets.Read("original/palettes.json")
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &a.palettes); err != nil {
		return err
	}
	if len(a.palettes) != 200 {
		return fmt.Errorf("invalid recovered arena palettes")
	}
	a.random = seed
	return nil
}

func (a *ambientState) sync(before, state game.AmbientState, frame uint64, phases [21]byte, drawing int, frozen ...bool) {
	a.syncSource(before, state, frame, byte(frame), phases, drawing, frozen...)
}

func (a *ambientState) syncSource(before, state game.AmbientState, frame uint64, sourceFrame byte, phases [21]byte, drawing int, frozen ...bool) {
	if frame <= a.lastFrame {
		a.planes = newScenePlanes()
		a.lastSceneClear = 0
	}
	if len(frozen) == 0 || !frozen[0] {
		a.planes.tick(before, state, sourceFrame, phases, drawing, true)
	}
	a.AmbientState = state
	a.displayTicks = sourceFrame
	a.lastFrame = frame
	a.videoBank = drawing
}

// keySequence checks the game's compact key history. FISH launches the jumping
// fish and BIRD launches its flying silhouette. PAC/PERI retain source requests.
func (a *ambientState) keySequence(history uint32) bool {
	return a.AmbientState.KeySequence(history)
}

// pollKeys retains the physical history for platform shortcut arbitration.
// Game.pollSpeedKey owns original command gates, effects and key consumption.
func (a *ambientState) pollKeys(speedKey func(byte), suspended ...bool) bool {
	commands := len(suspended) == 0 || !suspended[0]
	codes := map[ebiten.Key]byte{
		ebiten.KeyMinus: 0x0c, ebiten.KeyEqual: 0x0d, ebiten.KeyNumpadSubtract: 0x4a, ebiten.KeyNumpadAdd: 0x4e,
		ebiten.KeyDigit1: 0x02, ebiten.KeyDigit2: 0x03, ebiten.KeyDigit3: 0x04, ebiten.KeyDigit4: 0x05, ebiten.KeyDigit5: 0x06,
		ebiten.KeyDigit6: 0x07, ebiten.KeyDigit7: 0x08, ebiten.KeyDigit8: 0x09, ebiten.KeyDigit9: 0x0a, ebiten.KeyDigit0: 0x0b,
		ebiten.KeySpace: 0x39, ebiten.KeyPause: 0x39, ebiten.KeyEnter: 0x1c, ebiten.KeyBackspace: 0x0e, ebiten.KeyEscape: 0x01, ebiten.KeyTab: 0x0f,
		ebiten.KeyArrowUp: 0x48, ebiten.KeyArrowDown: 0x50, ebiten.KeyArrowRight: 0x4d, ebiten.KeyArrowLeft: 0x4b,
		ebiten.KeyF1: 0x3b, ebiten.KeyF2: 0x3c, ebiten.KeyF3: 0x3d, ebiten.KeyF4: 0x3e, ebiten.KeyF5: 0x3f,
		ebiten.KeyF6: 0x40, ebiten.KeyF7: 0x41, ebiten.KeyF8: 0x42, ebiten.KeyF9: 0x43, ebiten.KeyF10: 0x44,
		ebiten.KeyF12: 0x62,
		ebiten.KeyQ:   0x10, ebiten.KeyW: 0x11, ebiten.KeyE: 0x12, ebiten.KeyR: 0x13, ebiten.KeyT: 0x14,
		ebiten.KeyY: 0x15, ebiten.KeyU: 0x16, ebiten.KeyI: 0x17, ebiten.KeyO: 0x18, ebiten.KeyP: 0x19,
		ebiten.KeyA: 0x1e, ebiten.KeyS: 0x1f, ebiten.KeyD: 0x20, ebiten.KeyF: 0x21, ebiten.KeyG: 0x22,
		ebiten.KeyH: 0x23, ebiten.KeyJ: 0x24, ebiten.KeyK: 0x25, ebiten.KeyL: 0x26,
		ebiten.KeyZ: 0x2c, ebiten.KeyX: 0x2d, ebiten.KeyC: 0x2e, ebiten.KeyV: 0x2f,
		ebiten.KeyB: 0x30, ebiten.KeyN: 0x31, ebiten.KeyM: 0x32,
	}
	for _, key := range inpututil.AppendJustReleasedKeys(nil) {
		if code, ok := codes[key]; ok {
			a.history = a.history&0xffffff00 | uint32(code|0x80)
			speedKey(code | 0x80)
		}
	}
	handled := false
	for _, key := range inpututil.AppendJustPressedKeys(nil) {
		if code, ok := codes[key]; ok {
			a.history = a.history<<8 | uint32(code)
			speedKey(code)
			if commands {
				handled = game.IsHintSequence(a.history) || a.history == 0xa1979f23 || handled
			}
		}
	}
	return handled
}
