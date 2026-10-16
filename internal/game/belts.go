package game

import (
	"encoding/json"

	"github.com/olivierh59500/ikplus/assets"
)

type beltSourceTable struct {
	Thresholds     []int             `json:"thresholds"`
	Labels         []string          `json:"labels"`
	WhiteColors    []uint16          `json:"white_colors"`
	RedColors      []uint16          `json:"red_colors"`
	ComputerColors map[string]uint16 `json:"computer_colors"`
}

var sourceBelts = func() beltSourceTable {
	raw, err := assets.Read("original/belt-tables.json")
	if err != nil {
		panic(err)
	}
	var result beltSourceTable
	if err = json.Unmarshal(raw, &result); err != nil {
		panic(err)
	}
	if len(result.Thresholds) != 23 || len(result.Labels) != 24 || len(result.WhiteColors) != 24 || len(result.RedColors) != 24 {
		panic("invalid original belt table")
	}
	return result
}()

func (f Fighter) beltTier() int {
	tier := 0
	for tier < len(sourceBelts.Thresholds) && f.Score >= sourceBelts.Thresholds[tier] {
		tier++
	}
	return tier
}

// Belt returns the exact original eight-character rank label from $AA12.
func (f Fighter) Belt() string { return sourceBelts.Labels[f.beltTier()] }

// BeltColor returns the original OCS12-bit color, preserving the separate
// white/red human palettes and the three computer fighter belt colors.
func (f Fighter) BeltColor() uint16 {
	if !f.Human {
		return sourceBelts.ComputerColors[[3]string{"white", "red", "blue"}[min(max(f.ID, 0), 2)]]
	}
	if f.ID == 1 {
		return sourceBelts.RedColors[f.beltTier()]
	}
	return sourceBelts.WhiteColors[f.beltTier()]
}

func (g *Game) updateBeltPalette() {
	if g.OriginalMode() == 6 {
		g.beltColors = [3]uint16{0xc00, 0xe6e, 0xcc0}
		return
	}
	for id, fighter := range g.Fighters {
		g.beltColors[id] = fighter.BeltColor()
	}
}

func (g *Game) resetBeltPalette() {
	g.updateBeltPalette()
	g.beltColorsVisible = nativeGameplay.Defaults.Belts
}
