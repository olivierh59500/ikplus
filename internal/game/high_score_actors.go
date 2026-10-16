package game

// HighScoreActors follows $A32C/$7772. White holds the authored pose while
// red and blue advance in that order, without combat contacts or countdown.
type HighScoreActors struct {
	TitleActors
	Difficulty byte
	Wait, Rest [3]byte
}

// NewHighScoreActors translates $A332..$A360. The caller performs the random
// priority read before HoldWhite, preserving the original initialization order.
func NewHighScoreActors(initial [3]TitleActorState, round, difficulty byte) *HighScoreActors {
	h := &HighScoreActors{TitleActors: TitleActors{actors: initial, Round: round}, Difficulty: difficulty}
	for id := 2; id >= 0; id-- {
		action := byte(87)
		if id == 2 {
			action = 0
		}
		s := &h.actors[id]
		s.Pending, s.Idle = 0, action
		s.selectDirect(action)
		s.Frame = 87
	}
	return h
}

// HoldWhite follows $A36E after the source's intervening random priority read.
func (h *HighScoreActors) HoldWhite() {
	h.actors[0].Position, h.actors[0].Facing, h.actors[0].Frame = 18, 0, 63
}

// Step follows $7772's red/blue controller and animation calls.
func (h *HighScoreActors) Step(random func() byte, keys ...byte) {
	key := byte(0)
	if len(keys) != 0 {
		key = keys[0]
	}
	for id := 1; id < 3; id++ {
		s := aiSnapshot{id: byte(id), stage: 4, enabled: 1, mode: 0, targetOffset: 6, difficulty: h.Difficulty, defence: 2, activity: 5}
		for actor, state := range h.actors {
			points := byte(1)
			if actor == 0 {
				points = 0
			}
			s.poses[actor] = aiPose{x: state.Position, facing: state.Facing, action: state.Action, flags: state.Flags, held: state.Held, idle: state.Idle, points: points}
		}
		action, _ := originalDecision(s, random)
		if action >= 0 {
			h.actors[id].issueWithKey(byte(action), h.Round, key, random)
		}
		h.actors[id].advanceWithKey(4, h.Difficulty, h.Round, key, h.Wait[id], h.Rest[id], random)
	}
}
