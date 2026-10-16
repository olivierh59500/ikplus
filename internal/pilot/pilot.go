// Package pilot drives a normal human joystick from visible match state.
// It never changes the simulation, selects internal actions, or reads its RNG.
package pilot

import (
	"math"

	"github.com/olivierh59500/ikplus/internal/game"
)

// Pilot is a deterministic skilled-player controller. Combat decisions are
// refreshed at 100 ms intervals and attacks are held through their contact pose.
type Pilot struct {
	player          int
	phase           game.Phase
	round           int
	input           game.Input
	next            uint64
	until           uint64
	attack          bool
	previousBalls   []game.Ball
	ballGravity     []float64
	ballHalfNext    []bool
	lastBallFrame   uint64
	previousCenters [3]float64
	velocity        [3]float64
	observedFrame   uint64
	shieldFrame     uint64
	shieldPose      int
	previousShield  int
}

// New selects the white (0) or red (1) human joystick.
func New(player int) *Pilot {
	if player < 0 || player > 1 {
		player = 0
	}
	return &Pilot{player: player}
}

// Input observes the current frame and returns only legal joystick controls.
// Call it once before each game.Tick; player slots never change during a match.
func (p *Pilot) Input(match *game.Game) game.Input {
	observed := match.Observation()
	g := &observed
	if p.phase != g.Phase || p.round != g.Round {
		p.phase, p.round, p.input = g.Phase, g.Round, game.Input{}
		p.next, p.until, p.attack = 0, 0, false
		p.previousBalls, p.ballGravity, p.lastBallFrame = nil, nil, 0
		p.ballHalfNext = nil
		p.velocity, p.observedFrame = [3]float64{}, g.Frame
		p.shieldFrame = g.Frame
		p.shieldPose = g.Fighters[p.player].ShieldState
		p.previousShield = p.shieldPose
		for i, f := range g.Fighters {
			p.previousCenters[i] = center(f)
		}
	}
	if g.Phase == game.BallBonus && p.shieldFrame != g.Frame {
		p.shieldFrame, p.previousShield = g.Frame, p.shieldPose
		p.shieldPose = g.Fighters[p.player].ShieldState
	}
	if g.Frame < p.next {
		return p.input
	}
	p.next = g.Frame + 5
	switch g.Phase {
	case game.Ready, game.Fight:
		// A human can leave the opening bow with ordinary joystick input.
		// Waiting for an opponent to hit first can lose an entire round.
		dt := float64(g.Frame - p.observedFrame)
		if dt > 0 {
			for i, f := range g.Fighters {
				p.velocity[i] = math.Max(-2.4, math.Min(2.4, (center(f)-p.previousCenters[i])/dt))
			}
		}
		p.input = p.fight(g)
		p.observedFrame = g.Frame
		for i, f := range g.Fighters {
			p.previousCenters[i] = center(f)
		}
	case game.BallBonus:
		p.input = p.balls(g)
	case game.BombBonus:
		dt := float64(g.Frame - p.observedFrame)
		if dt > 0 {
			for id, f := range g.Fighters {
				p.velocity[id] = math.Max(-2.4, math.Min(2.4, (center(f)-p.previousCenters[id])/dt))
			}
		}
		p.input = p.bombs(g)
		p.observedFrame = g.Frame
		for id, f := range g.Fighters {
			p.previousCenters[id] = center(f)
		}
	default:
		p.input = game.Input{}
	}
	return p.input
}

func center(f game.Fighter) float64 {
	if f.Facing < 0 {
		return f.X + 24
	}
	return f.X - 8
}

func sign(v float64) int {
	if v < 0 {
		return -1
	}
	return 1
}

func (p *Pilot) fight(g *game.Observation) game.Input {
	f := g.Fighters[p.player]
	if !f.Human {
		return game.Input{}
	}
	if f.Move == game.Knockdown || f.Move == game.Rise {
		p.attack, p.until = false, 0
		return game.Input{Y: -1}
	}
	if g.Frame < p.until {
		return p.input
	}
	if p.attack {
		p.attack = false
		p.until = g.Frame + 5
		return game.Input{}
	}
	if in, ok := p.guard(g); ok {
		return in
	}
	target, distance := -1, math.MaxFloat64
	for i, o := range g.Fighters {
		if i == p.player || o.Move == game.Knockdown || o.Move == game.Rise {
			continue
		}
		d := math.Abs(center(o) - center(f))
		if d < distance {
			target, distance = i, d
		}
	}
	if target < 0 {
		return game.Input{}
	}
	o := g.Fighters[target]
	direction := sign(center(o) - center(f))
	if direction != f.Facing {
		p.attack, p.until = true, g.Frame+20
		return game.Input{X: -f.Facing, Y: -1}
	}
	if in, ok := p.attackAt(g, target); ok {
		return in
	}
	if distance < 22 {
		return game.Input{X: -f.Facing}
	}
	return game.Input{X: direction}
}

// balls predicts each observed arc against the static original shield/body
// outlines. It never clones or advances the game, reads RNG, or selects a source
// action: only ordinary joystick directions are returned.
func (p *Pilot) balls(g *game.Observation) game.Input {
	if g.Bonus.Fighter != p.player {
		return game.Input{}
	}
	if len(p.ballGravity) != len(g.Bonus.Balls) {
		p.ballGravity = make([]float64, len(g.Bonus.Balls))
		p.ballHalfNext = make([]bool, len(g.Bonus.Balls))
	}
	nearest, arrival := -1, math.MaxFloat64
	for id, b := range g.Bonus.Balls {
		if !b.Active || b.Deflected || b.VX == 0 {
			continue
		}
		fresh := id >= len(p.previousBalls) || !p.previousBalls[id].Active || p.previousBalls[id].Deflected
		if !fresh && math.Abs(b.X-p.previousBalls[id].X) > 20 {
			fresh = true
		}
		if fresh {
			p.ballHalfNext[id] = b.Flashing
			p.ballGravity[id] = 12.0 / 256
		}
		if id < len(p.previousBalls) && !fresh {
			prev := p.previousBalls[id]
			dt := float64(g.Frame - p.lastBallFrame)
			if dt > 0 && b.VY >= prev.VY {
				p.ballGravity[id] = (b.VY - prev.VY) / dt
			}
			if b.Flashing && prev.VY > 0 && b.VY < 0 {
				p.ballHalfNext[id] = !p.ballHalfNext[id]
			}
		}
		x := 152.0
		if b.VX < 0 {
			x = 184
		}
		t := (x - b.X) / b.VX
		if t >= -8 && t < arrival {
			nearest, arrival = id, t
		}
	}
	p.previousBalls = append(p.previousBalls[:0], g.Bonus.Balls...)
	p.lastBallFrame = g.Frame
	if nearest < 0 {
		return game.Input{}
	}
	ball := g.Bonus.Balls[nearest]
	choices := []game.Input{{X: -1, Y: -1}, {X: -1}, {X: -1, Y: 1}, {X: 1, Y: -1}, {X: 1}, {X: 1, Y: 1}, {}}
	best, cost := game.Input{}, math.MaxFloat64
	for _, input := range choices {
		worst := 0.0
		// The picture truncates both 8.8 fractions. Plan several plausible
		// subpixel positions instead of treating the integer pixel as exact.
		for _, fraction := range [...]float64{0.1, 0.5, 0.9} {
			state := g.Fighters[p.player].ShieldState
			maskState := p.previousShield
			x, y, vy := ball.X+fraction, ball.Y+fraction, ball.VY
			spin := ball.Frame
			half := p.ballHalfNext[nearest]
			gravity := p.ballGravity[nearest]
			if gravity <= 0 || gravity > 0.25 {
				gravity = 12.0 / 256
			}
			result := 1000.0
			for tick := 0; tick < 240; tick++ {
				body, weapon := game.ShieldPoseContact(maskState, ball.Color, spin, int(math.Floor(x)), int(math.Floor(y)))
				if weapon {
					result = float64(tick)
					break
				}
				if body {
					result = 5000 + float64(240-tick)
					break
				}
				vy += gravity
				y += vy
				if y >= 187 {
					whole := math.Floor(y)
					y = 374 - whole + (y - whole)
					vy = -vy
					if ball.Flashing {
						if half {
							vy /= 2
						} else {
							vy *= 2
						}
						half = !half
					}
				}
				x += ball.VX
				spin = (spin + sign(ball.VX) + 9) % 9
				maskState, state = state, game.NextShieldState(state, input)
				if x < -20 || x > 345 {
					break
				}
			}
			worst = math.Max(worst, result)
		}
		if worst < cost {
			best, cost = input, worst
		}
	}
	return best
}

const bombIgnitionWarningFrame = 12

func (p *Pilot) bombs(g *game.Observation) game.Input {
	f := g.Fighters[p.player]
	if f.Move == game.WalkForward || f.Move == game.WalkBackward {
		f.X += p.velocity[p.player] * 4
	}
	if g.Bonus.Fighter != p.player {
		return game.Input{}
	}
	if input, danger := p.avoidFragments(g); danger {
		p.attack, p.until = false, 0
		return input
	}
	// A turning sequence retains the previous joystick interpretation until
	// it is released. Let it finish before requesting absolute walking input.
	if f.Move == game.ReversePunch || f.Move == game.ReverseSweep {
		return game.Input{}
	}
	// A kicked bomb still burns its fuse while travelling. Keep clear of its
	// final ignition sequence rather than following it into the blast radius.
	for _, b := range g.Bonus.Bombs {
		if b.Fragment {
			continue
		}
		if !b.Active || !bombNeedsClearance(b) {
			continue
		}
		// Visible ignition is already a warning. Leaving room during the
		// first ignition avoids waiting for the shorter final burn window.
		last := 36
		if b.Stage == 2 {
			last = 26
		}
		// Visible fuse cells advance every four PAL ticks, while the observed
		// projectile velocity is measured per tick. Predict in the same units.
		x := b.X + b.VX*float64(4*(last-b.StageTicks))
		if math.Abs(f.X-x) < 85 {
			p.attack, p.until = false, 0
			direction := sign(f.X - x)
			if x+85 > game.Width-18 {
				direction = -1
			}
			if x-85 < 18 {
				direction = 1
			}
			return game.Input{X: direction}
		}
	}
	if g.Frame < p.until {
		return p.input
	}
	if p.attack {
		p.attack = false
		p.until = g.Frame + 5
		return game.Input{}
	}
	target, distance := -1, math.MaxFloat64
	for i, b := range g.Bonus.Bombs {
		if !b.Active || b.Fragment || b.Kicked || b.Explosion > 0 || bombNeedsClearance(b) {
			continue
		}
		d := math.Abs(b.X - f.X)
		if target < 0 || b.Fuse < g.Bonus.Bombs[target].Fuse {
			target, distance = i, d
		}
	}
	if target < 0 {
		return game.Input{}
	}
	b := g.Bonus.Bombs[target]
	if (b.X-f.X)*float64(f.Facing) < 14 && math.Abs(b.X-center(f)) < 45 {
		// A small step back is safer than a full turn when the bomb is
		// beside the feet. Preserve the current facing and make kick room.
		return game.Input{X: -f.Facing}
	}
	direction := sign(b.X - center(f))
	if direction != f.Facing {
		landing := f.X - 42*float64(f.Facing)
		for _, hazard := range g.Bonus.Bombs {
			if hazard.Active && bombNeedsClearance(hazard) && math.Abs(landing-hazard.X) < 85 {
				return game.Input{}
			}
		}
		p.attack = true
		p.until = g.Frame + 20
		// Use the original turning punch, then release its held pose.
		return game.Input{X: -f.Facing, Y: -1}
	}
	if (b.X-f.X)*float64(f.Facing) < 14 {
		return game.Input{X: -f.Facing}
	}
	kickDistance := 36.0
	if f.Facing < 0 {
		kickDistance = 20
	}
	if distance > kickDistance+5 {
		return game.Input{X: direction}
	}
	if distance < kickDistance-9 {
		return game.Input{X: -f.Facing}
	}
	p.until = g.Frame + 15
	return game.Input{Y: 1}
}

// avoidFragments extrapolates visible debris for one human reaction window
// and move startup. Its pose queries inspect only the static original artwork.
func (p *Pilot) avoidFragments(g *game.Observation) (game.Input, bool) {
	f := g.Fighters[p.player]
	if f.Move == game.Jump || f.Move == game.FlyingKick || f.Move == game.Backflip {
		return game.Input{}, true
	}
	soonest := 31
	for _, b := range g.Bonus.Bombs {
		if !b.Active || !b.Fragment {
			continue
		}
		x, y, vy := b.X, b.Y, b.VY
		for tick := 0; tick < soonest; tick++ {
			if y > 205 {
				break
			}
			for spin := 0; spin < 8; spin++ {
				if game.FragmentPoseContact(0, f.Facing, int(f.X), b.Variant, spin, int(math.Floor(x)), int(math.Floor(y))) {
					soonest = tick
					break
				}
			}
			x += b.VX
			vy += 24.0 / 256
			y += vy
		}
	}
	if soonest > 30 {
		return game.Input{}, false
	}
	choices := []game.Input{{Y: 1}, {Y: -1}, {X: -f.Facing}, {X: f.Facing}}
	best, risk := game.Input{}, math.MaxFloat64
	for _, in := range choices {
		cost := 0.0
		for _, b := range g.Bonus.Bombs {
			if !b.Active || !b.Fragment {
				continue
			}
			x, y, vy := b.X, b.Y, b.VY
			for tick := 0; tick < 40 && y <= 205; tick++ {
				anchor, pose := f.X, f.SpriteFrame
				if tick >= 8 {
					step := (tick - 8) / 4
					switch {
					case in.Y > 0:
						pose = 32
						if step == 0 {
							pose = 33
						}
					case in.Y < 0:
						frames := [...]int{5, 22, 23, 22, 5, 0}
						pose = frames[min(step, len(frames)-1)]
						anchor += float64(f.Facing) * math.Min(12, float64(tick-8)*0.5)
					default:
						pose = [...]int{1, 2, 3, 59, 12, 0}[step%6]
						anchor += float64(in.X) * float64(tick-8) * 2.2
					}
				}
				anchor = math.Max(32, math.Min(304, anchor))
				for spin := 0; spin < 8; spin++ {
					if game.FragmentPoseContact(pose, f.Facing, int(anchor), b.Variant, spin, int(math.Floor(x)), int(math.Floor(y))) {
						cost += float64(41 - tick)
						break
					}
				}
				x += b.VX
				vy += 24.0 / 256
				y += vy
			}
		}
		if cost < risk {
			best, risk = in, cost
		}
	}
	return best, true
}

func bombNeedsClearance(b game.Bomb) bool {
	return b.Explosion > 0 || b.Stage == 3 || b.Stage == 4 || b.Stage == 2 && b.StageTicks >= bombIgnitionWarningFrame
}
