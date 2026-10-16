package game

// CaptionState is a value snapshot of the original speech/referee display.
// Anchor and Column use eight-pixel cells; RefereeX uses two-pixel units.
type CaptionState struct {
	Message, Previous, Width, Anchor, Column byte
	Visible, Shadow                          bool
	Counters, X, Y                           [8]byte
	Draw                                     [8]bool
	Budget, RefereeFrame, RefereeX           byte
	Alternate                                bool
	Order                                    [3]byte
	Amount                                   byte
}

type captionContext struct {
	positions, facing, frames [3]byte
	stage, counter, elapsed   byte
	variant                   byte
}

// sourceCaption retains $7F74/$825A's eight decorative sparkle slots. They
// consume the same shared random stream as fighters and bonus projectiles.
type sourceCaption struct {
	counters [8]byte
	x, y     [8]byte
	budget   byte
	state    CaptionState
}

func (g *Game) Caption() CaptionState { return g.caption.snapshot() }

func (c *sourceCaption) snapshot() CaptionState {
	s := c.state
	s.Counters, s.X, s.Y, s.Budget = c.counters, c.x, c.y, c.budget
	if !s.Visible {
		s.RefereeFrame, s.RefereeX, s.Draw = 0, 0, [8]bool{}
		s.Shadow = false
	}
	return s
}

func (c *sourceCaption) load(s CaptionState) {
	c.state, c.counters, c.x, c.y, c.budget = s, s.Counters, s.X, s.Y, s.Budget
}

func (c *sourceCaption) tick(random func() byte) {
	work := c.prepareSparkles()
	work.advance(c, func() (byte, bool) { return random(), true })
}

// place translates $7F44's occupied-cell map and largest-gap selection. Blue
// reserves the original edge regions; equal gaps keep the first candidate.
func (c *sourceCaption) place(ctx captionContext) {
	s := &c.state
	if ctx.stage == 6 {
		s.Anchor = 34
		c.column()
		return
	}
	for slot := 0; slot < 6; slot++ {
		c.counters[slot] = byte(slot & 3)
	}
	var left, right [3]byte
	for id := 2; id >= 0; id-- {
		left[id] = ctx.positions[id]>>2 - 4
		if int8(left[id]) < 0 {
			left[id] = 0
		}
		width := nativeGameplay.Animation.Widths[ctx.frames[id]]>>2 + 1
		if ctx.facing[id] != 0 {
			right[id] = left[id] + 10
			left[id] = right[id] - width
		} else {
			right[id] = left[id] + width
		}
	}
	if int8(right[2]) >= 29 {
		right[2] = 39
	}
	if int8(left[2]) < 9 {
		left[2] = 0
	}
	var occupied [256]byte
	for id := 0; id < 3; id++ {
		for cell := left[id]; ; {
			if int8(cell) < 40 {
				occupied[cell] = byte(id + 1)
			}
			cell++
			if int8(cell) >= int8(right[id]) {
				break
			}
		}
	}
	var run, largest, start byte
	for cell := byte(0); cell <= 40; cell++ {
		if cell == 40 || occupied[cell] != 0 {
			if run > largest {
				largest, start = run, cell-run
			}
			run = 0
		} else {
			run++
		}
	}
	if largest < 6 {
		for {
			if start != 0 {
				if occupied[start-1] >= 3 {
					start++
					continue
				}
				if occupied[start+largest] >= 3 {
					start--
				}
			}
			if int8(start) < 34 {
				break
			}
			start--
		}
		largest = 0
	}
	s.Anchor = start
	if largest != 0 {
		s.Anchor += (largest - 6) >> 1
	}
	c.column()
}

func (c *sourceCaption) column() {
	s := &c.state
	switch {
	case int8(s.Anchor) < 10:
		s.Column = s.Anchor
	case int8(s.Anchor) >= 27:
		s.Column = s.Anchor - s.Width + 4
	default:
		s.Column = s.Anchor + 3 - s.Width>>1
	}
}

// prepare follows the message-change branches of $7E32. A replacement keeps
// the referee's anchor and sparkle budget while adapting the bubble width.
func (c *sourceCaption) prepare(message byte, ctx captionContext) bool {
	s := &c.state
	s.Message = message
	if int8(message) < 0 {
		s.Previous, s.Visible = message, false
		return false
	}
	if message != s.Previous {
		s.Previous = message
		if message >= 40 {
			s.Visible = false
			return false
		}
		language := 0
		if s.Alternate {
			language = 1
		}
		s.Width = nativeGameplay.Caption.Widths[language][message] + 2
		if s.Visible {
			c.column()
		} else {
			c.place(ctx)
			c.budget = 40
		}
	}
	s.Visible = true
	return true
}

func (c *sourceCaption) update(message byte, ctx captionContext, random func() byte) {
	if c.beginUpdate(message, ctx) {
		c.tick(random)
	}
}

// beginUpdate prepares the referee independently of its subsequent sparkle
// work, whose random draws may continue on another native update.
func (c *sourceCaption) beginUpdate(message byte, ctx captionContext) bool {
	if !c.prepare(message, ctx) {
		return false
	}
	s := &c.state
	if message >= 40 {
		s.Width = 3
	}
	s.RefereeFrame = 70 + ctx.variant
	if ctx.elapsed < 10 && nativeGameplay.Caption.Speaking[message] != 0 {
		s.RefereeFrame = nativeGameplay.Caption.Referees[ctx.counter&3] + ctx.variant
	}
	s.RefereeX = s.Anchor<<2 + 20
	s.Shadow = ctx.stage != 8
	return true
}

func (g *Game) captionContext() captionContext {
	ctx := captionContext{stage: g.OriginalMode(), counter: g.displayCounter, elapsed: g.displayElapsed, variant: g.RefereeVariant()}
	for id, f := range g.Fighters {
		ctx.positions[id], ctx.frames[id] = byte(int(f.X/2)), byte(f.SpriteFrame)
		if f.Facing < 0 {
			ctx.facing[id] = 1
		}
	}
	return ctx
}

func (g *Game) startCaption(message byte) {
	g.caption.state.Alternate = g.GermanText()
	previous := g.caption.state.Previous
	g.caption.prepare(message, g.captionContext())
	if previous != g.caption.state.Previous {
		g.eraseCaptionPictures(g.caption.state.Anchor)
	}
}

func (g *Game) tickCaption() {
	g.caption.state.Alternate = g.GermanText()
	previous := g.caption.state.Previous
	if g.captionTraceEnabled {
		s := CaptionPreparationSnapshot{Frame: g.Frame, Mode: g.OriginalMode(), Counter: g.displayCounter, Message: g.caption.state.Message, Before: g.Caption()}
		g.caption.update(g.caption.state.Message, g.captionContext(), func() byte {
			before := g.random
			state := g.Caption()
			value := g.originalRandomByte()
			s.Random = append(s.Random, CaptionRandomRequest{g.Frame, g.OriginalVBL(), before, g.random, value, state})
			return value
		})
		s.After = g.Caption()
		s.CompletedFrame = g.Frame
		g.captionPreparations = append(g.captionPreparations, s)
	} else {
		g.caption.update(g.caption.state.Message, g.captionContext(), g.originalRandomByte)
	}
	if previous != g.caption.state.Previous {
		g.eraseCaptionPictures(g.caption.state.Anchor)
	}
}

func (g *Game) hideCaption() {
	g.startCaption(255)
}
