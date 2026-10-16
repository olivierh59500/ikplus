package presentation

import (
	"encoding/binary"
	"math/bits"
)

// reducedTitlePicture retains the two authored preview sizes by animation ID.
type reducedTitlePicture struct {
	Large []byte
	Small []byte
}

// titleLogo follows $5974, including its inclusive 121-row source counter.
func (p *sceneryPlanes) titleLogo() {
	p.paintTitleRectangle(40*160+10, p.artwork.Title.Logo[:])
}

func (p *sceneryPlanes) paintTitleRectangle(target int, pixels [][4][10]uint16) {
	for row, planes := range pixels {
		for plane := 0; plane < 4; plane++ {
			for col, word := range planes[plane] {
				p.pictures[1].putWord(target+row*160+plane*40+col*2, word)
			}
		}
	}
}

// $2596 removes the logo's fourth plane on exactly 120 rows.
func (p *sceneryPlanes) titleRemoveFourthPlane() {
	for row := 0; row < 120; row++ {
		clear(p.pictures[1].pixels[(40*160+10)+row*160+120 : (40*160+10)+row*160+140])
	}
}

// $59BE/$59D2 merge the authored instruction legends through their color mask.
func (p *sceneryPlanes) titleLegend(second bool) {
	index, target := 0, (62*160 + 10)
	if second {
		index, target = 1, (102*160 + 10)
	}
	for row := 0; row < 41; row++ {
		for col := 0; col < 20; col += 4 {
			coverage := uint32(0)
			var ink [4]uint32
			for plane := 0; plane < 4; plane++ {
				words := p.artwork.Title.Legends[index][row][plane]
				ink[plane] = uint32(words[col/2])<<16 | uint32(words[col/2+1])
				coverage |= ink[plane]
			}
			for plane := 0; plane < 4; plane++ {
				at := target + row*160 + plane*40 + col
				p.pictures[1].putLong(at, p.pictures[1].long(at)&^coverage|ink[plane])
			}
		}
	}
}

// $294A packs the even source bits from two words into one reduced word.
func titleReduceWord(first, second uint16) uint16 {
	pack := func(word uint16) uint16 {
		value := uint16(0)
		for bit := uint(0); bit < 8; bit++ {
			value |= (word >> (bit * 2) & 1) << bit
		}
		return value
	}
	return pack(first)<<8 | pack(second)
}

// $28C8/$290C build normal and horizontally mirrored reduced picture fields.
func reduceTitleField(source, target, mirror []byte, small bool) {
	rows, columns, stride := 40, 10, 80
	if small {
		rows, columns, stride = 20, 5, 40
	}
	for row := 0; row < rows; row++ {
		for col := 0; col < columns; col++ {
			at := row*stride + col*4
			value := titleReduceWord(binary.BigEndian.Uint16(source[at:]), binary.BigEndian.Uint16(source[at+2:]))
			binary.BigEndian.PutUint16(target[row*columns*2+col*2:], value)
			binary.BigEndian.PutUint16(mirror[row*columns*2+(columns-col-1)*2:], bits.Reverse16(value))
		}
	}
}

// $29E0 copies one generated field into a scanline-strided destination.
func copyTitleField(source, target []byte, rows int) {
	width := 10
	if rows == 40 {
		width = 20
	}
	for row := 0; row < rows; row++ {
		copy(target[row*160:row*160+width], source[row*width:(row+1)*width])
	}
}

// The mirrored preview atlas occupies the unused top of the first picture.
// Its slices are bitmap storage; no executable pointer is resolved here.
func (p *sceneryPlanes) mirroredTitleField(index int, small bool) []byte {
	offset, size := index*800, 800
	if small {
		offset, size = 16*800+index*200, 200
	}
	return p.pictures[0].pixels[offset : offset+size]
}

// $308E animates three hundred points in the logo's fourth plane. At tick300
// only the first25 keep moving; at390 those points also leave their old ink.
func (p *sceneryPlanes) titleParticles(tick uint16) {
	for index := 0; index < 300; index++ {
		particle := &p.state.Particles[index]
		oldMask, oldOffset := particle.PreviousX, particle.PreviousY
		at := 120 + int(oldOffset)
		p.pictures[1].putWord(at, p.pictures[1].word(at)&^oldMask)
		if tick >= 300 && index >= 25 {
			continue
		}
		if tick >= 390 {
			p.pictures[1].putWord(at, p.pictures[1].word(at)|oldMask)
		}
		x, y := particle.X, particle.Y
		vx, vy := particle.VX, particle.VY
		move := func(position, velocity, low, high uint16) (uint16, uint16) {
			position += velocity
			for int16(position) < int16(low) || int16(position) >= int16(high) {
				velocity = -velocity
				position += velocity
			}
			return position, velocity
		}
		x, vx = move(x, vx, 0x1540, 0x3980)
		y, vy = move(y, vy, 0x0b40, 0x2580)
		particle.X = x
		particle.Y = y
		particle.VX = vx
		particle.VY = vy
		mask := uint16(0x8000 >> (x >> 6 & 15))
		offset := uint16((y>>6)*160) + (x>>6&0x1f0)>>3
		at = 120 + int(offset)
		p.pictures[1].putWord(at, p.pictures[1].word(at)|mask)
		particle.PreviousX = mask
		particle.PreviousY = offset
	}
}
