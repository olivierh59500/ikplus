package presentation

import (
	"github.com/olivierh59500/ikplus/internal/game"
	"github.com/olivierh59500/ikplus/internal/graphics"
)

// titleEffects advances palette fades, particles and authored title transitions.
// Actor picture preparation is a separate bounded Go operation.
type titleEffects struct {
	planes      *sceneryPlanes
	random      func() uint16
	hardware    [16]uint16
	paletteMode scenePaletteMode
}

func (t *titleEffects) packedWord(address int) uint16 { return t.planes.pictures[1].word(address) }
func (t *titleEffects) writePackedWord(address int, value uint16) {
	t.planes.pictures[1].putWord(address, value)
}
func (t *titleEffects) copy(to, from, bytes int) {
	copy(t.planes.pictures[1].pixels[to:to+bytes], t.planes.pictures[1].pixels[from:from+bytes])
}
func (t *titleEffects) advance() { t.planes.state.TitleAdvance = 1 }

func (t *titleEffects) letterMotion(index uint16) (graphics.TitleMotion, bool) {
	language := 0
	if t.planes.state.TitleLanguage != 0 {
		language = 1
	}
	motions := t.planes.artwork.Title.Motions[language]
	if int(index) >= len(motions) {
		return graphics.TitleMotion{}, false
	}
	return motions[index], true
}

// pictureFrames keeps fighter publication in step with the original title
// choreography. Broad letter shifts and screen scrolls use its slower cadence.
// These are animation delays, independent of native drawing performance.
func (t *titleEffects) pictureFrames() int {
	switch t.planes.state.TitleStage {
	case 17, 27, 30:
		return 6
	case 24:
		if motion, ok := t.letterMotion(t.planes.state.LetterOperation); ok {
			if motion.Span >= 5 || motion.Action&1 == 0 && motion.Span >= 4 {
				return 6
			}
		}
	}
	return 5
}

// pictureThreshold carries the small publication-phase offsets around broad
// leftward letter motion and the scrolling transition. The signed remainder
// keeps these adjustments local instead of adding a whole-frame delay.
func (t *titleEffects) pictureThreshold() int {
	stage := t.planes.state.TitleStage
	if stage == 27 {
		return titlePictureUnits - 1
	}
	if stage == 28 || stage == 29 {
		return titlePictureUnits - 4
	}
	if stage == 24 {
		if motion, ok := t.letterMotion(t.planes.state.LetterOperation); ok && motion.Action == 2 && motion.Span >= 7 {
			return titlePictureUnits + int(motion.Span) - 4
		}
	}
	return titlePictureUnits
}

// fade advances the current title palette toward an authored color target.
func (t *titleEffects) fade(colors [16]uint16, interval int) {
	p := game.TitlePalette{Divider: t.planes.state.FadeDivider}
	for i := range colors {
		p.Colors[i] = t.planes.state.TitlePalette[i]
	}
	if p.Fade(colors, uint16(interval)) {
		t.advance()
	}
	t.planes.state.FadeDivider = p.Divider
	for i, c := range p.Colors {
		t.planes.state.TitlePalette[i] = c
	}
}

func (t *titleEffects) rotate(frame byte) {
	if frame&3 != 0 {
		return
	}
	last := t.planes.state.TitlePalette[15]
	for i := 6; i > 0; i-- {
		t.planes.state.TitlePalette[9+i] = t.planes.state.TitlePalette[8+i]
	}
	t.planes.state.TitlePalette[9] = last
}

func (t *titleEffects) tick(frame byte) {
	t.planes.state.TitleTimer = t.planes.state.TitleTimer + 1
	for i := range t.hardware {
		t.hardware[i] = t.planes.state.TitlePalette[i]
		if t.planes.state.TitleStage == 0 {
			t.hardware[i] = t.planes.artwork.Title.Black[i]
		}
	}
	if t.planes.state.TitleAdvance != 0 && t.planes.state.TitleEnding == 0 {
		t.planes.state.TitleAdvance = 0
		t.planes.state.TitleStage = t.planes.state.TitleStage + 1
		t.planes.state.TitleTimer = 0
	}
	stage := t.planes.state.TitleStage
	if stage >= 32 {
		t.planes.state.TitleEnding = 1
		return
	}
	if t.planes.state.Frozen != 0 {
		return
	}
	switch stage {
	case 0:
	case 1:
		t.planes.state.RainbowY = 35
		t.rainbowPosition()
		t.planes.state.RainbowColor = 0
		t.advance()
	case 2:
		if t.planes.state.TitleTimer > 450 {
			t.advance()
			return
		}
		t.rainbowPosition()
		t.paletteMode = paletteRainbow
		t.copyPalette(0, t.planes.artwork.Title.ParticlePalette[:])
		t.copyPalette(8, t.planes.artwork.Title.ParticlePalette[:])
		t.planes.titleParticles(t.planes.state.TitleTimer)
		index := int(t.planes.state.RainbowColor >> 1)
		color := t.planes.artwork.Title.ParticleColors[index]
		if color != 0 {
			t.planes.state.RainbowColor = t.planes.state.RainbowColor + 1
			t.hardware[13] = color
			t.planes.state.TitlePalette[13] = color
		}
		if frame&1 != 0 && t.planes.state.RainbowY <= 160 {
			t.planes.state.RainbowY = t.planes.state.RainbowY + 1
			return
		}
		if t.planes.state.TitleTimer >= 300 {
			if t.planes.state.TitleTimer == 300 {
				t.reseedParticles()
			}
			source := t.planes.artwork.Title.ParticleTail
			if t.planes.state.TitleTimer >= 390 {
				source = t.planes.artwork.Title.ParticleExit
			}
			t.copyPalette(8, source[:])
			for i := 8; i < 16; i++ {
				t.hardware[i] = source[i-8]
			}
		}
	case 3:
		t.planes.titleParticles(t.planes.state.TitleTimer)
		if t.planes.state.TitleTimer >= 5 {
			t.advance()
		}
	case 4:
		t.fade(t.planes.artwork.Title.Fades[0], 4)
		t.planes.titleParticles(t.planes.state.TitleTimer)
	case 5:
		t.fade(t.planes.artwork.Title.Fades[1], 5)
		t.planes.titleParticles(t.planes.state.TitleTimer)
	case 6:
		clear(t.planes.pictures[1].pixels[:32000])
		t.advance()
	case 7:
		t.planes.titleLogo()
		t.advance()
	case 8:
		t.planes.titleLegend(false)
		t.advance()
	case 9:
		t.planes.titleLegend(true)
		t.advance()
	case 10:
		t.fade(t.planes.artwork.Title.Fades[2], 1)
	case 11:
		t.fade(t.planes.artwork.Title.Fades[3], 1)
	case 12:
		t.fade(t.planes.artwork.Title.Fades[4], 1)
	case 13:
		t.fade(t.planes.artwork.Title.Fades[5], 2)
		if t.planes.state.TitleAdvance != 0 {
			t.planes.state.TitleActorsEnabled = 1
		}
	case 14:
		if t.planes.state.TitleTimer >= 300 {
			t.advance()
		}
	case 15:
		t.fade(t.planes.artwork.Title.Fades[6], 3)
	case 16:
		t.fade(t.planes.artwork.Title.Fades[7], 3)
	case 17:
		t.instructions()
		t.copyPalette(0, t.planes.artwork.Title.InstructionPalette[:])
		t.instructionMasks()
		t.advance()
	case 18:
		if t.planes.state.TitleTimer >= 50 {
			t.advance()
		}
	case 19:
		t.fade(t.planes.artwork.Title.Fades[4], 3)
	case 20, 28:
		for row := 0; row < 120; row++ {
			for plane := 0; plane < 4; plane++ {
				clear(t.planes.pictures[1].pixels[(40*160+10)+row*160+plane*40 : (40*160+10)+row*160+plane*40+20])
			}
		}
		t.copyPalette(8, t.planes.artwork.Title.Black[:8])
		t.advance()
	case 21:
		if t.planes.state.TitleTimer >= 20 {
			t.advance()
		}
	case 22:
		t.rotate(frame)
		t.planes.state.LetterOperation, t.planes.state.LetterMotion, t.planes.state.LetterComplete = 0, 0, 0
		t.planes.state.LetterCount, t.planes.state.LetterIndex = 0, 0
		t.planes.state.LetterStep = t.planes.artwork.Title.LetterSteps[(byte(t.random())&6)/2]
		t.copyPalette(0, t.planes.artwork.Title.LetterPalette[:])
		t.advance()
	case 23:
		t.rotate(frame)
		t.revealLetters()
	case 24:
		t.rotate(frame)
		t.shuffleLetters()
	case 25:
		t.rotate(frame)
		if t.planes.state.TitleTimer >= 200 {
			t.advance()
		}
	case 26:
		t.rotate(frame)
		t.planes.state.ScrollCount = 80
		t.planes.state.ScrollDirection = 2
		t.advance()
	case 27:
		t.rotate(frame)
		t.horizontalScroll()
		if frame&1 != 0 {
			t.verticalScroll()
		}
		t.planes.state.ScrollCount = t.planes.state.ScrollCount - 1
		if t.planes.state.ScrollCount == 0 {
			t.planes.state.ScrollCount = 80
			t.planes.state.ScrollDirection = t.planes.state.ScrollDirection - 1
			if t.planes.state.ScrollDirection == 0 {
				t.advance()
			}
		}
	case 29:
		if t.planes.state.TitleTimer >= 50 {
			t.advance()
			t.planes.state.ColorSequence = 0
			t.planes.state.ColorRotations = 0
			t.planes.state.ColorRotationTimer = 0
		}
	case 30:
		t.planes.paintTitleRectangle((40*160 + 10), t.planes.artwork.Title.FinalLogo[:])
		t.advance()
	case 31:
		t.finalColors()
	}
}

func (t *titleEffects) rainbowPosition() {
	value := uint16(t.planes.state.RainbowY<<8) + 0x3401
	for i := 0; i < 9; i++ {
		value += 0x100
		t.planes.state.RainbowLines[i] = value
	}
}

func (t *titleEffects) reseedParticles() {
	value := byte(t.random())
	if int8(value) >= 0 {
		velocities := t.planes.artwork.Title.ParticleVelocity[(value&4)/4]
		for i := 0; i < 25; i++ {
			index := (byte(t.random()) & 12) / 4
			t.planes.state.Particles[i].VX = velocities[index][0]
			t.planes.state.Particles[i].VY = velocities[index][1]
		}
	} else {
		for i := 0; i < 25; i++ {
			particle := &t.planes.state.Particles[i]
			x := uint16(byte(t.random())&127) | 32
			if int16(particle.VX) < 0 {
				x = -x
			}
			particle.VX = x
			y := uint16(byte(t.random()) & 127)
			if int16(particle.VY) < 0 {
				y = -y
			}
			particle.VY = y
		}
	}
}

func (t *titleEffects) instructions() {
	t.planes.paintTitleRectangle((62*160 + 10), t.planes.artwork.Title.Instructions[:])
	for row := 0; row < 22; row++ {
		for plane := 0; plane < 4; plane++ {
			for _, base := range []int{(40*160 + 10), (141*160 + 10)} {
				clear(t.planes.pictures[1].pixels[base+row*160+plane*40 : base+row*160+plane*40+20])
			}
		}
	}
}

func (t *titleEffects) instructionMasks() {
	for row := 0; row < 81; row++ {
		for col := 0; col < 20; col += 4 {
			base := (62*160 + 10) + row*160 + col
			value := t.planes.pictures[1].long(base) | t.planes.pictures[1].long(base+40) | t.planes.pictures[1].long(base+80)
			t.planes.pictures[1].putLong(base+120, t.planes.pictures[1].long(base+120)|value)
		}
	}
}

func titleCell(index uint16) int { return int(index/10)*13*160 + int(index%10)*2 }

func (t *titleEffects) revealLetters() {
	count, index := t.planes.state.LetterCount, t.planes.state.LetterIndex
	language := 0
	if t.planes.state.TitleLanguage != 0 {
		language = 1
	}
	letter := t.planes.artwork.Title.Letters[t.planes.artwork.Title.LetterOrder[language][index]]
	to := (61*160 + 10) + titleCell(index)
	for row := 0; row < 13; row++ {
		for plane := 0; plane < 4; plane++ {
			t.writePackedWord(to+row*160+plane*40, letter[row][plane])
		}
	}
	index += t.planes.state.LetterStep
	if int16(index) >= 60 {
		index -= 60
	}
	count++
	if int16(count) >= 60 {
		t.advance()
	}
	t.planes.state.LetterCount = count
	t.planes.state.LetterIndex = index
}

func (t *titleEffects) shuffleLetters() {
	index := t.planes.state.LetterOperation
	motion, ok := t.letterMotion(index)
	if !ok {
		t.advance()
		return
	}
	column, row, span, operation := motion.Column, motion.Row, motion.Span, motion.Action
	base := (61*160 + 10) + int(column)*2 + int(row)*13*160
	switch operation {
	case 0:
		t.shiftLetter(base, int(span), false)
	case 1:
		t.moveLetterDown(base, int(span))
	case 2:
		t.shiftLetter(base, int(span), true)
	case 3:
		t.moveLetterUp(base, int(span))
	}
	if t.planes.state.LetterComplete != 0 {
		t.planes.state.LetterComplete = 0
		t.planes.state.LetterOperation = index + 1
		t.planes.state.LetterMotion = 0
	}
}

func (t *titleEffects) shiftLetter(base, span int, left bool) {
	for row := 0; row < 13; row++ {
		for plane := 0; plane < 4; plane++ {
			at := base + row*160 + plane*40
			carry := uint16(0)
			if left {
				for col := span - 1; col >= -1; col-- {
					word := t.packedWord(at + col*2)
					next := word >> 15
					t.writePackedWord(at+col*2, word<<1|carry)
					carry = next
				}
			} else {
				// The source processes span+1 words in its rightward chain.
				for col := 0; col <= span; col++ {
					word := t.packedWord(at + col*2)
					next := word & 1
					t.writePackedWord(at+col*2, word>>1|carry<<15)
					carry = next
				}
			}
		}
	}
	t.planes.state.LetterMotion = t.planes.state.LetterMotion + 1
	if t.planes.state.LetterMotion >= 16 {
		t.planes.state.LetterComplete = 1
	}
}

func (t *titleEffects) moveLetterUp(base, span int) {
	base -= 13 * 160
	rows := (span+1)*13 - 1
	for row := 0; row < rows; row++ {
		for plane := 0; plane < 4; plane++ {
			t.writePackedWord(base+row*160+plane*40, t.packedWord(base+(row+1)*160+plane*40))
		}
	}
	for plane := 0; plane < 4; plane++ {
		t.writePackedWord(base+rows*160+plane*40, 0)
	}
	t.planes.state.LetterMotion = t.planes.state.LetterMotion + 1
	if t.planes.state.LetterMotion >= 13 {
		t.planes.state.LetterComplete = 1
	}
}

func (t *titleEffects) moveLetterDown(base, span int) {
	var carry [4]uint16
	for row := 0; row < (span+1)*13; row++ {
		for plane := 0; plane < 4; plane++ {
			at := base + row*160 + plane*40
			word := t.packedWord(at)
			t.writePackedWord(at, carry[plane])
			carry[plane] = word
		}
	}
	t.planes.state.LetterMotion = t.planes.state.LetterMotion + 1
	if t.planes.state.LetterMotion >= 13 {
		t.planes.state.LetterComplete = 1
	}
}

func (t *titleEffects) horizontalScroll() {
	base := (61*160 + 50)
	leftStart, rightStart := 4, 5
	if t.planes.state.ScrollDirection == 1 {
		base = (61*160 + 130)
		leftStart, rightStart = 9, 0
	}
	for row := 0; row < 79; row++ {
		at := base + row*160
		carry := uint16(0)
		for i := 0; i < 5; i++ {
			column := leftStart - i
			word := t.packedWord(at + column*2)
			next := word >> 15
			t.writePackedWord(at+column*2, word<<1|carry)
			carry = next
		}
		carry = 0
		for i := 0; i < 5; i++ {
			column := rightStart + i
			word := t.packedWord(at + column*2)
			next := word & 1
			t.writePackedWord(at+column*2, word>>1|carry<<15)
			carry = next
		}
	}
}

func (t *titleEffects) verticalScroll() {
	up, down, clear1, clear2 := (61*160 + 10), (138*160 + 10), (100*160 + 10), (100*160 + 10)
	if t.planes.state.ScrollDirection == 1 {
		up, down, clear1, clear2 = (100*160 + 90), (99*160 + 90), (61*160 + 90), (100*160 + 90)
	}
	for row := 0; row < 39; row++ {
		t.copy(up, up+160, 20)
		t.copy(down+160, down, 20)
		up += 160
		down -= 160
	}
	clear(t.planes.pictures[1].pixels[clear1 : clear1+20])
	clear(t.planes.pictures[1].pixels[clear2 : clear2+20])
}

func (t *titleEffects) finalColors() {
	counter := (t.planes.state.ColorTimer & 15) - 1
	t.planes.state.ColorTimer = counter
	if counter != 0 {
		return
	}
	t.planes.state.ColorTimer = 3
	offset := t.planes.state.ColorSequence
	if int16(offset) >= 308 {
		t.advance()
		return
	}
	if t.planes.state.ColorRotations != 0 {
		first := t.planes.state.TitlePalette[8]
		for i := 0; i < 7; i++ {
			t.planes.state.TitlePalette[8+i] = t.planes.state.TitlePalette[9+i]
		}
		t.planes.state.TitlePalette[15] = first
		t.planes.state.ColorRotationTimer = t.planes.state.ColorRotationTimer - 1
		if t.planes.state.ColorRotationTimer == 0 {
			t.planes.state.ColorRotations = t.planes.state.ColorRotations - 1
			if t.planes.state.ColorRotations != 0 {
				t.planes.state.ColorRotationTimer = 8
			}
		}
	} else {
		t.planes.state.ColorSequence = offset + 2
		// $F2 is the signed eight-bit displacement in the indexed operand.
		entry := t.planes.artwork.Title.FinalColors[offset/2]
		t.copyPalette(8, entry.Colors[:])
		rotations := entry.Rotations
		if rotations != 0 {
			t.planes.state.ColorRotations = rotations
			t.planes.state.ColorRotationTimer = 8
		}
	}
}

// copyPalette loads authored colors without a mutable memory bridge.
func (t *titleEffects) copyPalette(first int, colors []uint16) {
	copy(t.planes.state.TitlePalette[first:], colors)
}
