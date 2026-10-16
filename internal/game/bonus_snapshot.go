package game

// BonusSnapshot exposes the translated source storage for independent
// comparisons. It is read-only and is not an observation used by the pilot.
type BonusSnapshot struct {
	Preparing, Balls, Failed, Done bool
	Owner, Number                  int
	Slots                          [10][16]uint16
}

// OriginalBonusSnapshot samples active and retained projectile words without
// advancing their integrators, random state, fuse timers or display history.
func (g *Game) OriginalBonusSnapshot() BonusSnapshot {
	b := &g.Bonus
	state := BonusSnapshot{Preparing: g.BonusPreparing(), Balls: b.kind == 0,
		Owner: b.Fighter, Number: b.Number, Failed: b.failed, Done: b.spawnDone}
	slots := g.bonusMemory
	if g.Phase == BallBonus || g.Phase == BombBonus || g.Phase == BonusEnd {
		if b.kind == 1 {
			copy(slots[:], b.Bombs)
		} else {
			for id, ball := range b.Balls {
				if id == 4 {
					break
				}
				slots[id].source = ball.source
				slots[id].Frame, slots[id].Color, slots[id].drawn = ball.Frame, ball.Color, ball.drawn
			}
			slots[4].source = b.shieldMotion
			slots[4].Frame, slots[4].Color = b.shieldFrame, b.shieldColor
		}
	}
	for id, p := range slots {
		s := p.source
		drawn := uint16(0)
		if p.drawn {
			drawn = 1
		}
		state.Slots[id] = [16]uint16{s.X, s.Y, uint16(s.VX), uint16(s.VY), uint16(s.Gravity), uint16(s.Floor), uint16(s.Flags), uint16(s.Bounces), uint16(p.Stage), uint16(p.StageTicks), uint16(byte(p.burnTicks)), uint16(p.Frame), uint16(p.Color), uint16(p.blastCounter), uint16(p.Variant), drawn}
	}
	return state
}
