// Command verify exercises uninterrupted tournaments through a human joystick.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/pilot"
)

type batchReport struct {
	PALFrameLines                                 int    `json:"pal_frame_lines"`
	Version                                       int    `json:"version"`
	TickRate                                      int    `json:"tick_rate"`
	ReactionTicks                                 int    `json:"reaction_ticks"`
	Controller                                    string `json:"controller"`
	Scope                                         string `json:"scope"`
	Total, ReachedTarget, Eliminated, TickLimited int
	SuccessfulBalls, SuccessfulBombs              int
	Runs                                          []pilot.CampaignReport `json:"runs"`
}

func main() { os.Exit(run()) }

func run() int {
	seedText := flag.String("seeds", "1-5", "Comma-separated seeds or inclusive ranges, such as 1-40")
	singleSeed := flag.Uint64("seed", 1, "Run one seed instead of -seeds")
	targetRound := flag.Int("round", 13, "Stop upon reaching this displayed tournament round")
	maxTicks := flag.Int("max-ticks", 200000, "Maximum PAL ticks per seed; no restarts after defeat")
	palLines := flag.Int("pal-lines", 313, "PAL profile: 313 for native playback, 312 for historical controlled recordings")
	output := flag.String("out", ".cache/verification/campaign.json", "Detailed deterministic campaign report")
	tracePath := flag.String("trace", "", "Optional replay trace in newline-delimited JSON")
	flag.Parse()
	if *targetRound < 2 || *maxTicks < 1 || (*palLines != 312 && *palLines != 313) {
		fmt.Fprintln(os.Stderr, "round must be at least 2, max-ticks positive, and pal-lines 312 or 313")
		return 2
	}
	seeds, err := parseSeeds(*seedText)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seeds = []uint64{*singleSeed}
		}
	})
	var traceFile *os.File
	var traceBuffer *bufio.Writer
	var traceEncoder *json.Encoder
	if *tracePath != "" {
		traceFile, err = createOutput(*tracePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		defer traceFile.Close()
		traceBuffer = bufio.NewWriterSize(traceFile, 256*1024)
		traceEncoder = json.NewEncoder(traceBuffer)
	}
	result := batchReport{Version: 2, TickRate: game.TickRate, ReactionTicks: 5, Controller: "Observable pose/range tactics, measured motion, static artwork and trajectory estimates, and legal X/Y/Fire input; no future game copies, random-state access, or restarts.", Scope: "Native campaign regression evidence. Original routine oracles are separate; this report does not establish complete Amiga match equivalence."}
	result.PALFrameLines = *palLines
	failed := false
	var traceError error
	var summary io.Writer = os.Stdout
	runCampaign := pilot.RunNativeCampaign
	if *palLines == 312 {
		runCampaign = pilot.RunCampaign
	}
	if *output == "-" {
		summary = os.Stderr
	}
	fmt.Fprintln(summary, "SEED  ROUND  SCORE   BELT      FIGHTS  BALLS  BOMBS  RESULT")
	for _, seed := range seeds {
		var trace func(pilot.TraceFrame)
		if traceEncoder != nil {
			trace = func(frame pilot.TraceFrame) {
				if traceError != nil {
					return
				}
				traceError = traceEncoder.Encode(struct {
					Seed uint64 `json:"seed"`
					pilot.TraceFrame
				}{seed, frame})
			}
		}
		report := runCampaign(seed, *targetRound, *maxTicks, trace)
		result.Runs = append(result.Runs, report)
		result.Total++
		balls, bombs := 0, 0
		for _, bonus := range report.Bonuses {
			if !bonus.Success {
				continue
			}
			if bonus.Kind == game.BallBonus.String() {
				balls++
			} else {
				bombs++
			}
		}
		result.SuccessfulBalls += balls
		result.SuccessfulBombs += bombs
		switch report.Status {
		case "reached_target":
			result.ReachedTarget++
		case "eliminated":
			result.Eliminated++
		case "tick_limit":
			result.TickLimited++
		}
		if !report.ReachedTarget {
			failed = true
		}
		fmt.Fprintf(summary, "%-5d %-6d %-7d %-9s %-7d %-6d %-6d %s\n", seed, report.FinalRound, report.Score, strings.TrimSpace(report.Belt), report.CombatRounds, balls, bombs, report.Status)
		if report.InvariantError != "" {
			fmt.Fprintf(os.Stderr, "seed %d: %s\n", seed, report.InvariantError)
		}
	}
	if traceBuffer != nil {
		if err = traceBuffer.Flush(); err != nil && traceError == nil {
			traceError = err
		}
		if err = traceFile.Close(); err != nil && traceError == nil {
			traceError = err
		}
		if traceError != nil {
			fmt.Fprintln(os.Stderr, traceError)
			return 2
		}
	}
	if err = writeReport(*output, result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	fmt.Fprintf(summary, "Reached round %d: %d/%d; completed ball bonuses: %d; completed bomb bonuses: %d. Report: %s\n", *targetRound, result.ReachedTarget, result.Total, result.SuccessfulBalls, result.SuccessfulBombs, *output)
	if failed {
		return 1
	}
	return 0
}

func parseSeeds(text string) ([]uint64, error) {
	var seeds []uint64
	seen := map[uint64]bool{}
	for _, part := range strings.Split(text, ",") {
		bounds := strings.Split(strings.TrimSpace(part), "-")
		if len(bounds) < 1 || len(bounds) > 2 {
			return nil, fmt.Errorf("invalid seed range %q", part)
		}
		first, err := strconv.ParseUint(bounds[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid seed %q", part)
		}
		last := first
		if len(bounds) == 2 {
			last, err = strconv.ParseUint(bounds[1], 10, 64)
		}
		if err != nil || last < first || last-first > 10000 {
			return nil, fmt.Errorf("invalid or excessive seed range %q", part)
		}
		for seed := first; ; seed++ {
			if !seen[seed] {
				seeds = append(seeds, seed)
				seen[seed] = true
			}
			if seed == last {
				break
			}
		}
	}
	return seeds, nil
}

func createOutput(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func writeReport(path string, result batchReport) error {
	var writer io.Writer = os.Stdout
	var file *os.File
	if path != "-" {
		var err error
		file, err = createOutput(path)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return err
	}
	if file != nil {
		return file.Close()
	}
	return nil
}
