package game

// TitleActorWork prepares one actor transaction. Cached random returns let a
// decision resume on another update without consuming its earlier draws again.
// Actor state is published only when the complete operation is ready.
type TitleActorWork struct {
	id            int
	before, after computerAnimationState
	snapshot      aiSnapshot
	round, key    byte
	draws         []byte
	done          bool
}

type titleActorNeedsRandom struct{}

// PrepareActor snapshots the decision inputs before any new values are drawn.
func (t *TitleActors) PrepareActor(id int, frame, key byte) *TitleActorWork {
	defence := int8(1)
	if frame&64 != 0 {
		defence = 5
	}
	target := byte(6)
	if frame&8 != 0 {
		target = 0
	}
	snapshot := aiSnapshot{id: byte(id), stage: 8, enabled: 1, mode: 0, targetOffset: target, difficulty: 5, defence: defence, activity: 5}
	for actor, state := range t.actors {
		snapshot.poses[actor] = aiPose{x: state.Position, facing: state.Facing, action: state.Action, flags: state.Flags, held: state.Held, idle: state.Idle}
	}
	return &TitleActorWork{id: id, before: t.actors[id], snapshot: snapshot, round: t.Round, key: key}
}

// Advance obtains new random values only while the owner has preparation
// capacity. A private yield unwinds a local state copy; genuine errors propagate.
func (w *TitleActorWork) Advance(random func() (byte, bool)) (complete bool) {
	if w.done {
		return true
	}
	defer func() {
		if reason := recover(); reason != nil {
			if _, waiting := reason.(titleActorNeedsRandom); !waiting {
				panic(reason)
			}
			complete = false
		}
	}()
	index := 0
	next := func() byte {
		if index < len(w.draws) {
			value := w.draws[index]
			index++
			return value
		}
		value, available := random()
		if !available {
			panic(titleActorNeedsRandom{})
		}
		w.draws = append(w.draws, value)
		index++
		return value
	}
	state := w.before
	action, _ := originalDecision(w.snapshot, next)
	if action >= 0 {
		state.issueWithKey(byte(action), w.round, w.key, next)
	}
	state.advanceWithKey(8, 5, w.round, w.key, 0, 0, next)
	w.after, w.done = state, true
	return true
}

// CommitActor publishes the completed transaction for the next actor and painter.
func (t *TitleActors) CommitActor(work *TitleActorWork) (int, TitleActorState) {
	if !work.done {
		panic("cannot publish unfinished title actor")
	}
	t.actors[work.id] = work.after
	return work.id, work.after
}
