package game

// originalBallState corresponds to one slot in the original projectile arrays:
// Y=0x6C5:0x6BB, VY=0x6D9:0x6CF, X=0x6ED:0x6E3, VX=0x6F7:0x701,
// Gravity=0x715, Floor=0x71F, Flags=0x70B, and Bounces=0x73D.
// The position bytes are unsigned 8.8; velocity uses signed 8.8 arithmetic.
type originalBallState struct {
	X, Y                           uint16
	VX, VY                         int16
	Gravity, Floor, Flags, Bounces byte
}

// originalBallStep translates 0xBCDC..0xBEB0, including integer-only floor
// reflection, carry propagation, the flashing ball's alternating elasticity,
// and the original edge removal. It returns whether the floor was contacted.
func originalBallStep(s *originalBallState) bool {
	s.VY += int16(s.Gravity)
	s.Y += uint16(s.VY)
	bounced := byte(s.Y>>8) >= s.Floor
	if bounced {
		// The source reflects the integer coordinate without reflecting its
		// fractional byte. Whole-position reflection produces a different arc.
		integer := 2*s.Floor - byte(s.Y>>8)
		s.Y = uint16(integer)<<8 | s.Y&255
		s.VY = -s.VY
		if s.VY != 0 {
			s.Bounces++
		}
		if s.Flags&8 != 0 {
			s.VY >>= 1
		}
		if s.Flags&32 != 0 {
			if s.Flags&16 != 0 {
				s.VY >>= 1
				s.Flags &^= 16
			} else {
				s.VY *= 2
				s.Flags |= 48
			}
		} else if s.Flags&16 != 0 {
			s.VY >>= 1
			s.Flags &^= 16
		}
		if s.Flags&4 != 0 {
			s.VX >>= 1
		}
	} else if byte(s.Y>>8) >= 104 {
		s.Flags = 0
		s.X &= 255
		return false
	}
	s.X += uint16(s.VX)
	if integer := byte(s.X >> 8); integer >= 183 || integer < 3 {
		s.Flags = 0
		s.X &= 255
	}
	return bounced
}

// originalBallPosition converts the original renderer's doubled horizontal
// position and vertical offset into the port's visible projectile centre.
func originalBallPosition(s originalBallState) (float64, float64) {
	// The original blitter retains one horizontal fractional bit and discards
	// the vertical fractional byte. The recovered ball cell is 16 by 12 pixels.
	return float64(s.X>>7) - 24, float64(s.Y>>8) + 102
}
