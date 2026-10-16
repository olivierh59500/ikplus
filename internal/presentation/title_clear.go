package presentation

import "github.com/olivierh59500/ikplus/internal/game"

// clearTitlePictures resets the title buffers and particle masks. The original
// picture banks contain 32000 visible bytes followed by 768 preserved bytes.
func (p *sceneryPlanes) clearTitlePictures() {
	p.state.Background, p.state.Drawing = game.PictureSecond, game.PictureFirst
	p.state.EraseStart, p.state.EraseCursor = game.RestorationCursor{Buffer: game.PictureFirst}, game.RestorationCursor{Buffer: game.PictureFirst}
	p.state.CollisionRead, p.state.CollisionReadMirror = game.CoverageFirst, game.CoverageFirstMirrored
	p.state.CollisionWrite, p.state.CollisionWriteMirror = game.CoverageFirst, game.CoverageFirstMirrored
	clear(p.state.Humans[:])
	clear(p.state.Points[:])
	p.state.CollisionEnabled = 0
	p.state.SecondaryCollisionEnabled = 0
	p.state.ScoresVisible = 1
	clear(p.pictures[1].pixels[:32000])
	clear(p.pictures[0].pixels[:32000])
	clear(p.titleAtlas)
}
