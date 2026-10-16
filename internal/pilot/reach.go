package pilot

import (
	"encoding/json"
	"math"

	"github.com/olivierh59500/ikplus/assets"
	"github.com/olivierh59500/ikplus/internal/game"
)

// The tactical player knows the original visible strike geometry, just as a
// practiced player learns which poses and distances each joystick move hits.
// It evaluates the current picture only; it never advances a speculative game
// or predicts the computer's next action or random byte.
type strike struct {
	move              game.Move
	x, y              int
	fire              bool
	startup, hold     int
	displacement      float64
	frame, preference int
	low               bool
}

var strikes = []strike{
	{game.Headbutt, 1, -1, true, 6, 15, 12, 27, 120, false},
	{game.FrontPunch, 1, -1, false, 6, 15, 10, 20, 90, false},
	{game.CrouchPunch, -1, 1, false, 6, 15, 8, 18, 80, false},
	{game.ShinKick, 1, 1, false, 6, 15, 8, 21, 120, true},
	{game.Sweep, 0, 1, false, 6, 15, 4, 32, 120, true},
	{game.FaceKick, 1, 1, true, 11, 20, -12, 16, 95, false},
	{game.StomachKick, 1, 0, true, 11, 20, -18, 14, 65, false},
	{game.FlyingKick, 0, -1, true, 11, 30, 18, 69, 85, false},
}

type anatomy struct {
	frames, tips, masks, widths, poses, bodies []byte
}

// Extracted contact geometry describes the rendered anatomy. Reading this
// static artwork metadata does not inspect private match state.
var knownAnatomy = func() anatomy {
	raw, err := assets.Read("original/gameplay.json")
	if err != nil {
		panic(err)
	}
	var decoded struct {
		Collision struct{ Frames, Tips, Masks, Widths, Poses, Bodies []byte }
	}
	if err = json.Unmarshal(raw, &decoded); err != nil {
		panic(err)
	}
	c := decoded.Collision
	return anatomy{c.Frames, c.Tips, c.Masks, c.Widths, c.Poses, c.Bodies}
}()

func strikeError(f, o game.Fighter, s strike) float64 {
	if s.low && o.Height >= 9 {
		return math.MaxFloat64
	}
	if !s.low && (o.Move == game.Block || o.Move == game.WalkBackward) && o.Facing != f.Facing {
		return math.MaxFloat64
	}
	if o.SpriteFrame < 0 || o.SpriteFrame >= len(knownAnatomy.poses) {
		return math.MaxFloat64
	}
	pose := int(knownAnatomy.poses[o.SpriteFrame])
	if pose >= 128 {
		return math.MaxFloat64
	}
	index := -1
	for i, frame := range knownAnatomy.frames {
		if int(frame) == s.frame {
			index = i
			break
		}
	}
	if index < 0 {
		return math.MaxFloat64
	}
	if pose == 23 && index == 2 {
		pose = 35
	}
	self := f.X + float64(f.Facing)*s.displacement
	forward := (center(o)-center(f))*float64(f.Facing) > 0
	if pose == 19 && !forward && o.Facing == f.Facing {
		pose = 0
	}
	tip := float64(knownAnatomy.tips[index])
	if f.Facing < 0 {
		tip = 40 - tip
	}
	point := self/2 + tip
	width := float64(knownAnatomy.widths[index])
	best := math.MaxFloat64
	for c := 0; c < 6; c++ {
		if knownAnatomy.masks[index]&(32>>c) == 0 {
			continue
		}
		body := float64(knownAnatomy.bodies[pose*6+c])
		if body >= 128 {
			continue
		}
		if o.Facing < 0 {
			body = 40 - body
		}
		bodyPoint := o.X/2 + body
		delta := point - bodyPoint
		if f.Facing < 0 {
			delta = -delta
		}
		if f.Facing == o.Facing {
			delta += width
		}
		gap := math.Max(0, math.Max(-delta, delta-width)) * 2
		best = math.Min(best, gap)
	}
	return best
}

func (p *Pilot) attackAt(g *game.Observation, target int) (game.Input, bool) {
	f, o := g.Fighters[p.player], g.Fighters[target]
	best, quality := -1, -math.MaxFloat64
	for i, s := range strikes {
		predicted := o
		// Account for one visible walking update before the joystick reaches
		// contact. This uses measured past motion, never a future game copy.
		if o.Move == game.WalkForward || o.Move == game.WalkBackward || o.Move == game.Jump {
			predicted.X += p.velocity[target] * 5
		}
		err := strikeError(f, predicted, s)
		if err > 4 {
			continue
		}
		q := float64(s.preference) - err*5
		if q > quality {
			best, quality = i, q
		}
	}
	if best < 0 {
		return game.Input{}, false
	}
	s := strikes[best]
	p.attack, p.until = true, g.Frame+uint64(s.hold)
	return game.Input{X: s.x * f.Facing, Y: s.y, Fire: s.fire}, true
}
