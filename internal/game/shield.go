package game

import (
	"encoding/json"

	"github.com/olivierh59500/ikplus/assets"
)

type shieldSourceState struct {
	State    int `json:"state"`
	Type     int `json:"shield_type"`
	ShieldX  int `json:"shield_x_byte"`
	FighterX int `json:"fighter_x_byte"`
	Y        int `json:"shield_y"`
	Facing   int `json:"facing"`
	Frame    int `json:"frame"`
}

type shieldSourceTable struct {
	Initial     int                 `json:"initial_state"`
	Directions  []int               `json:"joystick_directions"`
	Transitions [][]int             `json:"transitions"`
	States      []shieldSourceState `json:"states"`
}

var shieldControls = loadShieldControls()

func loadShieldControls() shieldSourceTable {
	raw, err := assets.Read("original/bonus-controls.json")
	if err != nil {
		panic(err)
	}
	var result shieldSourceTable
	if err = json.Unmarshal(raw, &result); err != nil {
		panic(err)
	}
	if len(result.States) != 7 || len(result.Directions) != 16 || len(result.Transitions) != 7 {
		panic("invalid original shield tables")
	}
	return result
}

// joystickNibble reproduces $1B8E's active-low direction bits, ordered
// up/down/left/right. Fire is separate and does not affect the shield state.
func joystickNibble(input Input) int {
	i := normalize(input)
	n := 15
	if i.Y < 0 {
		n &^= 1
	}
	if i.Y > 0 {
		n &^= 2
	}
	if i.X < 0 {
		n &^= 4
	}
	if i.X > 0 {
		n &^= 8
	}
	return n
}

func (g *Game) setShieldState(id, state int) {
	f := &g.Fighters[id]
	s := shieldControls.States[state]
	f.ShieldState, f.ShieldType, f.SpriteFrame = state, s.Type, s.Frame
	f.ShieldAlternate = g.AlternateShield()
	f.X = float64(s.FighterX&248) * 2
	f.ShieldX = float64(s.ShieldX&248)*2 - 32
	f.ShieldY = float64(s.Y)
	f.Facing = 1
	if s.Facing != 0 {
		f.Facing = -1
	}
	f.ShieldSide = f.Facing
	// C3EC publishes these same bytes in the shared controller/actor arrays.
	f.computerAnimation.Position = byte(int(f.X / 2))
	f.computerAnimation.Facing = byte(s.Facing)
	f.computerAnimation.Frame = byte(s.Frame)
	f.computerAnimation.Pending = byte(state)
}

func (g *Game) tickShield(id int, input Input) {
	f := &g.Fighters[id]
	state := NextShieldState(f.ShieldState, input)
	g.setShieldState(id, state)
}

// NextShieldState is the original public-control transition, independent of
// match state. It also permits tactical tools to learn the authored controls.
func NextShieldState(current int, input Input) int {
	direction := shieldControls.Directions[joystickNibble(input)]
	state := shieldControls.Transitions[current][direction]
	if state&128 != 0 {
		// $C438 selects the retained height on the opposite side.
		facing := shieldControls.States[current].Facing
		state = (state & 3) + ((facing ^ 1) << 2)
	}
	return state
}
