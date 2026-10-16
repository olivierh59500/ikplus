package presentation

import (
	"encoding/binary"
	"math/bits"

	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/graphics"
)

// sceneryPlanes translates the original packed drawing operations. Both
// screens retain their history; restoring a rectangle copies original planes,
// rather than removing an independent RGBA sprite.
type sceneryPlanes struct {
	stableDisplay              bool
	publishedScoreFrame        []byte
	state                      game.PresentationState
	artwork                    *graphics.Data
	titleHistory               [16]reducedTitlePicture
	hintStencil                []byte
	pictures                   [2]packedBitmap
	working                    packedBitmap
	titleMask                  packedBitmap
	titleAtlas                 []byte
	bird, fish, ripple, script savedDrawing
	stamps                     [16]savedDrawing
	stampCursor                int
	left, right                [21]byte
	patterns                   [21]int
	glitter                    [8]uint16
	waterUpdates, waterSlow    byte
	presentation               *presentationIO
}

func newScenePlanes() *sceneryPlanes {
	p := &sceneryPlanes{state: game.NewPresentationState(), artwork: graphics.MustLoad()}
	p.working.allocate(32768)
	p.titleMask.allocate(8192)
	p.titleAtlas = make([]byte, 16004)
	for index := range p.pictures {
		size := 32768
		if index == 0 {
			size += 16
		}
		p.pictures[index].allocate(size)
		copy(p.pictures[index].pixels, p.artwork.Background[:32000])
	}
	for index := range p.titleHistory {
		p.titleHistory[index].Large = p.titleAtlas[index*800 : (index+1)*800]
		p.titleHistory[index].Small = p.titleAtlas[12800+index*200 : 12800+(index+1)*200]
	}
	p.glitter[6], p.glitter[7] = 32, 32
	return p
}

// $19B92 uses an inclusive row counter and restores four bytes in each plane.
func (p *sceneryPlanes) restore(saved savedDrawing, rows int) {
	if saved.Picture == nil {
		return
	}
	background := saved.Offset & 0x7ffe
	for row := 0; row < rows; row++ {
		for plane := 0; plane < 4; plane++ {
			saved.Picture.putLong(saved.Offset+row*160+plane*40, binary.BigEndian.Uint32(p.artwork.Background[background+row*160+plane*40:]))
		}
	}
}

func (p *sceneryPlanes) restoreBoth(saved savedDrawing, rows int) {
	if saved.Picture == nil {
		return
	}
	for _, bank := range p.screens() {
		p.restore(savedDrawing{bank, saved.Offset & 0x7ffe}, rows)
	}
}

// $1BAFE accepts pixel X; $1BB12 accepts the original two-pixel X units.
// Their shifts, edge masks and packed row spill are preserved exactly.
func (p *sceneryPlanes) sprite(bank, x, y, height int, source graphics.Sprite, foreground, pixelX bool) savedDrawing {
	return p.spriteAt(&p.pictures[bank], x, y, height, source, foreground, pixelX, false)
}

func (p *sceneryPlanes) spriteAt(base *packedBitmap, x, y, height int, source graphics.Sprite, foreground, pixelX, black bool) savedDrawing {
	return p.paintSpriteAt(base, x, y, height, source, foreground, pixelX, black, false)
}

func (p *sceneryPlanes) paintSpriteAt(base *packedBitmap, x, y, height int, source graphics.Sprite, foreground, pixelX, black, restore bool) savedDrawing {
	shift := (x & 7) * 2
	if pixelX {
		x &= 511
		shift = x & 15
		x >>= 1
	} else {
		x &= 255
	}
	y &= 4095
	if y >= 200 {
		return savedDrawing{}
	}
	if y+height >= 201 {
		height = 200 - y
	}
	aligned := x & 248
	column := aligned >> 2
	offset := ((y*160 + column) & 0x7ffe) - 4
	address := offset
	edgeMask := p.artwork.EdgeMasks[0]
	if !foreground && aligned < 16 {
		edgeMask = p.artwork.EdgeMasks[1]
		if aligned < 8 {
			return savedDrawing{}
		}
	} else if !foreground && aligned >= 168 {
		edgeMask = p.artwork.EdgeMasks[2]
		if aligned >= 176 {
			return savedDrawing{}
		}
	}
	for row := 0; row < height; row++ {
		mask := edgeMask
		if foreground {
			mask = binary.BigEndian.Uint32(p.artwork.Foreground[(y+row)*40+column:])
		}
		var planes [4]uint32
		opacity := uint32(0)
		if black {
			opacity = bits.RotateLeft32(uint32(source.Rows[row][0])<<16, -shift) & mask
		} else {
			for plane := range planes {
				planes[plane] = bits.RotateLeft32(uint32(source.Rows[row][plane])<<16, -shift) & mask
				opacity |= planes[plane]
			}
		}
		for plane, ink := range planes {
			at := address + row*160 + plane*40
			if restore {
				ink = binary.BigEndian.Uint32(p.artwork.Background[at&0x7ffe:]) & opacity
			}
			if restore && black {
				// $1C076 restores a black collision mask with OR. Artwork
				// painted over that mask survives, including its set bits.
				base.putLong(at, base.long(at)|ink)
				continue
			}
			base.putLong(at, base.long(at)&^opacity|ink)
		}
	}
	return savedDrawing{base, address}
}

func (p *sceneryPlanes) drawBird(a game.AmbientState) {
	p.restoreBoth(p.bird, 5)
	x, y := int(a.X), int(a.Y)
	p.bird = savedDrawing{&p.pictures[0], y*160 + (x&248)/4 - 4}
	if x < 8 || x >= 176 {
		return
	}
	mask := y*40 + (x&248)/4
	shape := p.artwork.Birds[int(a.Style)+(x&3)]
	for row := 0; row < 5; row++ {
		rotated := bits.RotateLeft32(uint32(shape[row]), -(x&7)*2)
		first := uint16(rotated) & binary.BigEndian.Uint16(p.artwork.Foreground[mask+row*40:])
		second := uint16(rotated>>16) & binary.BigEndian.Uint16(p.artwork.Foreground[mask+row*40+2:])
		if x < 16 {
			first = 0
		}
		if x >= 168 {
			second = 0
		}
		for column, coverage := range [2]uint16{first, second} {
			at := p.bird.Offset + row*160 + column*2
			ink := uint16(0)
			if a.Color != 0 {
				ink = coverage
			}
			firstPlane := p.bird.Picture.word(at)&^coverage | ink
			for _, bank := range p.screens() {
				destination := at & 0x7ffe
				bank.putWord(destination, firstPlane)
				for plane := 1; plane < 4; plane++ {
					bank.putWord(destination+plane*40, bank.word(destination+plane*40)&^coverage)
				}
			}
		}
	}
}

// $196A2 stamps seven rows into both screens, retaining earlier portions of
// the growing outline outside the current seven-row window.
func (p *sceneryPlanes) drawOutline(a game.AmbientState) {
	frame := int(a.Frame)
	shape := p.artwork.Outlines[frame]
	offset := int(a.Y)*160 + int(a.X&248)/4 + shape.Offset - 2
	for row := 0; row < 7; row++ {
		// The indexed $C0 displacement is signed: $1985A - $40.
		coverage := uint16(bits.RotateLeft32(uint32(shape.Rows[row]), -int(a.X&7)*2))
		for plane := 0; plane < 4; plane++ {
			at := offset + row*160 + plane*40
			value := binary.BigEndian.Uint16(p.artwork.Background[at:])
			value &^= coverage
			if plane == 0 && a.Color != 0 {
				value |= coverage
			}
			for _, bank := range p.screens() {
				bank.putWord(at, value)
			}
		}
	}
}

// tick follows $18C42: restore transient artwork, update water, then draw the
// selected scenery handler. The state has already advanced in the PAL core.
func (p *sceneryPlanes) tick(before, after game.AmbientState, frame byte, phases [21]byte, bank int, water bool) {
	switch before.Mode {
	case 3:
		p.restore(p.ripple, 5)
		p.ripple = savedDrawing{}
	case 4:
		p.restore(p.fish, 15)
		p.fish = savedDrawing{}
	case 7:
		p.restore(p.script, 12)
		p.script = savedDrawing{}
	}
	if water {
		p.water(frame, phases, bank)
	}
	switch before.Mode {
	case 1:
		if after.Counter&3 == 0 {
			if after.Mode == 1 {
				p.drawBird(after)
			} else {
				p.restoreBoth(p.bird, 5)
			}
		}
	case 2:
		if after.Counter&3 == 0 && after.Mode == 2 {
			p.drawOutline(after)
		}
	case 3:
		if after.Mode == 3 {
			p.ripple = p.sprite(bank, int(after.X), int(after.Y), 4, p.artwork.Sprites[p.artwork.Ripples[after.Frame]], true, false)
		}
	case 4:
		if after.Mode == 4 && after.Visible {
			p.fish = p.sprite(bank, int(after.X), int(after.Y), 14, p.artwork.Sprites[p.artwork.Fish[after.Direction][after.Frame]], true, false)
		}
	case 5:
		if before.ScriptInitial {
			p.script = savedDrawing{}
		}
		p.restoreBoth(p.script, 3)
		p.script = savedDrawing{}
		if after.Mode != 0 {
			p.script = p.sprite(bank, int(after.ScriptX), int(after.ScriptY), 2, p.artwork.Sprites[p.artwork.ScriptStamp], false, true)
		} else {
			var address savedDrawing
			for target := range p.pictures {
				address = p.sprite(target, int(after.ScriptX), int(after.ScriptY), 2, p.artwork.Sprites[p.artwork.ScriptStamp], false, true)
			}
			p.restoreBoth(p.stamps[p.stampCursor], 3)
			p.stamps[p.stampCursor] = address
			p.stampCursor = (p.stampCursor + 1) & 15
		}
	case 6:
		if before.ScriptInitial {
			p.script = savedDrawing{}
		}
		p.restore(p.script, 9)
		p.script = savedDrawing{}
		if after.Mode == 6 {
			direction := 0
			if p.artwork.PacFacing[after.PacCycle&3] {
				direction = 1
			}
			foreground := after.ScriptX >= 34 && after.ScriptX < 340
			p.script = p.sprite(bank, int(after.ScriptX), int(after.ScriptY), 8, p.artwork.Sprites[p.artwork.Pac[direction][after.ScriptClock]], foreground, true)
		}
	case 7:
		if after.Mode == 7 {
			pose, height := int(after.PeriPose), 9
			if pose == 254 {
				pose, height = 0, 1
			}
			p.script = p.sprite(bank, int(after.ScriptX), int(after.ScriptY), height, p.artwork.Sprites[p.artwork.Peri[pose]], true, false)
			p.sprite(bank, int(after.ScriptX), int(after.ScriptY)+7, 4, p.artwork.Sprites[p.artwork.PeriFeet[after.PeriFrame]], true, false)
		}
	}
}

func (p *sceneryPlanes) rgba(bank int, palettes [][]uint16, out []byte) {
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			index := byte(0)
			for plane := 0; plane < 4; plane++ {
				index |= (p.pictures[bank].pixels[y*160+plane*40+x/8] >> uint(7-x%8) & 1) << plane
			}
			color := palettes[y][index]
			at := (y*320 + x) * 4
			out[at], out[at+1], out[at+2], out[at+3] = byte(color>>8&15)*17, byte(color>>4&15)*17, byte(color&15)*17, 255
		}
	}
}
