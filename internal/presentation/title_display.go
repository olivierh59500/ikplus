package presentation

// pixels expands the title's four packed planes with the colors active
// on each display row, retaining the VBL-published screen through title entry.
// Warm startup briefly uses $BF0; $2D3C adds the title's visible rainbow rows.
func (t *titleSequence) pixels() []byte {
	if t.rgba == nil {
		t.rgba = make([]byte, 320*200*4)
	}
	palette := t.effects.hardware
	p := t.effects.planes
	bank := p.pictureBuffer(p.state.Displayed)
	band := 0
	for y := 0; y < 200; y++ {
		if t.effects.paletteMode == paletteArena {
			palette = p.arenaPalette(y)
		}
		// $2CD0 relocates eight successive gradient rows. Each WAIT uses
		// horizontal position zero, so all eight COLOR writes finish
		// before the first visible pixel. The ninth WAIT ends the list.
		if t.effects.paletteMode == paletteRainbow {
			for band < 8 && int(p.state.RainbowLines[band]>>8) <= y+52 {
				for index := 0; index < 8; index++ {
					palette[index] = p.artwork.Title.Rainbow[band][index]
				}
				band++
			}
		}
		for column := 0; column < 40; column++ {
			var planes [4]byte
			for plane := range planes {
				planes[plane] = bank.pixels[y*160+plane*40+column]
			}
			for bit := 0; bit < 8; bit++ {
				index := byte(0)
				for plane, data := range planes {
					index |= (data >> uint(7-bit) & 1) << plane
				}
				color := palette[index]
				at := (y*320 + column*8 + bit) * 4
				t.rgba[at], t.rgba[at+1], t.rgba[at+2], t.rgba[at+3] = byte(color>>8&15)*17, byte(color>>4&15)*17, byte(color&15)*17, 255
			}
		}
	}
	return t.rgba
}
