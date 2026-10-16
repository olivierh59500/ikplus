package presentation

import (
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/olivierh59500/ikplus/assets"
)

type hudState struct {
	Points, Scores                                    [3]int
	Round, Seconds, Mode, Speed, Humans, VBL, Counter int
	Owners                                            [2]bool
	German                                            bool
}

type hudDraw struct {
	Glyph, Color, X, Y int
	Image              string
}

type hudData struct {
	Palette   [16]uint16 `json:"palette"`
	Points    [3]int     `json:"point_columns"`
	Scores    [3]int     `json:"score_columns"`
	Colors    [3]int     `json:"player_colors"`
	Empty     [3]int     `json:"empty_point_colors"`
	Countdown [2]int     `json:"countdown_colors"`
	Pause     [2]int     `json:"pause_colors"`
}

var originalHUD = func() hudData {
	raw, err := assets.Read("original/hud.json")
	if err != nil {
		panic(err)
	}
	var data hudData
	if err = json.Unmarshal(raw, &data); err != nil {
		panic(err)
	}
	return data
}()

// hudLayout follows $6AD8/$7CCE/$7D44. It emits the same eight-row cells
// used by the runtime and independent pixel tests, including leading blanks.
func hudLayout(s hudState) []hudDraw {
	draws := make([]hudDraw, 0, 52)
	glyph := func(index, column, ink int) {
		draws = append(draws, hudDraw{Glyph: index & 63, Color: ink & 15, X: (column % 40) * 8, Y: (column / 40) * 8})
	}
	for player := 0; player < 3; player++ {
		for point := 0; point < 6; point++ {
			ink := originalHUD.Empty[player]
			if point < s.Points[player] {
				ink = originalHUD.Colors[player]
			}
			glyph(12, originalHUD.Points[player]+point, ink)
		}
		digits := fmt.Sprintf("%06d", s.Scores[player]%1000000)
		started := false
		for digit, ch := range digits {
			index := int(ch - '0')
			if index != 0 {
				started = true
			}
			if !started && digit < 5 {
				index = 10
			}
			glyph(index, originalHUD.Scores[player]+digit, originalHUD.Colors[player])
		}
	}
	glyph(28, 28, 12)
	glyph(38, 29, 12)
	glyph(s.Round/10%10, 68, 12)
	glyph(s.Round%10, 69, 12)
	label, ink := "TIME=", 6
	if s.German {
		label = "ZEIT="
	}
	if s.Humans == 0 {
		label, ink = "DEMO ", 3
	}
	if s.Mode == 6 {
		label = "PAUSE"
		ink = originalHUD.Pause[(s.VBL>>4)&1]
	}
	for column, ch := range label {
		glyph(int(ch)-48, 32+column, ink)
	}
	ink = 6
	if s.Mode == 1 && s.Seconds < 6 {
		ink = originalHUD.Countdown[(s.Counter>>2)&1]
	}
	glyph(s.Seconds/10%10, 38, ink)
	glyph(s.Seconds%10, 39, ink)
	draws = append(draws, hudDraw{Image: "hud-speed-label", X: 256, Y: 10}, hudDraw{Image: fmt.Sprintf("hud-speed-%d", min(4, max(0, s.Speed))), X: 288, Y: 10})
	for player, shown := range s.Owners {
		if shown {
			draws = append(draws, hudDraw{Image: fmt.Sprintf("hud-owner-%d", player), X: 48 + player*80})
		}
	}
	return draws
}

func hudInk(index int) color.RGBA {
	value := originalHUD.Palette[index&15]
	return color.RGBA{byte(value>>8&15) * 17, byte(value>>4&15) * 17, byte(value&15) * 17, 255}
}

func (a *App) currentHUDState() hudState {
	g := a.match
	seconds, mode := g.DisplayClock()
	state := hudState{Round: g.DisplayRound(), Seconds: seconds, Mode: int(mode), Speed: g.DisplaySpeed(), VBL: int(g.DisplayVBL()), Counter: int(g.DisplayCounter()), Owners: g.OwnerIndicators(), German: g.DisplayGermanText()}
	for id, f := range g.Fighters {
		state.Points[id], state.Scores[id] = f.Points, f.Score
		if id < 2 && f.Human {
			state.Humans |= 1 << id
		}
	}
	if a.paused {
		state.Mode = 6
	}
	return state
}
