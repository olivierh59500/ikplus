package game

// CaptionRandomRequest observes a game-owned random call during preparation.
// It does not supply timing, values or expected state to the game.
type CaptionRandomRequest struct {
	Frame         uint64
	VBL           byte
	Before, After uint64
	Value         byte
	State         CaptionState
}

type CaptionPreparationSnapshot struct {
	Frame                  uint64
	CompletedFrame         uint64
	Mode, Counter, Message byte
	Before, After          CaptionState
	Random                 []CaptionRandomRequest
	Stages                 []CaptionWorkStage
}

type CaptionWorkStage struct {
	Stage    byte
	Frame    uint64
	Progress int
}

// TraceCaptionPreparation enables bounded caller-owned development recording.
// The caller must collect each update before the next one clears its journal.
func (g *Game) TraceCaptionPreparation() { g.captionTraceEnabled = true }

func (g *Game) CaptionPreparations() []CaptionPreparationSnapshot { return g.captionPreparations }
