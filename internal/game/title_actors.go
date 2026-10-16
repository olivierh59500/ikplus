package game

// TitleActors owns the original no-contact computer display used by $25F0.
// Unlike arena combat, blue/red/white each advance once per title-loop update.
type TitleActors struct {
	actors [3]computerAnimationState
	Round  byte
}

// TitleActorState exposes copied source animation bytes for independent
// sequence comparisons and memory-indexed RNG observations.
type TitleActorState = computerAnimationState

func (t *TitleActors) States() [3]TitleActorState { return t.actors }

func NewTitleActors(positions, facing [3]byte, round byte) *TitleActors {
	t := &TitleActors{Round: round}
	for id := range t.actors {
		t.actors[id].Position, t.actors[id].Facing = positions[id], facing[id]
		action := byte(0)
		if id < 2 {
			// $25E0 leaves D0=$57 after blue's reset. The loop does not
			// clear it again before selecting red and white at $9830.
			action = 87
		}
		t.actors[id].Idle = action
		t.actors[id].selectDirect(action)
		t.actors[id].Frame = 87
	}
	return t
}

// Step follows $2890 then the three $6FF0/$98F8 calls. The shared game RNG is
// supplied by the owner; no collision, score or countdown routine runs here.
func (t *TitleActors) Step(frame byte, random func() byte) {
	t.StepWithPublication(frame, random, nil)
}

// StepWithPublication lets the owner publish and paint each actor before the
// next actor runs. The original RNG can read that live actor/mask memory.
func (t *TitleActors) StepWithPublication(frame byte, random func() byte, publish func(int, TitleActorState), keys ...byte) {
	key := byte(0)
	if len(keys) != 0 {
		key = keys[0]
	}
	defence := int8(1)
	if frame&64 != 0 {
		defence = 5
	}
	target := byte(6)
	if frame&8 != 0 {
		target = 0
	}
	for id := 2; id >= 0; id-- {
		s := aiSnapshot{id: byte(id), stage: 8, enabled: 1, mode: 0, targetOffset: target, difficulty: 5, defence: defence, activity: 5}
		for actor, state := range t.actors {
			s.poses[actor] = aiPose{x: state.Position, facing: state.Facing, action: state.Action, flags: state.Flags, held: state.Held, idle: state.Idle}
		}
		action, _ := originalDecision(s, random)
		if action >= 0 {
			t.actors[id].issueWithKey(byte(action), t.Round, key, random)
		}
		t.actors[id].advanceWithKey(8, 5, t.Round, key, 0, 0, random)
		if publish != nil {
			publish(id, t.actors[id])
		}
	}
}

// Fighters exposes only the authored poses needed by the native title painter.
func (t *TitleActors) Fighters() [3]Fighter {
	var result [3]Fighter
	for id, s := range t.actors {
		facing := 1
		if s.Facing != 0 {
			facing = -1
		}
		result[id] = Fighter{ID: id, X: float64(s.Position) * 2, Y: GroundY, Facing: facing, SpriteFrame: int(s.Frame), Action: int(s.Action), computerAnimation: s, computerAnimationReady: true}
	}
	return result
}
