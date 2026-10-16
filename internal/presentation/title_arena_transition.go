package presentation

import "github.com/olivierh59500/ikplus/internal/game"

// advanceTitleArenaTransition gives the original eight-update handoff its
// native phases: restore the scene, reset, prepare the fighters, then publish.
// Presentation music and the frame counter continue while controls are blanked.
func (a *App) advanceTitleArenaTransition() {
	c := a.presentation
	p, title := a.ambient.planes, c.title
	c.arenaElapsed++
	title.frame++
	p.state.Frame = title.frame
	if title.frame == 0 {
		p.state.FrameWraps++
	}
	p.publishDisplay()
	a.match.BeginPresentationUpdate()
	for index := range title.effects.hardware {
		title.effects.hardware[index] = p.artwork.Title.Black[index]
	}
	title.effects.paletteMode = paletteTitle
	switch c.arenaElapsed {
	case 4:
		a.match.DetachPresentation()
		p.state.ResetArena()
		p.state.Mode, p.state.BufferPhase = 0, 4
		p.state.Background = game.PictureSecond
		p.state.Drawing = game.PictureFirst
		for _, bank := range p.screens() {
			copy(bank.pixels[:32000], p.artwork.Background[:32000])
		}
		a.attachPresentation()
		a.match.BeginDemoPreparation()
	case 5:
		a.paintArenaResetPictures(a.match.DisplaySpeed())
	case 6:
		a.match.PrepareDemoFighters()
	case 7:
		p.state.Mode = 1
	case 8:
		p.state.BlackScreen = 0
		a.endPreparedDemo()
	}
	a.match.EndPresentationUpdate(true)
}
