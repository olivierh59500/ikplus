package presentation

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olivierh59500/ikplus/internal/game"
)

// inputReplay contains legal recorded input only, never expected native state.
type inputReplay struct {
	Version, TickRate, Players int
	Seed                       uint64
	Inputs                     [][2]byte
	Keys                       []byte
	Demo                       bool
	ColdStart                  bool
	PALFrameLines              int
}

func loadInputReplay(path string) (*inputReplay, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	r := &inputReplay{}
	if err = json.Unmarshal(raw, r); err != nil {
		return nil, err
	}
	if r.Version != 1 || r.TickRate != 50 || r.Players < 0 || r.Players > 2 || r.Players == 0 && !r.Demo || r.Seed == 0 && !r.ColdStart || r.ColdStart && (!r.Demo || r.Players != 0) || len(r.Inputs) == 0 {
		return nil, fmt.Errorf("invalid replay version, PAL rate, players, seed or empty input")
	}
	if r.PALFrameLines != 0 && r.PALFrameLines != 312 && r.PALFrameLines != 313 {
		return nil, fmt.Errorf("unsupported PAL field length")
	}
	for _, pair := range r.Inputs {
		for _, value := range pair {
			if value > 31 || value&3 == 0 || value&12 == 0 {
				return nil, fmt.Errorf("replay contains conflicting joystick directions")
			}
		}
	}
	for _, key := range r.Keys {
		if !game.ValidSourceKey(key) {
			return nil, fmt.Errorf("replay contains an unsupported keyboard command")
		}
	}
	return r, nil
}

func replayJoystick(pair [2]byte) [2]game.Input {
	var input [2]game.Input
	for id, value := range pair {
		input[id].Fire = value&16 != 0
		input[id].X = axis(value&4 == 0, value&8 == 0)
		input[id].Y = axis(value&1 == 0, value&2 == 0)
	}
	return input
}
