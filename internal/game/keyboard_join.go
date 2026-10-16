package game

// pollSourceJoin follows $69D2/$6A26/$6A2E. Consumed commands reach the
// physical-fire branch directly; only ordinary polling requires $608>=25.
func (g *Game) pollSourceJoin(handled, ready bool) {
	if g.sourceJoinPlayers != 0 || g.restartRequested || !handled && (!ready || g.OriginalVBL() < 25) {
		return
	}
	players := 0
	if !handled {
		switch g.speedKeys[0] {
		case 0x3b:
			players = 1
		case 0x3c:
			players = 2
		}
	}
	humans := g.Fighters[0].Human || g.Fighters[1].Human
	if holder := g.presentationState; holder != nil {
		humans = holder.Humans[0]|holder.Humans[1] != 0
	}
	if players == 0 && !humans {
		if g.joystick[0].Fire {
			players = 1
		} else if g.joystick[1].Fire {
			players = 2
		}
	}
	if players == 0 {
		return
	}
	g.sourceJoinPlayers = players
	g.speedKeyDelay, g.speedKeys[0], g.speedKeys[1] = 0, 0, 0
	if holder := g.presentationState; holder != nil {
		holder.Humans[0], holder.Humans[1], holder.JoinPending = 1, byte(players-1), 1
	}
}
