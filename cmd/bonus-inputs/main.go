// Command bonus-inputs records ordinary joystick input for independent replay
// by tools/bonus-oracle. The recording contains no expected native game state.
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
	Version, TickRate   int
	Kind, Humans, Frame int
	Seed                uint64
	Practice            bool
	Inputs              [][2]byte
}

func main() {
	kind := flag.Int("kind", 0, "bonus: 0 balls, 1 bombs")
	seed := flag.Uint64("seed", 17, "reproducible fresh-match seed")
	players := flag.Int("players", 1, "one or two simulated human players")
	maxTicks := flag.Int("max-ticks", 20000, "maximum PAL input samples")
	out := flag.String("out", ".cache/verification/bonus-inputs.json", "legal joystick recording")
	flag.Parse()
	if *kind < 0 || *kind > 1 || *players < 1 || *players > 2 || *maxTicks < 1 || *seed == 0 {
		fmt.Fprintln(os.Stderr, "kind must be 0 or 1, players 1 or 2, and seed/max-ticks positive")
		os.Exit(2)
	}
	g := game.New(*seed)
	g.Start(*players)
	g.StartBonusSequence(*kind)
	controllers := [2]*pilot.Pilot{pilot.New(0), pilot.New(1)}
	record := recording{Version: 1, TickRate: game.TickRate, Kind: *kind, Humans: (1 << *players) - 1, Seed: *seed, Practice: true}
	for tick := 0; tick < *maxTicks && g.Phase != game.Ready; tick++ {
		var inputs [2]game.Input
		var decoded [2]byte
		for id := range inputs {
			inputs[id] = controllers[id].Input(g)
			in, value := inputs[id], byte(15)
			if in.X < 0 {
				value &^= 4
			}
			if in.X > 0 {
				value &^= 8
			}
			if in.Y < 0 {
				value &^= 1
			}
			if in.Y > 0 {
				value &^= 2
			}
			if in.Fire {
				value |= 16
			}
			decoded[id] = value
		}
		record.Inputs = append(record.Inputs, decoded)
		g.Tick(inputs)
	}
	if g.Phase != game.Ready {
		fmt.Fprintln(os.Stderr, "bonus sequence did not finish within max-ticks")
		os.Exit(1)
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
	fmt.Printf("Recorded kind%d seed%d players%d ticks%d final launches%d success%t scores%d/%d: %s\n", *kind, *seed, *players, len(record.Inputs), g.Bonus.Number, g.Bonus.Success, g.Fighters[0].Score, g.Fighters[1].Score, *out)
}
