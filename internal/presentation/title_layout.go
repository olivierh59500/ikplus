package presentation

import "image"

type titleAction uint8

const (
	titleOnePlayer titleAction = iota
	titleTwoPlayers
	titleMoves
	titleScores
)

type titleButton struct {
	Action titleAction
	Bounds image.Rectangle
	Label  string
}

// titleActions keeps every menu control outside the original 320x200 scene.
// Drawing and pointer navigation share these rectangles on both platforms.
func titleActions(mobile bool) [4]titleButton {
	if mobile {
		return [4]titleButton{
			{titleOnePlayer, image.Rect(5, 52, 75, 84), "1 PLAYER"},
			{titleTwoPlayers, image.Rect(5, 92, 75, 124), "2 PLAYERS"},
			{titleMoves, image.Rect(88, 232, 236, 252), "MOVES [H]"},
			{titleScores, image.Rect(244, 232, 392, 252), "SCORES"},
		}
	}
	return [4]titleButton{
		{titleOnePlayer, image.Rect(82, 4, 194, 24), "1 PLAYER [F1]"},
		{titleTwoPlayers, image.Rect(198, 4, 314, 24), "2 PLAYERS [F2]"},
		{titleMoves, image.Rect(8, 232, 156, 252), "MOVES [H]"},
		{titleScores, image.Rect(164, 232, 312, 252), "SCORES"},
	}
}

func titleActionAt(mobile bool, x, y int) (titleAction, bool) {
	for _, button := range titleActions(mobile) {
		if image.Pt(x, y).In(button.Bounds) {
			return button.Action, true
		}
	}
	return 0, false
}
