package game

// Actor drawing samples effect timers after restoration and the preceding
// actors. Retaining that work prevents a nearly expired drool timer from
// advancing a cell before the original drawing would reach it.
type fighterEffectPreparation struct {
	bank  int
	frame uint64
	order [3]int
	actor int
	debt  int
	poses [3]Fighter
}

func (g *Game) prepareTimedFighterEffects() {
	_, bank := g.SceneryBuffers()
	p := g.Picture(bank)
	debt := 3500 + 1046 + p.restoreRows*79 + p.restoreRuns*181
	g.fighterEffectPreparation = &fighterEffectPreparation{bank: bank, frame: g.Frame,
		order: [3]int{2, int(g.drawPriority&1) ^ 1, int(g.drawPriority & 1)}, debt: debt, poses: g.Fighters, actor: -1}
	g.advanceTimedFighterEffects(259 * picturePreparationQuota / 313)
}

func (g *Game) advanceTimedFighterEffects(position int) {
	w := g.fighterEffectPreparation
	if w == nil {
		return
	}
	for position < picturePreparationQuota {
		used := min(w.debt, picturePreparationQuota-position)
		w.debt -= used
		position += used
		if w.debt != 0 {
			return
		}
		if w.actor >= 0 {
			id := w.order[w.actor]
			f := &g.Fighters[id]
			if w.poses[id].SpriteFrame >= 36 && w.poses[id].SpriteFrame <= 47 && f.recoveryWait != 0 {
				f.DroolFrame = (f.DroolFrame + 1) % 5
			}
			p := &g.pictures[w.bank]
			if p.Frame == w.frame {
				p.Fighters[id].DroolFrame = f.DroolFrame
				p.Fighters[id].recoveryWait = f.recoveryWait
				p.Fighters[id].Stun = f.Stun
			}
		}
		w.actor++
		if w.actor >= len(w.order) {
			g.fighterEffectPreparation = nil
			return
		}
		id := w.order[w.actor]
		f := w.poses[id]
		w.debt = actorPreparationWork(f.SpriteFrame, int(f.X/2))
	}
}
