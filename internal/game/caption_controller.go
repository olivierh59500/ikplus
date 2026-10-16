package game

// CaptionContext supplies the source registers used by $7E32/$8194 without
// requiring a combat match to own an independent presentation sequence.
type CaptionContext struct {
	Positions, Facing, Frames [3]byte
	Mode, Counter, Elapsed    byte
	Variant                   byte
}

// CaptionController reuses the same verified caption state machine as Game.
// The owner supplies its live shared RNG and handles packed-screen restoration.
type CaptionController struct{ sourceCaption }

func NewCaptionController(state CaptionState) CaptionController {
	var controller CaptionController
	controller.load(state)
	return controller
}

func (c *CaptionController) Snapshot() CaptionState { return c.snapshot() }

// SetAlternate supplies the live $632 language word without preparing a new
// message. The source keeps the previous bubble width until its ID changes.
func (c *CaptionController) SetAlternate(alternate bool) {
	c.state.Alternate = alternate
}

func (ctx CaptionContext) original() captionContext {
	return captionContext{positions: ctx.Positions, facing: ctx.Facing, frames: ctx.Frames, stage: ctx.Mode, counter: ctx.Counter, elapsed: ctx.Elapsed, variant: ctx.Variant}
}

func (c *CaptionController) Prepare(message byte, ctx CaptionContext) {
	c.prepare(message, ctx.original())
}

func (c *CaptionController) Update(message byte, ctx CaptionContext, random func() byte) {
	c.update(message, ctx.original(), random)
}
