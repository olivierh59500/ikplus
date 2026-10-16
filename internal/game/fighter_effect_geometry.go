package game

import "github.com/olivierh59500/ikplus/internal/graphics"

var fighterArtwork = graphics.MustLoad()

// FighterEffect selects a native sprite cell at the sampled actor coordinates.
type FighterEffect struct {
	X, Y   int
	Sprite graphics.Sprite
}

// VisitFighterEffects supplies the same native cells for drawing and erasure.
func VisitFighterEffects(f Fighter, visit func(FighterEffect)) {
	if f.SpriteFrame < 36 || f.SpriteFrame > 47 {
		return
	}
	if f.Stun != 0 {
		for column := 0; column < 2; column++ {
			cell := fighterArtwork.Stars[f.StarColor][f.StarFrame][column]
			visit(FighterEffect{int(f.StarX) + column*8, int(f.StarY), fighterArtwork.Sprites[cell]})
		}
	}
	if f.RecoveryWait() != 0 {
		pose := -1
		switch f.SpriteFrame {
		case 39:
			pose = 0
		case 43:
			pose = 1
		case 47:
			pose = 2
		}
		if pose >= 0 {
			if f.Facing < 0 {
				pose += 3
			}
			x := int(f.X/2) + fighterArtwork.DroolPosition[pose][0]
			y := fighterArtwork.DroolPosition[pose][1]
			for column := 0; column < 2; column++ {
				cell := fighterArtwork.Drool[pose][f.DroolFrame][column]
				visit(FighterEffect{x + column*8, y, fighterArtwork.Sprites[cell]})
			}
		}
	}
	if f.Puff && f.SpriteFrame >= 40 && f.SpriteFrame < 43 {
		for column := 0; column < 2; column++ {
			cell := fighterArtwork.Puffs[f.SpriteFrame-40][column]
			visit(FighterEffect{int(f.PuffX) + column*8, 131, fighterArtwork.Sprites[cell]})
		}
	}
}
