package presentation

import "math/bits"

// water translates $18DC2/$18F44/$18CE2, including the moving sparkle history
// omitted by the former independent reflection atlas. Only the central wave
// rows and edge bytes target $634; sparkle bytes update both screens.
func (p *sceneryPlanes) water(frame byte, phases [21]byte, bank int) {
	data := &p.artwork.Water
	if frame&1 == 0 {
		p.waterUpdates++
		if p.waterUpdates == 0 {
			p.waterSlow++
		}
		for row, phase := range phases {
			// $191A0 applies slow jitter after the masks/patterns are sampled.
			// The PAL core publishes the updated phases used by ball launching.
			if p.waterUpdates == 0 {
				phase = (phase - data.Jitter[((row&3)+int(p.waterSlow>>4))&15]) & 31
			}
			pattern := int(data.Patterns[int(data.Waveform[phase])+int(data.RowOffsets[row])])
			p.patterns[row] = pattern
			p.left[row], p.right[row] = data.LeftMasks[pattern], data.RightMasks[pattern]
		}
	}
	for _, edge := range data.Edges {
		offset, row := edge.Offset, edge.Phase
		mask := p.left[row%21]
		if row >= 21 {
			mask = p.right[row-21]
		}
		at := offset
		p.pictures[bank].pixels[at] &^= mask
		p.pictures[bank].pixels[at+40], p.pictures[bank].pixels[at+80] = mask, mask
		p.pictures[bank].pixels[at+120] &^= mask
	}
	values := [8]byte{
		p.left[19] ^ p.right[20],
		p.left[17] ^ p.right[18] ^ 0xc3,
		p.right[17] ^ p.right[19],
		p.left[6] ^ p.right[11],
		p.left[15] ^ p.left[16],
		p.left[16] ^ p.left[17],
		(p.left[16] ^ p.right[16]) & 0xdf,
		p.left[17] ^ p.right[19],
	}
	for index, value := range values {
		offset := int(int16(p.glitter[index]))
		for _, screen := range p.screens() {
			switch {
			case index < 4:
				screen.pixels[0x32ba+offset], screen.pixels[0x3ded-offset] = value, value
			case index < 6:
				screen.pixels[0x30d6+offset] = value
			default:
				screen.pixels[0x30c0+offset] = value
			}
		}
		if value != 0 {
			continue
		}
		table, choice := 0, byte(0)
		switch index {
		case 0, 1, 2, 3:
			choice = (frame + phases[index]) & 63
		case 4:
			table, choice = 1, ((bits.RotateLeft8(frame, 3)^phases[5])+phases[6])&31
		case 5:
			table, choice = 1, ((bits.RotateLeft8(frame, 6)^phases[3])+phases[5])&31
		case 6:
			table, choice = 2, (frame-p.waterUpdates+phases[4])&63
		case 7:
			table, choice = 2, (frame+p.waterUpdates+phases[2])&63
		}
		p.glitter[index] = data.Glitter[table][choice]
	}
	for row, pattern := range p.patterns {
		at := 78*160 + 16 + row*160
		for word := 0; word < 2; word++ {
			p.pictures[bank].putLong(at+word*4, 0)
			p.pictures[bank].putLong(at+40+word*4, 0xffffffff)
			p.pictures[bank].putLong(at+80+word*4, data.Ink[pattern][word])
			p.pictures[bank].putLong(at+120+word*4, data.Ink[pattern][word+2])
		}
	}
	for plane := 0; plane < 4; plane++ {
		at := 98*160 + 18 + plane*40
		p.pictures[bank].putLong(at, p.pictures[bank].long(at)&0xc00fffc3)
	}
}
