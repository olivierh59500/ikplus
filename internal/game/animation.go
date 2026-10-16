package game

import (
	"encoding/json"
	"fmt"

	"github.com/olivierh59500/ikplus/assets"
)

// AnimationTicks is the original normal-speed animation cadence. Startup sets
// $60F=2, $612=0, and $65A6 waits until VBL counter $82E exceeds table $65A0[2]
// (4). The frame therefore advances once per five 50 Hz vertical blanks.
const AnimationTicks = 5

type sourceAnimation struct {
	Action int   `json:"action"`
	Frames []int `json:"frames"`
	Motion []int `json:"x_deltas"`
	Flags  []int `json:"flags"`
}

var sourceAnimations = loadAnimations()

func loadAnimations() [41]sourceAnimation {
	raw, err := assets.Read("original/animations.json")
	if err != nil {
		panic(fmt.Sprintf("read original animations: %v", err))
	}
	var decoded []sourceAnimation
	if err = json.Unmarshal(raw, &decoded); err != nil {
		panic(fmt.Sprintf("decode original animations: %v", err))
	}
	if len(decoded) != 41 {
		panic("original animation table must contain41 actions")
	}
	var result [41]sourceAnimation
	for _, sequence := range decoded {
		if sequence.Action < 0 || sequence.Action >= len(result) || len(sequence.Frames) == 0 || len(sequence.Frames) != len(sequence.Motion) || len(sequence.Frames) != len(sequence.Flags) {
			panic("invalid original animation sequence")
		}
		result[sequence.Action] = sequence
	}
	return result
}

func sourceAction(move Move) int {
	switch move {
	case Idle:
		return 0
	case Jump:
		return 1
	case FrontPunch:
		return 2
	case WalkForward:
		return 3
	case ShinKick:
		return 4
	case Sweep:
		return 5
	case CrouchPunch:
		return 6
	case WalkBackward:
		return 7
	case ReversePunch:
		return 8
	case FlyingKick:
		return 9
	case Headbutt:
		return 10
	case StomachKick:
		return 11
	case FaceKick:
		return 12
	case ReverseSweep:
		return 13
	case BackstepKick:
		return 14
	case Backflip:
		return 15
	case DoubleKick:
		return 16
	case Knockdown:
		return 18
	case Block:
		return 20
	case Rise:
		return 36
	default:
		return 0
	}
}

// MoveDuration returns one complete original sequence at normal PAL speed.
func MoveDuration(move Move) int {
	return len(sourceAnimations[sourceAction(move)].Frames) * AnimationTicks
}

func (g *Game) animationTicks() int {
	return int(g.waitThreshold()) + 1
}

// animationTick follows the wait counter relative to the current scene's
// entry. The free-running VBL byte used by RNG and bomb fuses is separate.
func (g *Game) animationTick() bool {
	return g.Frame >= g.animationOrigin && (g.Frame-g.animationOrigin)%uint64(g.animationTicks()) == 0
}

func (g *Game) moveDuration(move Move) int {
	return len(sourceAnimations[sourceAction(move)].Frames) * g.animationTicks()
}

func moveForSourceAction(action int) Move {
	for move := Idle; move <= DoubleKick; move++ {
		if sourceAction(move) == action {
			return move
		}
	}
	switch action {
	case 17, 18, 19, 21, 31, 38, 39:
		return Knockdown
	case 35, 36, 37:
		return Rise
	case 20:
		return Block
	case 26:
		return WalkForward
	case 27, 29, 30:
		return WalkBackward
	case 33:
		return Jump
	case 40:
		return FlyingKick
	default:
		return Idle
	}
}
