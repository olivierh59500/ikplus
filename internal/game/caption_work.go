package game

// captionSparkleWork retains a partially prepared $825A sparkle pass. X and Y
// use separate random draws, so a preparation boundary may fall between them.
// Completed slots are never replayed or charged to the random stream again.
type captionSparkleWork struct {
	slot  int
	phase byte
}

// Allocation retains two independent random requests. Its allowance includes
// the source-observed selection work around each request.
const captionSparkleRandomWork = 220

// A sparkle paints five rows across four packed color planes. Its horizontal
// alignment changes all twenty rotations rather than adding a fixed slot cost.
func (c *sourceCaption) sparkleSlotWork(slot int, draw bool) int {
	if !draw {
		return 49
	}
	x := c.state.Anchor*4 + 14 + c.x[slot]
	return 1657 + captionSparkleRotationWork(x)
}

func captionSparkleRotationWork(rawX byte) int {
	return 20 * int(rawX&7) * 2 * picturePreparationQuota / (227 * 313)
}

func (c *sourceCaption) prepareSparkles() captionSparkleWork {
	c.state.Draw = [8]bool{}
	return captionSparkleWork{slot: 7}
}

func (w *captionSparkleWork) advance(c *sourceCaption, random func() (byte, bool)) bool {
	return w.advancePainting(c, random, func(int) bool { return true })
}

// advancePainting can also yield during a slot's authored image work. Keeping
// that phase prevents the following slot from sampling randomness too early.
func (w *captionSparkleWork) advancePainting(c *sourceCaption, random func() (byte, bool), paint func(int) bool) bool {
	return w.advanceSlots(c, random, func(slot int, draw bool) bool {
		return !draw || paint(slot)
	})
}

// Empty slots retain their place in the pass too. A preparation can yield
// there without repeating a counter decrement, allocation or random request.
func (w *captionSparkleWork) advanceSlots(c *sourceCaption, random func() (byte, bool), visit func(int, bool) bool) bool {
	for w.slot >= 0 {
		slot := w.slot
		if w.phase == 0 {
			if c.counters[slot] != 0 {
				c.counters[slot]--
				w.phase = 3
			} else if c.budget == 0 {
				w.phase = 4
			} else {
				c.budget--
				w.phase = 1
			}
		}
		if w.phase < 3 {
			value, available := random()
			if !available {
				return false
			}
			if w.phase == 1 {
				c.x[slot] = nativeGameplay.Caption.SparkleX[value&31]
				w.phase = 2
				continue
			}
			c.y[slot] = nativeGameplay.Caption.SparkleY[value&31]
			c.counters[slot] = 3
			w.phase = 3
		}
		draw := w.phase != 4
		if !visit(slot, draw) {
			return false
		}
		if draw {
			c.state.Draw[slot] = true
		}
		w.phase = 0
		w.slot--
	}
	return true
}
