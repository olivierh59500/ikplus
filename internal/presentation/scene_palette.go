package presentation

import (
	"encoding/json"
	"github.com/olivierh59500/ikplus/assets"
)

var arenaPalettes = func() [200][16]uint16 {
	data, err := assets.Read("original/palettes.json")
	if err != nil {
		panic(err)
	}
	var colors [200][16]uint16
	if err = json.Unmarshal(data, &colors); err != nil {
		panic(err)
	}
	return colors
}()

// arenaPalette applies the published scene colors directly to a display row.
// Ebitengine receives pixels; no display registers or Copper list are run.
func (p *sceneryPlanes) arenaPalette(y int) [16]uint16 {
	palette := arenaPalettes[y]
	if y >= 77 && y < 115 {
		palette[10] = p.state.GraphicsColor
	}
	if y >= 115 {
		palette[12], palette[13], palette[9] = p.state.VisibleWhiteBelt, p.state.VisibleRedBelt, p.state.VisibleBlueBelt
		if y < 192 {
			palette[1] = p.state.VisibleEffectColor
		}
	}
	if y >= 192 {
		palette[15] = p.state.VisibleGroundShade
	}
	return palette
}

// scenePaletteMode selects an authored color treatment, independent of hardware.
type scenePaletteMode uint8

const (
	paletteUnset scenePaletteMode = iota
	paletteArena
	paletteTitle
	paletteRainbow
)
