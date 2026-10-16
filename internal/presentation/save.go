package presentation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

type score struct {
	Points int    `json:"points"`
	Round  int    `json:"round"`
	Belt   string `json:"belt"`
}

type savedData struct {
	Scores []score `json:"scores"`
	Muted  bool    `json:"muted"`
}

func loadData(directory string) savedData {
	var data savedData
	raw, err := os.ReadFile(filepath.Join(directory, "scores.json"))
	if err == nil {
		_ = json.Unmarshal(raw, &data)
	}
	if len(data.Scores) > 10 {
		data.Scores = data.Scores[:10]
	}
	return data
}

// save uses an atomic rename so an interrupted mobile lifecycle leaves the
// previous score file intact. Verification runs never modify personal scores.
func (a *App) save() {
	if a.options.SmokeTicks > 0 || a.options.Autoplay || a.options.Replay != "" {
		return
	}
	a.saved.Muted = a.muted
	if err := os.MkdirAll(a.options.DataDir, 0o755); err != nil {
		a.saveError = err
		return
	}
	raw, err := json.MarshalIndent(a.saved, "", "  ")
	if err != nil {
		a.saveError = err
		return
	}
	file := filepath.Join(a.options.DataDir, "scores.json")
	if err = os.WriteFile(file+".tmp", append(raw, '\n'), 0o600); err == nil {
		err = os.Rename(file+".tmp", file)
	}
	a.saveError = err
}

func (a *App) recordScores() {
	if a.scoreRecorded || a.options.SmokeTicks > 0 || a.options.Autoplay || a.options.Replay != "" {
		return
	}
	a.scoreRecorded = true
	for i, f := range a.match.Fighters {
		// Elimination changes Human to false; the original player slot still
		// owns its earned score when the last surviving player loses.
		if i < a.players && f.Score > 0 {
			entry := score{Points: f.Score, Round: a.match.Round, Belt: f.Belt()}
			if a.eliminatedScores[i] != nil {
				entry = *a.eliminatedScores[i]
			}
			a.saved.Scores = append(a.saved.Scores, entry)
		}
	}
	sort.SliceStable(a.saved.Scores, func(i, j int) bool { return a.saved.Scores[i].Points > a.saved.Scores[j].Points })
	if len(a.saved.Scores) > 10 {
		a.saved.Scores = a.saved.Scores[:10]
	}
	a.save()
}

// Subsequent AI points belong to the computer replacement, so retain the
// human's score at the instant elimination occurs during a two-player match.
func (a *App) captureEliminatedScores() {
	for i := 0; i < a.players; i++ {
		f := a.match.Fighters[i]
		if f.Eliminated && a.eliminatedScores[i] == nil {
			entry := score{Points: f.Score, Round: a.match.Round, Belt: f.Belt()}
			a.eliminatedScores[i] = &entry
		}
	}
}
