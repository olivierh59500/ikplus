// Command game-inputs records legal joystick samples for whole-program replay.
// Expected native state is deliberately absent from the resulting recording.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/pilot"
)

type recording struct {
	Version, TickRate, Players, StopRound int
	Seed                                  uint64
	Inputs                                [][2]byte
	AudioSequenced                        bool `json:",omitempty"`
}

func main() {
	seed := flag.Uint64("seed", 15, "fresh-match seed")
	players := flag.Int("players", 1, "one or two human controllers")
	ticks := flag.Int("ticks", 30000, "maximum PAL samples")
	round := flag.Int("round", 13, "stop on entering this displayed round")
	pattern := flag.String("pattern", "pilot", "pilot, neutral, or cycle")
	pilotPlayer := flag.Int("pilot-player", -1, "pilot controller slot (0 or 1); -1 drives every human slot")
	audio := flag.Bool("audio", false, "run original shared audio/RNG while recording legal joystick inputs")
	out := flag.String("out", ".cache/verification/game-inputs.json", "legal input recording")
	flag.Parse()
	if *seed == 0 || *players < 1 || *players > 2 || *ticks < 1 || *round < 2 || *pilotPlayer < -1 || *pilotPlayer >= *players || (*pattern != "pilot" && *pattern != "neutral" && *pattern != "cycle") {
		fmt.Fprintln(os.Stderr, "invalid seed, players, ticks, round or pattern")
		os.Exit(2)
	}
	g := game.New(*seed)
	g.Start(*players)
	if *audio {
		if err := g.EnableAudio(); err != nil {
			panic(err)
		}
	}
	controllers := [2]*pilot.Pilot{pilot.New(0), pilot.New(1)}
	record := recording{Version: 1, TickRate: game.TickRate, Players: *players, StopRound: *round, Seed: *seed, AudioSequenced: *audio}
	directions := [9]game.Input{{}, {X: 1}, {X: 1, Y: -1}, {Y: -1}, {X: -1, Y: -1}, {X: -1}, {X: -1, Y: 1}, {Y: 1}, {X: 1, Y: 1}}
	for tick := 0; tick < *ticks && g.Phase != game.GameOver && g.Round < *round; tick++ {
		var inputs [2]game.Input
		decoded := [2]byte{15, 15}
		for id := 0; id < *players; id++ {
			switch *pattern {
			case "pilot":
				if *pilotPlayer < 0 || *pilotPlayer == id {
					inputs[id] = controllers[id].Input(g)
				}
			case "cycle":
				inputs[id] = directions[tick/15%9]
				inputs[id].Fire = tick/135%2 != 0
			}
			in := inputs[id]
			if in.X < 0 {
				decoded[id] &^= 4
			}
			if in.X > 0 {
				decoded[id] &^= 8
			}
			if in.Y < 0 {
				decoded[id] &^= 1
			}
			if in.Y > 0 {
				decoded[id] &^= 2
			}
			if in.Fire {
				decoded[id] |= 16
			}
		}
		record.Inputs = append(record.Inputs, decoded)
		g.Tick(inputs)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0755); err != nil {
		panic(err)
	}
	file, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	if err = json.NewEncoder(file).Encode(record); err != nil {
		panic(err)
	}
	if err = file.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("Recorded seed%d players%d pattern%s samples%d final round%d phase%s score%d: %s\n", *seed, *players, *pattern, len(record.Inputs), g.Round, g.Phase, g.Fighters[0].Score, *out)
}
