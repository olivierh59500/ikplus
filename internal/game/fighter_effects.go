package game

// RecoveryWait is the source countdown used by the drool drawing routine.
func (f Fighter) RecoveryWait() int { return f.recoveryWait }

// $9290 uses the original strike's fixed star cell and contact coordinates.
// $91D6 advances the drool cell once per prepared actor drawing, not per VBL.
func (g *Game) advanceFighterEffects(id int) {
	f := &g.Fighters[id]
	if f.SpriteFrame >= 36 && f.SpriteFrame <= 47 && f.recoveryWait != 0 {
		f.DroolFrame = (f.DroolFrame + 1) % 5
	}
}

func (g *Game) publishFighterEffects(id int, flags byte) {
	if id < 2 && flags&1 != 0 {
		// $9C22 also changes overlap priority when a marked pose is published.
		g.drawPriority = byte(id)
	}
	if flags&32 != 0 {
		g.Fighters[id].Puff = false
	}
}

func (g *Game) setHitEffects(h contact) {
	f := &g.Fighters[h.target]
	index := strikeIndex(h.frame)
	style := 0
	if h.points == 1 {
		style = 1
	}
	f.StarFrame, f.StarColor = nativeGameplay.HitStars[style][index], byte(h.attacker)
	f.StarX, f.StarY = byte(int(f.X/2)+12), byte(121+h.target*4)
	f.Puff = sourceCollisions.reactions[index*4+(h.orientation&3)]&128 != 0
	f.PuffX = byte(int(f.X/2) + 4)
	if f.Facing < 0 {
		f.PuffX += 18
	}
}
