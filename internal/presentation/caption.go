package presentation

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math/bits"

	"github.com/olivierh59500/ikplus/assets"
	"github.com/olivierh59500/ikplus/internal/game"
)

type captionData struct {
	Messages                                     [2][40][]byte `json:"messages"`
	Colors                                       [40]byte      `json:"colors"`
	NameLengths                                  [3]byte       `json:"name_lengths"`
	NameOffsets                                  [3]byte       `json:"name_offsets"`
	Names                                        [2][]byte     `json:"names"`
	NameColors                                   [3]byte       `json:"name_colors"`
	Rows                                         [4]int        `json:"row_y"`
	palettes                                     [200][16]uint16
	font                                         image.Image
	tail, sparkles, sprites, shadows, background []byte
}

var originalCaptions = func() captionData {
	var data captionData
	read := func(name string) []byte {
		raw, err := assets.Read("original/" + name)
		if err != nil {
			panic(err)
		}
		return raw
	}
	if err := json.Unmarshal(read("captions.json"), &data); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(read("palettes.json"), &data.palettes); err != nil {
		panic(err)
	}
	decode := func(name string) image.Image {
		picture, err := png.Decode(bytes.NewReader(read(name)))
		if err != nil {
			panic(err)
		}
		return picture
	}
	data.font = decode("font.png")
	data.tail, data.sparkles = read("caption-tail.indices"), read("caption-sparkles.indices")
	data.sprites, data.shadows, data.background = read("sprites.indices"), read("fighter-shadows.bin"), read("background.indices")
	return data
}()

// paintOriginalCaption translates $8CA8/$8A2E/$8C2A and draws the original
// referee and sparkle cells. Its RGBA output is shared by runtime and tests.
func paintOriginalCaption(picture *image.RGBA, s game.CaptionState, background []byte) {
	paintOriginalCaptionIndices(picture, s, background, nil)
}

func paintOriginalCaptionIndices(picture *image.RGBA, s game.CaptionState, background, output []byte) {
	paintOriginalCaptionParts(picture, s, background, output, true)
}

func paintOriginalCaptionParts(picture *image.RGBA, s game.CaptionState, background, output []byte, text bool) {
	if !s.Visible {
		return
	}
	a := &originalCaptions
	message := s.Previous
	indices := output
	if indices == nil {
		indices = make([]byte, len(background))
	}
	copy(indices, background)
	pixel := func(x, y int, index byte) {
		if x < 0 || x >= 320 || y < 0 || y >= 200 {
			return
		}
		v := a.palettes[y][index&15]
		indices[y*320+x] = index
		picture.SetRGBA(x, y, color.RGBA{byte(v>>8&15) * 17, byte(v>>4&15) * 17, byte(v&15) * 17, 255})
	}
	column := func(x, y, height int, mask byte, ink byte) {
		for row := y; row < y+height; row++ {
			for bit := 0; bit < 8; bit++ {
				if mask&(128>>bit) != 0 {
					pixel(x+bit, row, ink)
				}
			}
		}
	}
	indexed := func(rawX byte, y, height, stride, sourceX int, pixels []byte) {
		// $1BB12 accepts raw columns 8..175 and clips the edge words.
		if rawX&248 < 8 || rawX&248 >= 176 {
			return
		}
		x := int(rawX)*2 - 32
		for row := 0; row < height; row++ {
			for col := 0; col < 16; col++ {
				index := pixels[row*stride+sourceX+col]
				if index != 0 {
					pixel(x+col, y+row, index)
				}
			}
		}
	}
	if message < 40 && text {
		left := int(s.Column) * 8
		column(left, 117, 1, 1, 4)
		column(left, 118, 19, 3, 4)
		column(left, 137, 1, 1, 4)
		for cell := 1; cell <= int(s.Width)-2; cell++ {
			column(left+cell*8, 117, 21, 255, 4)
		}
		right := left + (int(s.Width)-1)*8
		column(right, 117, 1, 128, 4)
		column(right, 118, 19, 192, 4)
		column(right, 137, 1, 128, 4)
		indexed(s.Anchor<<2+19, 138, 7, 16, 0, a.tail)
		language := 0
		if s.Alternate {
			language = 1
		}
		line, col, savedLine := byte(0), s.Column+1, byte(0)
		doubled := false
		ink := a.Colors[message]
		glyph := func(ch byte) {
			index := int(ch - 48)
			x, y := (index%16)*8, (index/16)*5
			for row := 0; row < 5; row++ {
				for bit := 0; bit < 8; bit++ {
					_, _, _, alpha := a.font.At(x+bit, y+row).RGBA()
					if alpha != 0 {
						top := a.Rows[line] + row
						if doubled {
							top += row
							pixel(int(col&63)*8+bit, top+1, ink)
						}
						pixel(int(col&63)*8+bit, top, ink)
					}
				}
			}
			col++
		}
		for offset, ch := range a.Messages[language][message] {
			switch {
			case ch < 10:
				for count := byte(0); count < ch; count++ {
					glyph(64)
				}
			case ch < 13:
				player := s.Order[ch-10]
				previousInk := ink
				ink = a.NameColors[player]
				start := a.NameOffsets[player]
				for count := byte(0); count < a.NameLengths[player]; count++ {
					glyph(a.Names[language][start+count])
				}
				ink = previousInk
			case ch >= 48 && ch < 95:
				glyph(ch)
			case ch == 0x9a:
				doubled = true
				if offset == 0 {
					savedLine, line = line, 3
				}
			case ch == 0x9b:
				goto textDone
			case ch == 0x9c:
				if line < 3 {
					col, line = s.Column+1, line+1
				}
			case ch == 0x9d:
				tens := s.Amount >> 4
				if tens == 0 {
					glyph(64)
				} else {
					glyph(48 + tens)
				}
				glyph(48 + s.Amount&15)
			case ch == 0x9e:
				doubled = false
				if offset == 0 {
					line = savedLine
				}
			}
		}
	}
textDone:
	frame := int(s.RefereeFrame)
	if frame >= 70 && frame < 88 {
		for y := 0; y < 80; y++ {
			for x := 0; x < 96; x++ {
				index := a.sprites[(frame*80+y)*96+x]
				if index != 0 {
					pixel(int(s.RefereeX)*2-32+x, 116+y, index*2)
				}
			}
		}
		// $AF7E ORs the authored eight-row shadow into packed bitplanes.
		// At the right edge this can reach the next plane rather than clip.
		for row := 0; s.Shadow && row < 8; row++ {
			for word := 0; word < 5; word++ {
				source := (frame*8+row)*10 + word*2
				mask := bits.RotateLeft32(uint32(binary.BigEndian.Uint16(a.shadows[source:])), -int(s.RefereeX&7)*2)
				address := 0x77fc + int(s.RefereeX&248)/4 + row*160 + word*2
				for half, value := range [2]uint16{uint16(mask), uint16(mask >> 16)} {
					for bit := 0; bit < 16; bit++ {
						if value&(0x8000>>bit) != 0 {
							at := address + half*2 + bit/8
							y, plane, x := at/160, at%160/40, at%40*8+bit%8
							pixel(x, y, indices[y*320+x]|(1<<plane))
						}
					}
				}
			}
		}
	}
	for slot := 7; slot >= 0; slot-- {
		if s.Draw[slot] {
			indexed(s.Anchor<<2+14+s.X[slot], int(s.Y[slot])+140, 5, 64, int(s.Counters[slot])*16, a.sparkles)
		}
	}
}
