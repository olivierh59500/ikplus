package game

import (
	"math"

	"github.com/olivierh59500/ikplus/internal/graphics"
)

func advanceProjectileSpin(frame, color int, velocity int16, flags, vbl byte) (int, int) {
	// Retained animation bytes wrap before the directional range test.
	value := byte(frame)
	if velocity < 0 {
		value--
		if int8(value) < 0 {
			value = 8
		}
	} else {
		value++
		if int8(value) >= 9 {
			value = 0
		}
	}
	if flags&32 != 0 && vbl&7 == 0 {
		color = int(nativeGameplay.Bonus.ReboundColors[color&7])
	}
	return int(value), color
}

func (g *Game) tickOriginalBallObjects(mask Fighter) {
	b := &g.Bonus
	if g.ballPictureValid {
		for _, ball := range b.Balls {
			if ball.drawn {
				bank := g.ballPictureBank
				g.ballRestores[bank] = append(g.ballRestores[bank], BallPictureOperation{Frame: g.Frame, Ball: ball, Kind: BallPictureRestore})
				g.rememberBallActorErase(ball, bank)
			}
		}
	}
	g.ballPictureBank, g.ballPictureValid = g.irqPictureBank, true
	for len(b.Balls) < 4 {
		b.Balls = append(b.Balls, Ball{})
	}
	var body, weapon [4]bool
	// C32C samples all previous ball pictures before changing any slot.
	for id := 0; id < 4; id++ {
		p := &b.Balls[id]
		if !p.drawn {
			continue
		}
		x, y := originalBallPosition(p.source)
		if !b.failed && (int(x)+24 < 119 || int(x)+24 >= 270) {
			continue
		}
		body[id], weapon[id] = bonusMaskContact(mask, true, projectilePixelMask(p.Color, p.Frame), int(x)-8, int(y)-6)
	}
	// Retain contact geometry for the native preparation budget as well as
	// optional diagnostics. Gameplay resolution still consumes local values.
	b.preparation.bodyContacts, b.preparation.shieldContacts = body, weapon
	g.advanceBallMotion(body, weapon)
	// C2C6 saves the framebuffer pointer for every slot, including inactive
	// offscreen entries. Launching into a reused slot does not clear it.
	for id := range b.Balls {
		b.Balls[id].drawn = true
	}
	if b.live || b.introRemaining == 0 {
		g.advanceBallLaunch()
	}
	g.syncOriginalBalls()
	for _, ball := range b.Balls {
		if _, visible := graphics.ProjectileGroundShadow(ball.X, ball.Y, 187); visible {
			bank := g.ballPictureBank
			g.ballRestores[bank] = append(g.ballRestores[bank], BallPictureOperation{Frame: g.Frame, Ball: ball, Kind: BallGroundDraw})
		}
	}
	if g.paletteWaitFrame == g.Frame {
		// IRQ contact awards precede the main-thread wait's $A938 update.
		g.updateBeltPalette()
	}
}

// advanceBallMotion consumes all previously sampled contacts before changing
// any projectile. The shield moves first, followed by balls in reverse order.
func (g *Game) advanceBallMotion(body, weapon [4]bool) {
	b := &g.Bonus
	if b.shieldMotion.Flags != 0 {
		before := b.shieldMotion
		cooldown := g.soundCooldown
		bounces := b.shieldMotion.Bounces
		originalBallStep(&b.shieldMotion)
		if g.pacedBallPreparation() {
			b.preparation.motionInstructionCycles += ballMotionInstructionCycles(before, cooldown)
			b.preparation.spinInstructionCycles += ballSpinInstructionCycles(b.shieldFrame, b.shieldMotion.VX, b.shieldMotion.Flags, g.OriginalVBL())
		}
		g.soundFloorBounce(bounces, b.shieldMotion)
		b.shieldFrame, b.shieldColor = advanceProjectileSpin(b.shieldFrame, b.shieldColor, b.shieldMotion.VX, b.shieldMotion.Flags, g.OriginalVBL())
	}
	b.activeCount = 0
	for id := 3; id >= 0; id-- {
		p := &b.Balls[id]
		if p.source.Flags == 0 {
			continue
		}
		b.activeCount++
		if p.source.Flags&2 == 0 {
			x, y := originalBallPosition(p.source)
			if weapon[id] {
				id := uint16(15)
				if p.Color == 7 {
					id = 11
				}
				g.requestSound(id, true)
				g.soundCooldown[0] = 30
				p.source.Flags, p.source.VX = 19, -p.source.VX
				b.Deflected++
				g.Fighters[b.Fighter].addOriginalScore(100)
				g.emit(DeflectEvent, b.Fighter, -1, x, y, 100)
			} else if body[id] {
				if !b.failed {
					g.requestSound(17, true)
					g.soundCooldown = [2]byte{50, 50}
					b.failed = true
					facing := byte(0)
					if g.Fighters[b.Fighter].Facing < 0 {
						facing = 1
					}
					direction := (byte(uint16(p.source.VX)>>8) >> 7) ^ facing
					b.failureAction = int(nativeGameplay.Bonus.FailureActions[direction&1])
					b.shieldMotion.Gravity, b.shieldMotion.Floor = 36, 127
					b.shieldMotion.VY, b.shieldMotion.VX = -64, 64
					if g.Fighters[b.Fighter].Facing < 0 {
						b.shieldMotion.VX = -64
					}
					p.source.VY = 0
					g.emit(HitEvent, -1, b.Fighter, x, y, 0)
				} else {
					g.requestSound(3, true)
					p.source.VY = -p.source.VY
				}
				p.source.Flags = 19
			}
		}
		before := p.source
		cooldown := g.soundCooldown
		bounces := p.source.Bounces
		originalBallStep(&p.source)
		if g.pacedBallPreparation() {
			b.preparation.motionInstructionCycles += ballMotionInstructionCycles(before, cooldown)
			b.preparation.spinInstructionCycles += ballSpinInstructionCycles(p.Frame, p.source.VX, p.source.Flags, g.OriginalVBL())
		}
		g.soundFloorBounce(bounces, p.source)
		p.Frame, p.Color = advanceProjectileSpin(p.Frame, p.Color, p.source.VX, p.source.Flags, g.OriginalVBL())
	}
}

// advanceBallLaunch runs after painting the existing projectiles. Newly
// allocated motion therefore cannot change that update's drawing inputs.
func (g *Game) advanceBallLaunch() {
	b := &g.Bonus
	if b.failed {
		return
	}
	if b.spawnAccumulator >= 44*256 {
		g.spawnBall()
	} else {
		b.spawnAccumulator = (b.spawnAccumulator + b.launchRate) & 65535
	}
}

func (g *Game) spawnBall() bool {
	b := &g.Bonus
	for len(b.Balls) < 4 {
		b.Balls = append(b.Balls, Ball{})
	}
	id := 3
	for id >= 0 && b.Balls[id].source.Flags != 0 {
		id--
	}
	if id < 0 {
		return false
	}
	if b.Number >= 64 {
		b.spawnDone = true
		return false
	}
	p := &b.Balls[id]
	entropy := g.OriginalVBL()
	waves := g.ReflectionPhases()
	entropy = entropy>>4 | entropy<<4
	entropy = (entropy ^ waves[0]) + waves[3]
	p.source.Gravity, p.source.Floor = byte(b.ballGravity), 85
	p.source.Y = uint16(nativeGameplay.Bonus.BallHeights[entropy&7]) << 8
	p.source.VY, p.source.Flags = 0, 1
	p.source.VX = int16(b.launchRate)
	p.source.X = 4<<8 | p.source.X&255
	if entropy&128 != 0 {
		p.source.X = 182<<8 | p.source.X&255
		p.source.VX = -p.source.VX
	}
	b.Number++
	b.spawnAccumulator = 0
	level := min(max((g.bonusCount-1)/2, 0), 7)
	cap := int(nativeGameplay.Bonus.BallRates[level])
	if b.launchRate < 256 || b.launchRate&255 < cap {
		b.launchRate += 8
	}
	if b.ballGravity < 56 {
		b.ballGravity++
	}
	color := int(g.originalRandomByte() & 7)
	p.Frame, p.Color = color, color
	if id >= int(nativeGameplay.Bonus.BallLimits[level]) {
		p.source.Flags = 49
		p.Color, p.Frame = 1, 0
		p.source.Y = uint16(nativeGameplay.Bonus.BallHeights[(g.OriginalVBL()+waves[6])&1]) << 8
	}
	g.syncOriginalBalls()
	return true
}

func (g *Game) syncOriginalBalls() {
	b := &g.Bonus
	b.Ball = Ball{}
	nearest := math.MaxFloat64
	for id := range b.Balls {
		p := &b.Balls[id]
		p.X, p.Y = originalBallPosition(p.source)
		p.VX, p.VY = float64(p.source.VX)/128, float64(p.source.VY)/256
		p.Active, p.Deflected, p.Flashing = p.source.Flags != 0, p.source.Flags&2 != 0, p.source.Flags&32 != 0
		if p.Active {
			distance := math.Abs(p.X - bodyCenter(g.Fighters[b.Fighter]))
			if distance < nearest {
				nearest = distance
				b.Ball = *p
			}
		}
	}
	f := &g.Fighters[b.Fighter]
	f.ShieldX, f.ShieldY = float64(b.shieldMotion.X>>7)-32, float64(b.shieldMotion.Y>>8)+96
	// Clearing hides the board independently of its remaining motion updates.
	b.ShieldVisible = b.shieldMotion.Flags != 0 && !b.finalClearing
}

func (g *Game) tickBalls(inputs [2]Input) {
	if g.pacedBallPreparation() {
		g.tickPacedBalls(inputs)
		return
	}
	b := &g.Bonus
	b.Ticks++
	if g.tickBonusPreparation(inputs) {
		return
	}
	mask := g.Fighters[b.Fighter]
	if b.maskReady {
		mask = b.contactFighter
	}
	if !b.ShieldVisible {
		mask.ShieldX = -10000
	}
	g.tickOriginalBallObjects(mask)
	g.updateSoundEffects()
	f := &g.Fighters[b.Fighter]
	b.contactFighter, b.maskReady = *f, true
	if b.failed {
		if !b.deathStarted {
			b.deathStarted = true
			g.soundDown[b.Fighter] = 1
			f.Action, f.Move, f.reactionAction = b.failureAction, Knockdown, b.failureAction
			g.resetSourceFighter(b.Fighter)
			f.computerAnimation.Pending = byte(b.failureAction)
			g.advanceBonusDeathActor(5)
			b.deathVisible = true
		} else {
			g.advanceBonusDeathActor(5)
			b.deathVisible = true
		}
	} else {
		f.Age++
		g.tickShield(b.Fighter, g.bonusInput(inputs))
		g.syncShieldMotionFromPose()
	}
	if b.activeCount == 0 && (b.failed && b.deathVisible || !b.failed && b.spawnDone) {
		g.finishBonus(!b.failed)
		if b.failed && len(b.queue) > 1 {
			g.continueAfterBonus(inputs)
		}
	}
}
