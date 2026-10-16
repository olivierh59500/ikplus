package game

// FightSnapshot retains a completed arena step before the next picture's
// controllers run. It gives replay verification a stable source-main boundary.
type FightSnapshot struct {
	Frame                        uint64
	Counter, Mode, Round, Humans byte
	Random                       uint64
	Actors                       [3]TitleActorState
	Points, Scores               [3]int
	Caption                      CaptionState
	Effects                      [3]FighterEffectState
}

type FighterEffectState struct {
	Stun, RecoveryWait, GetUp                 byte
	Drool, StarFrame, StarColor, StarX, StarY byte
	Puff, PuffX                               byte
}

func (g *Game) FightSnapshot() FightSnapshot { return g.fightSnapshot }

// FightSnapshots retains consecutive dispatcher visits in the current update.
// A winning contact can return first in fight mode, then in judge mode before
// that program prepares its actors. Inspection must not collapse those visits.
func (g *Game) FightSnapshots() []FightSnapshot {
	if g.fightSnapshot.Frame != g.Frame {
		return nil
	}
	return g.fightSnapshots
}

func (g *Game) rememberFightBoundary() {
	g.rememberFightBoundaryMode(g.OriginalMode())
}

func (g *Game) rememberFightBoundaryMode(mode byte) {
	s := FightSnapshot{Frame: g.Frame, Counter: g.displayCounter, Mode: mode, Round: byte(g.Round), Random: g.random, Caption: g.Caption()}
	for id, fighter := range g.Fighters {
		if fighter.Human {
			s.Humans |= 1 << id
		}
		s.Actors[id] = fighter.computerAnimation
		s.Points[id], s.Scores[id] = fighter.Points, fighter.Score
		puff := byte(0)
		if fighter.Puff {
			puff = 1
		}
		s.Effects[id] = FighterEffectState{byte(fighter.Stun), byte(fighter.recoveryWait), byte(fighter.GetUp), byte(fighter.DroolFrame), fighter.StarFrame, fighter.StarColor, fighter.StarX, fighter.StarY, puff, fighter.PuffX}
	}
	g.fightSnapshot = s
	if len(g.fightSnapshots) != 0 && g.fightSnapshots[0].Frame != g.Frame {
		g.fightSnapshots = g.fightSnapshots[:0]
	}
	g.fightSnapshots = append(g.fightSnapshots, s)
}
