package presentation

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/ikplus/internal/game"
)

const (
	scoreReturnNone = iota
	scoreReturnPublication
	scoreReturnTitle
)

type presentationCycle struct {
	flow             *highScoreFlow
	title            *titleSequence
	image            *ebiten.Image
	rgba             []byte
	returningToArena bool
	scoreReturnPhase int
	arenaElapsed     int
	humanJoin        *humanJoinTransition
}

func initializeScoreHistory(p *sceneryPlanes) {
	for index := 0; index < 51; index++ {
		p.state.NameFirst[index] = p.artwork.Score.InitialNames[index][0] - 0x30
		p.state.NameSecond[index] = p.artwork.Score.InitialNames[index][1] - 0x30
		p.state.NameThird[index] = p.artwork.Score.InitialNames[index][2] - 0x30
		p.state.ScoreCandidate[index], p.state.ScoreHigh[index] = 0, 0
		value := (100 - index) % 100
		p.state.ScoreLow[index] = byte(value/10*16 + value%10)
	}
	p.state.ScoreHigh[0], p.state.ScoreLow[0] = 1, 0
	p.state.BootMarker = 0x01010202
}

func (a *App) beginPresentation() {
	a.beginPresentationMode(false)
}

// openScores enters the original display-only A110 branch. The ranked table
// belongs to this running session, independently of the legacy local archive.
func (a *App) openScores() {
	if a.presentation != nil {
		a.endPresentation(true)
	}
	a.beginPresentationMode(true)
	a.view, a.paused = scoresView, false
	a.CancelInput()
}

func (a *App) materializeArenaPictures() [2]game.PictureEraseState {
	p := a.ambient.planes
	var history [2]game.PictureEraseState
	visible := a.ambient.videoBank
	for bank, base := range p.screens() {
		a.ambient.videoBank = bank
		a.scene.compose(a.match, &a.ambient, a.currentHUDState())
		copy(base.pixels[:32000], p.working.pixels[:32000])
		picture := a.match.Picture(bank)
		history[bank] = game.PictureEraseState{Valid: picture.Valid, Mode: picture.Mode, Active: picture.Active, Fighters: picture.Fighters, Caption: picture.Caption, CaptionText: picture.CaptionText, ShieldVisible: picture.ShieldVisible, Blasts: picture.Blasts}
	}
	a.ambient.videoBank = visible
	return history
}

func (a *App) beginPresentationMode(displayOnly bool) {
	p := a.ambient.planes
	history := a.materializeArenaPictures()
	a.match.CopyPresentationState(&p.state)
	if displayOnly {
		p.state.Mode, p.state.CandidateOwner = 4, 0x80
	}
	clock := newPresentationClock(p)
	for id, active := range a.match.OwnerIndicators() {
		clock.hand(id, active)
	}
	a.attachPresentation()
	a.presentation = &presentationCycle{rgba: make([]byte, 320*200*4)}
	if p.state.Mode == 8 {
		a.beginNativeTitle()
	} else {
		a.presentation.flow = newHighScoreFlowWithHistory(p, &history)
	}
}

// prepareTitleArena translates the no-human $623C call made by the mode-four
// dispatcher after $A110. This call retains mode eight and the RNG counter.
func (a *App) prepareTitleArena() {
	p := a.ambient.planes
	h := newPresentationClock(p)
	state := hudState{}
	bcd := func(value byte) int { return int(value>>4)*10 + int(value&15) }
	for id := range state.Scores {
		state.Scores[id] = bcd(p.state.ScoreDigits[id*3])*10000 + bcd(p.state.ScoreDigits[1+(id*3)])*100 + bcd(p.state.ScoreDigits[2+(id*3)])
	}
	for _, draw := range hudLayout(state)[:36] {
		for row := 0; row < 8; row++ {
			mask := p.artwork.HUDFont[draw.Glyph][row]
			for plane := 0; plane < 4; plane++ {
				value := byte(0)
				if draw.Color&(1<<plane) != 0 {
					value = mask
				}
				for _, bank := range p.screens() {
					bank.pixels[(draw.Y+row)*160+draw.X/8+plane*40] = value
				}
			}
		}
	}
	clear(p.state.RoundScoreHigh[:])
	clear(p.state.RoundScoreLow[:])
	clear(p.state.Points[0:3])
	clear(p.state.FighterActivity[0:3])
	clear(p.state.SoundDown[0:3])
	clear(p.state.ScoreDigits[0:9])
	p.state.Round, p.state.RoundCycle = 1, 1
	p.state.Difficulty++
	if p.state.Difficulty >= 4 {
		p.state.Difficulty = 2
	}
	p.state.Subsecond, p.state.Seconds = 25, 0x30
	p.state.HumanCount, p.state.HumanMask, p.state.DrawPriority = 0, 0, 0
	p.state.OwnerFlash[0], p.state.OwnerFlash[1] = 30, 30
	p.state.Defence, p.state.Activity = p.state.Difficulty, p.state.Difficulty
	p.state.TargetPolicy = p.artwork.ArenaPolicy.Target
	if value := p.artwork.ArenaPolicy.Defence; value != 0 {
		p.state.Defence = value
	}
	if value := p.artwork.ArenaPolicy.Activity; value != 0 {
		p.state.Activity = value
	}
	for id := 2; id >= 0; id-- {
		p.state.Actors[id] = game.TitleActorState{Threat: p.state.Actors[id].Threat}
		p.state.Stun[id], p.state.Recovery[id], p.state.FighterActivity[id], p.state.Rest[id], p.state.DroolFrame[id], p.state.Puff[id] = 0, 0, 0, 0, 0, 0
		p.state.Actors[id].Facing = []byte{0, 1, 1}[id]
		p.state.Actors[id].Position = []byte{36, 80, 120}[id] + byte(h.randomWord())&3
		current := p.artwork.ArenaPolicy.Idle
		p.state.Actors[id].Action, p.state.Actors[id].Idle = 28, 28
		p.state.Actors[id].Current, p.state.Actors[id].Next, p.state.Actors[id].Fresh = current, current, 1

	}
	p.state.Message, p.state.PreviousMessage, p.state.CaptionVisible, p.state.Keys[0], p.state.LastActor = 255, 255, 0, 0, 255
}

func (a *App) beginNativeTitle() {
	a.prepareTitleArena()
	a.constructNativeTitle()
	// A110 leaves the arena list active until the first title IRQ installs
	// E74. Its final screen remains visible through this entry boundary.
	a.presentation.title.effects.paletteMode = paletteArena
}

func packedPresentationInput(input game.Input) byte {
	value := byte(15)
	if input.Fire {
		value |= 16
	}
	if input.X < 0 {
		value &^= 4
	} else if input.X > 0 {
		value &^= 8
	}
	if input.Y < 0 {
		value &^= 1
	} else if input.Y > 0 {
		value &^= 2
	}
	return value
}

func (a *App) tickPresentation(input [2]game.Input) {
	c := a.presentation
	a.match.SetSourceJoystick(input)
	if c.humanJoin != nil {
		a.advanceHumanJoin()
		return
	}
	if c.returningToArena {
		a.advanceTitleArenaTransition()
		return
	}
	if c.scoreReturnPhase == scoreReturnPublication {
		// The score controller returns after one final published update. Keep
		// its cleared retained picture visible through that boundary before
		// the title constructor changes the arena header and actor state.
		c.flow.clock.vbl()
		if a.match.RestartRequested() {
			a.warmRestart()
			return
		}
		if players := a.match.JoinRequested(); players != 0 {
			a.sourceJoin(players)
			return
		}
		a.match.EndPresentationUpdate(true)
		c.scoreReturnPhase = scoreReturnTitle
		return
	}
	if c.scoreReturnPhase == scoreReturnTitle {
		c.scoreReturnPhase = scoreReturnNone
		a.beginNativeTitle()
	}
	if io := a.ambient.planes.presentation; io != nil {
		io.joinRequested = a.match.JoinRequested
	}
	if c.flow != nil {
		key := byte(0)
		if c.flow.part == scoreDemo {
			if input[0].Y < 0 {
				key = 0x48
			} else if input[0].Y > 0 {
				key = 0x50
			}
		}
		if key != 0 || a.scrollKey != 0 {
			a.ambient.planes.state.Keys[0] = key
		}
		a.scrollKey = key
		c.flow.Step([2]byte{packedPresentationInput(input[0]), packedPresentationInput(input[1])})
		if a.match.RestartRequested() {
			a.warmRestart()
			return
		}
		if players := a.match.JoinRequested(); players != 0 {
			a.sourceJoin(players)
			return
		}
		a.match.EndPresentationUpdate(true)
		if c.flow.ended {
			if c.flow.clock.preparation != nil {
				c.scoreReturnPhase = scoreReturnPublication
			} else {
				a.beginNativeTitle()
			}
		}
		return
	}
	c.title.Step()
	if a.match.RestartRequested() {
		a.warmRestart()
		return
	}
	if players := a.match.JoinRequested(); players != 0 {
		a.sourceJoin(players)
		return
	}
	if c.title.ended {
		// $26D0/$77AA resets arena audio before this VBL's Copper work and
		// journal finish. The mixer consumes those writes with the same frame.
		if c.title.paced {
			c.returningToArena = true
			a.ambient.planes.state.BlackScreen = 1
		} else {
			a.endPresentation(true)
		}
	}
	a.match.EndPresentationUpdate(c.title.effects.paletteMode != paletteRainbow)
}

func (a *App) endPresentation(demo bool) {
	a.endPresentationMode(demo, false)
}

func (a *App) endPreparedDemo() {
	a.endPresentationMode(true, true)
}

func (a *App) endPresentationMode(demo, prepared bool) {
	if a.presentation == nil {
		return
	}
	p := a.ambient.planes
	title := a.presentation.title
	if title != nil {
		// Retain the title's CPU-written palette snapshot across programs;
		// its next VBL publishes the next direct color writes.
		a.titleColors = title.effects.hardware
	}
	a.match.DetachPresentation()
	// $77C6 clears this entire arena/title control region while retaining
	// the score table at $98E, options below $650 and code-section RNG words.
	p.state.ResetArena()
	p.presentation = nil
	p.stableDisplay = false
	p.publishedScoreFrame = nil
	a.presentation = nil
	for _, bank := range p.screens() {
		copy(bank.pixels[:32000], p.artwork.Background[:32000])
	}
	a.ambient.AmbientState = a.match.Ambient()
	a.ambient.lastFrame = a.match.Frame
	if demo {
		if prepared {
			a.match.FinishDemoPreparation()
		} else {
			a.match.ReturnToDemo()
		}
		a.view, a.paused = titleView, false
		if title != nil {
			// $26D0 restores the arena and its header before the current
			// IRQ's title palette/Copper are replaced by the next VBL.
			a.paintArenaResetPictures(a.match.DisplaySpeed())
			a.handoffPixels = append(a.handoffPixels[:0], title.pixels()...)
		}
	}
}

func (a *App) enteringName() bool {
	return a.presentation != nil && a.presentation.flow != nil && a.presentation.flow.part == scoreName
}

func (c *presentationCycle) pixels(p *sceneryPlanes) []byte {
	if c.humanJoin != nil {
		return c.humanJoin.picture.pixels()
	}
	if c.title != nil {
		return c.title.pixels()
	}
	if p.stableDisplay && p.publishedScoreFrame != nil {
		return p.publishedScoreFrame
	}
	return p.scorePixels(c.rgba)
}

// publishScoreFrame retains the complete visible image before the next actor
// preparation reuses its surface. Drawing cannot mutate this publication.
func (p *sceneryPlanes) publishScoreFrame() {
	if !p.stableDisplay {
		return
	}
	if p.publishedScoreFrame == nil {
		p.publishedScoreFrame = make([]byte, 320*200*4)
	}
	p.scorePixels(p.publishedScoreFrame)
}

func (p *sceneryPlanes) scorePixels(out []byte) []byte {
	bank := p.pictureBuffer(p.state.Displayed)
	for y := 0; y < 200; y++ {
		palette := p.arenaPalette(y)
		for x := 0; x < 320; x++ {
			index := byte(0)
			for plane := 0; plane < 4; plane++ {
				index |= (bank.pixels[y*160+plane*40+x/8] >> uint(7-x%8) & 1) << plane
			}
			color, at := palette[index], (y*320+x)*4
			out[at], out[at+1], out[at+2], out[at+3] = byte(color>>8&15)*17, byte(color>>4&15)*17, byte(color&15)*17, 255
		}
	}
	return out
}
