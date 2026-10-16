package pilot

import "github.com/olivierh59500/ikplus/internal/game"

// Guard recognizes the displayed contact pose rather than a computer action
// identifier. Original computer aliases can punch while their public Move tag
// is Idle; their visible sprite is still the same dangerous punching picture.
func (p *Pilot) guard(g *game.Observation) (game.Input, bool) {
	f := g.Fighters[p.player]
	if f.Move == game.Jump || f.Move == game.Backflip || f.Move == game.FlyingKick || f.Move == game.DoubleKick {
		return game.Input{}, false
	}
	for i, o := range g.Fighters {
		if i == p.player || o.Move == game.Knockdown || o.Move == game.Rise {
			continue
		}
		for _, frame := range knownAnatomy.frames {
			if o.SpriteFrame != int(frame) {
				continue
			}
			s := strike{frame: int(frame), low: frame == 21 || frame == 32}
			if strikeError(o, f, s) > 8 {
				continue
			}
			p.attack, p.until = false, 0
			if sign(center(o)-center(f)) != f.Facing {
				return game.Input{X: f.Facing}, true
			}
			if s.low {
				p.attack, p.until = true, g.Frame+uint64(game.MoveDuration(game.Backflip))
				return game.Input{X: -f.Facing, Fire: true}, true
			}
			return game.Input{X: -f.Facing}, true
		}
	}
	return game.Input{}, false
}
