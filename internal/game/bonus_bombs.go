package game

// bombGoal is the immediate operand of original $C99E, not the floor value
// $56 used by spawning. The supplied Amiga executable launches 32 bombs.
var bombGoal = nativeGameplay.Bonus.BombGoal

func (g *Game) tickOriginalBombObjects() {
	b := &g.Bonus
	for len(b.Bombs) < 10 {
		b.Bombs = append(b.Bombs, Bomb{})
	}
	b.activeCount = 0
	f := g.Fighters[b.Fighter]
	mask := f
	if b.maskReady {
		mask = b.contactFighter
	}
	for id := range b.Bombs {
		p := &b.Bombs[id]
		if p.source.Flags == 0 {
			continue
		}
		b.activeCount++
		handler := p.Stage
		if p.drawn && (handler >= 1 && handler <= 4 || handler == 6) {
			bank := p.pictureBank
			erase := BombPictureOperation{
				Frame: g.Frame, Slot: id, Bank: bank, Bomb: *p,
				PaintedAfterActors: p.pictureFrame > g.pictures[bank].Frame,
			}
			g.recordBombPicture(erase)
		}
		switch handler {
		case 1, 2, 3, 4:
			if p.drawn {
				x, y := originalBallPosition(p.source)
				_, weapon := bonusMaskContact(mask, false, bonusPixelMasks.bomb, int(x)-8, int(y)-6)
				if weapon && p.source.Flags&2 == 0 {
					p.source.Flags, p.source.Gravity = 19, 48
					p.source.VY, p.source.VX = -3<<8, int16(f.Facing)<<8
					b.Deflected++
					g.Fighters[b.Fighter].addOriginalScore(100)
					g.emit(DeflectEvent, b.Fighter, -1, x, y, 100)
				}
			}
			p.drawn = false
			if g.advanceSourceBombFuse(p, handler) {
				g.explodeSourceBomb(id)
				continue
			}
			bounces := p.source.Bounces
			originalBallStep(&p.source)
			g.soundFloorBounce(bounces, p.source)
			p.drawn = p.source.Flags != 0
			p.pictureBank = g.irqPictureBank
			p.pictureFrame = g.Frame
			if p.source.Flags == 0 {
				p.Stage = 0
			} else {
				g.recordBombGround(id, p)
			}
		case 6:
			if p.drawn && !b.failed {
				x, y := originalBallPosition(p.source)
				body, _ := bonusMaskContact(mask, false, bonusPixelMasks.fragments[p.Variant&3][p.Frame&7], int(x)-8, int(y)-6)
				if body {
					if g.audioControl != nil {
						// $CEBE schedules the same impact again after three VBLs.
						g.audioControl.State.Delay, g.audioControl.State.Delayed = 3, 17
					}
					g.requestSound(17, true)
					g.soundCooldown[1] = 50
					facing := byte(0)
					if f.Facing < 0 {
						facing = 1
					}
					direction := (byte(uint16(p.source.VX)>>8) >> 7) ^ facing
					b.failed, b.failureAction = true, int(nativeGameplay.Bonus.FailureActions[direction&1])
					p.source.VY, p.source.VX = -p.source.VY, -p.source.VX
					g.emit(HitEvent, -1, b.Fighter, x, y, 0)
				}
			}
			p.drawn = false
			bounces := p.source.Bounces
			originalBallStep(&p.source)
			g.soundFloorBounce(bounces, p.source)
			if p.source.Flags != 0 {
				p.Frame = (p.Frame + int(int8(nativeGameplay.Bonus.FragmentMotion[id&3][g.OriginalVBL()&3]))) & 7
				p.drawn = true
				p.pictureBank = g.irqPictureBank
				p.pictureFrame = g.Frame
			} else {
				p.Stage = 0
			}
		}
	}
	if !b.failed {
		g.spawnOriginalBomb()
	}
	g.syncOriginalBombs()
	if g.paletteWaitFrame == g.Frame {
		g.updateBeltPalette()
	}
}

func (g *Game) recordBombPicture(operation BombPictureOperation) {
	bank := operation.Bank
	g.bombOperations[bank] = append(g.bombOperations[bank], operation)
	g.bombIRQOperations = append(g.bombIRQOperations, operation)
}

// $C188 restores its own saved floor rectangle only when $D0B8 is called.
// Explosion and removal branches leave that ink until a later preparation.
func (g *Game) recordBombGround(slot int, bomb *Bomb) {
	old := bomb.groundPicture
	if old.valid {
		g.recordBombPicture(BombPictureOperation{Frame: g.Frame, Slot: slot, Bank: old.bank,
			Kind: BombGroundRestore, Bomb: Bomb{X: old.x, Y: old.y}})
	}
	bomb.groundPicture.valid = false
	x, y := originalBallPosition(bomb.source)
	rawX, rawY := int(bomb.source.X>>8), int(bomb.source.Y>>8)
	if rawX < 8 || rawX >= 176 || rawY < 79 || rawY > 86 {
		return
	}
	g.recordBombPicture(BombPictureOperation{Frame: g.Frame, Slot: slot, Bank: g.irqPictureBank,
		Kind: BombGroundDraw, Bomb: Bomb{X: x, Y: y}})
	// $C27E/$C282 tests only the two horizontal-velocity bytes at $6F7/$701.
	// Vertical motion alone does not save a floor rectangle for restoration.
	if bomb.source.VX != 0 {
		bomb.groundPicture = bombGroundPicture{true, g.irqPictureBank, x, y}
	}
}

// $D15A changes motion and flags, retaining the selected fuse phase, counter
// and remaining burn. Its signed indexed displacements address velocity bytes.
func (g *Game) advanceSourceBombFuse(p *Bomb, handler int) bool {
	switch handler {
	case 1:
		p.burnTicks = int(int8(byte(p.burnTicks - 1)))
		if p.burnTicks <= 0 {
			p.Stage = 2
			p.burnTicks = g.Bonus.bombFuse
		}
	case 2:
		if g.OriginalVBL()&3 != 0 {
			return false
		}
		p.StageTicks++
		if p.StageTicks >= 26 {
			if p.burnTicks == 0 {
				return true
			}
			p.Stage = 3
		}
	case 3:
		p.burnTicks = int(int8(byte(p.burnTicks - 1)))
		if p.burnTicks <= 0 {
			p.Stage = 4
		}
	case 4:
		if g.OriginalVBL()&3 != 0 {
			return false
		}
		p.StageTicks++
		return p.StageTicks > 35
	}
	return false
}

func (g *Game) explodeSourceBomb(id int) {
	b := &g.Bonus
	p := &b.Bombs[id]
	start, other := 2, 1
	if id == 1 {
		start, other = 6, 0
	}
	count := 4
	if b.Bombs[other].source.Flags != 0 {
		count = 3
	}
	for index := 0; index < count; index++ {
		child := &b.Bombs[start+index]
		x := byte(p.source.X >> 8)
		p.source.X &= 0xf8ff
		child.source.X = uint16(x+nativeGameplay.Bonus.FragmentX[index])<<8 | child.source.X&255
		child.source.Y = uint16(byte(p.source.Y>>8)+nativeGameplay.Bonus.FragmentY[index])<<8 | child.source.Y&255
		child.source.Flags, child.source.Gravity, child.source.Floor = 1, 24, 116
		child.Stage, child.Variant, child.Frame = 6, index, 7
		delta := nativeGameplay.Bonus.FragmentVX[(g.originalRandomByte()&30)/2]
		hi := byte(uint16(p.source.VX)>>8) + byte(delta>>8)
		lo := byte(p.source.VX) + byte(delta)
		child.source.VX = int16(uint16(hi)<<8 | uint16(lo))
		child.source.VY = int16(nativeGameplay.Bonus.FragmentVY[(g.originalRandomByte()&30)/2])
	}
	p.Stage = 5
}

func (g *Game) tickOriginalBombBlasts() {
	b := &g.Bonus
	for id := 0; id < 2 && id < len(b.Bombs); id++ {
		p := &b.Bombs[id]
		if p.source.Flags == 0 || p.Stage != 5 {
			continue
		}
		position := byte(p.source.X>>8) - 8
		if position < 16 || position >= 152 {
			p.source.Flags, p.Stage, p.blastCounter, p.BlastFrame = 0, 0, 0, 0
			continue
		}
		if p.blastCounter == 0 {
			g.requestSound(16, false)
			x, y := originalBallPosition(p.source)
			g.emit(ExplosionEvent, g.Bonus.Fighter, -1, x, y, 0)
		}
		index := p.blastCounter & 15
		p.blastCounter++
		frame := int(int8(nativeGameplay.Bonus.BlastFrames[index]))
		if frame < 0 {
			p.source.Flags, p.Stage, p.blastCounter, p.BlastFrame = 0, 0, 0, 0
		} else {
			p.BlastFrame = frame
		}
	}
	g.syncOriginalBombs()
}

func (g *Game) spawnOriginalBomb() {
	b := &g.Bonus
	if b.Number >= bombGoal {
		b.spawnDone = true
		return
	}
	// $C9B2 keeps the launch countdown unchanged until the introductory
	// speech has actually been cleared, including its last prepared picture.
	if g.caption.state.Visible {
		return
	}
	b.spawnWait = int(byte(b.spawnWait - 1))
	if b.spawnWait != 0 {
		return
	}
	b.bombDelay = max(8, b.bombDelay-8)
	b.spawnWait = b.bombDelay
	id, start := -1, 0
	for candidate := 0; candidate < 2; candidate++ {
		children := 2 + candidate*4
		free := b.Bombs[candidate].source.Flags == 0
		for slot := children; slot < children+4; slot++ {
			free = free && b.Bombs[slot].source.Flags == 0
		}
		if free {
			id, start = candidate, children
			break
		}
	}
	if id < 0 {
		return
	}
	p := &b.Bombs[id]
	p.source = originalBallState{Y: 86 << 8, Flags: 1, Floor: 86}
	p.Stage, p.StageTicks, p.Frame, p.Color, p.burnTicks, p.blastCounter = 1, 0, 0, 0, b.bombFuse, 0
	p.BlastFrame, p.drawn = 0, false
	position := 0
	if b.Number < 6 {
		position = int(nativeGameplay.Bonus.BombPositions[b.Number])
	} else {
		position = 3 + int(g.originalRandomByte()&15)
	}
	p.source.X = uint16(position*8+16) << 8
	b.Number++
	for slot := start; slot < start+4; slot++ {
		b.Bombs[slot].source.Flags = 0
		b.Bombs[slot].drawn = false
	}
	if b.bombFuse >= 4 {
		b.bombFuse -= 2
	}
	if b.Number >= 27 {
		b.bombFuse = 0
	}
}

func (g *Game) syncOriginalBombs() {
	for id := range g.Bonus.Bombs {
		p := &g.Bonus.Bombs[id]
		p.X, p.Y = originalBallPosition(p.source)
		p.VX, p.VY = float64(p.source.VX)/128, float64(p.source.VY)/256
		p.Active, p.Kicked, p.Fragment = p.source.Flags != 0, p.source.Flags&2 != 0, id >= 2
		p.Explosion = 0
		if p.Stage == 5 && p.Active {
			p.Explosion = 10 - p.blastCounter
		}
		switch p.Stage {
		case 1:
			p.Fuse = max(0, p.burnTicks) + 104 + g.Bonus.bombFuse + 40
		case 2:
			p.Fuse = (26-p.StageTicks)*4 + max(0, p.burnTicks) + 40
		case 3:
			p.Fuse = max(0, p.burnTicks) + (36-p.StageTicks)*4
		case 4:
			p.Fuse = (36 - p.StageTicks) * 4
		default:
			p.Fuse = 0
		}
	}
}
