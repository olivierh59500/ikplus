package presentation

import "github.com/olivierh59500/ikplus/internal/game"

// presentationIO joins the native programs to the existing game/audio owner.
// Isolated source comparisons leave these hooks nil and own their RNG locally.
type presentationIO struct {
	pacedScores      bool
	randomWord       func() uint16
	beginUpdate      func()
	beforeWait       func()
	mainWait         func(game.WaitPoint)
	restartRequested func() bool
	joinRequested    func() int
}

func (p *sceneryPlanes) restartRequested() bool {
	// TITL and $939 both replace the old program before its IRQ can resume.
	io := p.presentation
	return io != nil && (io.restartRequested != nil && io.restartRequested() || io.joinRequested != nil && io.joinRequested() != 0)
}

func (p *sceneryPlanes) setMainWait(wait game.WaitPoint) {
	if io := p.presentation; io != nil && io.mainWait != nil {
		io.mainWait(wait)
	}
}

// publishDisplay follows $FD2/$EC4 before the mode-specific VBL work.
func (p *sceneryPlanes) publishDisplay() {
	p.state.Displayed = p.state.Background
	p.state.GraphicsColor = p.artwork.GraphicsColors[p.state.Graphics/2]
	p.state.VisibleWhiteBelt, p.state.VisibleRedBelt, p.state.VisibleBlueBelt = p.state.BeltColors[0], p.state.BeltColors[1], p.state.BeltColors[2]
	p.state.VisibleEffectColor, p.state.VisibleGroundShade = p.state.EffectColor, p.state.GroundShade
}

// paintSpeed follows $6C22's two sixteen-pixel cells in both retained headers.
func (p *sceneryPlanes) paintSpeed(speed int) {
	for _, bank := range p.screens() {
		for column, cell := range p.artwork.Speeds[speed] {
			p.spriteAt(bank, 160+column*8, 10, 5, p.artwork.Sprites[cell], false, false, false)
		}
	}
}
