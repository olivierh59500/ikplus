package game

// EffectColor is COLOR1 copied from the original $624 word by the VBL driver.
func (g *Game) EffectColor() uint16 { return g.effectColorVisible }

// PictureVisible distinguishes a bomb already painted by the IRQ from a
// newly spawned active slot. Spawning follows the original drawing loop.
func (b Bomb) PictureVisible() bool { return b.drawn }

// BallPictureOperationKind distinguishes sprite erasure from ground painting.
type BallPictureOperationKind byte

const (
	BallPictureRestore BallPictureOperationKind = iota
	BallGroundDraw
)

// BallPictureOperation retains projectile erasures and ground ink on a screen.
// Ground shadows survive until that screen's next actor preparation clears it.
type BallPictureOperation struct {
	Frame uint64
	Ball  Ball
	Kind  BallPictureOperationKind
}

// BallActorErase records a projectile restoration between authored actor rows.
// Later actor rows can cover it again; earlier rows retain the erased pixels.
type BallActorErase struct {
	Ball Ball
	Rows int
}

func (g *Game) BallPictureOperations(bank int) []BallPictureOperation {
	if bank < 0 || bank >= len(g.ballRestores) {
		return nil
	}
	return g.ballRestores[bank]
}

func (g *Game) BallPictureVisible(bank int) bool {
	return g.ballPictureValid && bank == g.ballPictureBank
}

// BombPictureOperationKind separates opaque-mask and floor drawing operations.
type BombPictureOperationKind uint8

const (
	BombMaskRestore BombPictureOperationKind = iota
	BombGroundRestore
	BombGroundDraw
)

// BombPictureOperation records an original IRQ operation in a retained screen.
// Masks and floor rectangles can remove artwork between actor preparations.
type BombPictureOperation struct {
	Frame              uint64
	Slot               int
	Bank               int
	Bomb               Bomb
	PaintedAfterActors bool
	Kind               BombPictureOperationKind
}

// BombPictureOperations is read-only drawing history since this bank was prepared.
func (g *Game) BombPictureOperations(bank int) []BombPictureOperation {
	if bank < 0 || bank >= len(g.bombOperations) {
		return nil
	}
	return g.bombOperations[bank]
}

// CurrentBombOperations retains this IRQ's operations even if the main thread
// subsequently prepares that screen and resets its accumulated actor damage.
func (g *Game) CurrentBombOperations() []BombPictureOperation { return g.bombIRQOperations }

// PreparedPicture retains the actor/caption content prepared in one original
// screen bank. A swap does not make the other bank's newer picture visible.
type PreparedPicture struct {
	restoreRows, restoreRuns int
	Valid                    bool
	Frame                    uint64
	Mode                     byte
	Active                   int
	Priority                 byte
	Fighters                 [3]Fighter
	Caption                  CaptionState
	CaptionText              bool
	EraseCaption             bool
	EraseAnchor              byte
	ShieldVisible            bool
	HintActive               bool
	HintDraw                 bool
	HintDirect               bool
	HintOutline              uint16
	HintBacking              *HintPicture
	HintErasures             []PictureEraseState
	Hint                     HintState
	HintGerman               bool
	Blasts                   [10]Bomb
	Previous                 PictureEraseState
	BallActorErases          []BallActorErase
}

// HintPicture is an immutable font layer whose old outline bits can survive
// a later direct $6548 redraw with a different inherited $62E value.
type HintPicture struct {
	State   HintState
	Outline uint16
	German  bool
	Scores  [2]int
}

// PictureEraseState is the content queued for rectangle restoration before
// the next preparation in the same screen bank.
type PictureEraseState struct {
	Valid         bool
	Mode          byte
	Active        int
	Fighters      [3]Fighter
	Caption       CaptionState
	CaptionText   bool
	ShieldVisible bool
	Blasts        [10]Bomb
}

func (g *Game) Picture(bank int) PreparedPicture {
	if bank < 0 || bank >= len(g.pictures) {
		return PreparedPicture{}
	}
	return g.pictures[bank]
}

// UseDisplayLatch selects stable display publication for an interactive or
// recorded application, independently of a historical replay's update cadence.
func (g *Game) UseDisplayLatch() { g.displayLatch = true }

// StableDisplayEnabled reports whether presentation retains published images.
func (g *Game) StableDisplayEnabled() bool { return g.displayLatch }

// DisplayPicture retains the picture selected at this update's start. Reusing
// its bank for the following preparation must not expose a future pose and then
// revert to the previous pose on the next update. Controlled CPU-memory replays
// keep their explicitly separate instantaneous framebuffer profile.
func (g *Game) DisplayPicture(bank int) PreparedPicture {
	if g.displayPictureFrame != 0 && g.displayPictureFrame == g.Frame && bank == g.displayPictureBank {
		return g.displayPicture
	}
	return g.Picture(bank)
}

func (g *Game) rememberPicture() {
	_, bank := g.SceneryBuffers()
	old := g.pictures[bank]
	g.bombOperations[bank] = g.bombOperations[bank][:0]
	g.ballRestores[bank] = g.ballRestores[bank][:0]
	p := PreparedPicture{Valid: true, Frame: g.Frame, Mode: g.OriginalMode(), Active: -1, Fighters: g.Fighters, Caption: g.Caption(), EraseCaption: old.EraseCaption, EraseAnchor: old.EraseAnchor}
	hintPath := p.Mode == 1 || p.Mode == 3 || p.Mode == 4
	if hintPath && g.hint.Timer != 0 {
		// $6548 redraws every nonzero hint after shadows during the
		// initial judge loop, retaining the inherited outline word.
		p.HintDirect = g.judge.active && (g.judge.part == 0 || g.judge.part == 3 && g.judge.awardPrelude)
		p.HintActive, p.HintGerman = !p.HintDirect, g.germanText
		g.prepareHintText(p.HintDirect)
		p.Hint = g.hint
		p.HintOutline = g.hintOutline
		p.HintDraw = p.Hint.Timer != 0 && (p.HintDirect || int16(p.Hint.Timer) > 20)
		if p.HintDirect {
			p.HintBacking = old.HintBacking
			p.HintErasures = append([]PictureEraseState(nil), old.HintErasures...)
			if p.HintBacking == nil && old.HintDraw && old.HintOutline != p.HintOutline {
				p.HintBacking = &HintPicture{State: old.Hint, Outline: old.HintOutline, German: old.HintGerman, Scores: [2]int{old.Fighters[0].Score, old.Fighters[1].Score}}
			}
			if p.HintBacking != nil {
				p.HintErasures = append(p.HintErasures, PictureEraseState{Valid: old.Valid, Mode: old.Mode, Active: old.Active, Fighters: old.Fighters, Caption: old.Caption, CaptionText: old.CaptionText, ShieldVisible: old.ShieldVisible, Blasts: old.Blasts})
			}
		}
	}
	p.CaptionText = p.Caption.Visible
	p.Priority = g.drawPriority
	if old.Frame != g.Frame {
		p.Previous = PictureEraseState{
			Valid: old.Valid, Mode: old.Mode, Active: old.Active,
			Fighters: old.Fighters, Caption: old.Caption, CaptionText: old.CaptionText,
			ShieldVisible: old.ShieldVisible, Blasts: old.Blasts,
		}
	} else {
		p.Previous = old.Previous
	}
	if p.Mode == 5 || p.Mode == 7 {
		// Bonus routines draw the actor after $7E32 has cleared the strip.
		p.EraseCaption = false
		p.Active, p.ShieldVisible = g.Bonus.Fighter, g.Bonus.ShieldVisible
		if g.Phase == BonusEnd {
			p.Active = g.Bonus.resultActor
		}
		if g.Bonus.finalClearing {
			p.Active = 0
		}
		for slot, bomb := range g.Bonus.Bombs {
			if slot < len(p.Blasts) && bomb.Active && bomb.Stage == 5 {
				p.Blasts[slot] = bomb
			}
		}
	}
	p.restoreRows, p.restoreRuns = pictureRestorationGeometry(p)
	g.pictures[bank] = p
}

func (g *Game) beginPicture() {
	_, bank := g.SceneryBuffers()
	g.pictures[bank].EraseCaption = false
}

// $ADCE restores rows 117 through 199 in both banks after the final bonus wait.
// Logical shield poses can remain in RAM after their pictures disappear.
func (g *Game) clearArenaPictures() {
	for bank := range g.pictures {
		g.pictures[bank] = PreparedPicture{Frame: g.Frame}
		g.bombOperations[bank] = g.bombOperations[bank][:0]
		g.ballRestores[bank] = g.ballRestores[bank][:0]
	}
	g.ballPictureValid = false
	g.bombIRQOperations = g.bombIRQOperations[:0]
}

// $82FA restores the entire speech strip and its tail in both screens. The
// other bank's referee and sparkles survive until their own drawing/erasure.
func (g *Game) eraseCaptionPictures(anchor byte) {
	for bank := range g.pictures {
		p := &g.pictures[bank]
		p.CaptionText, p.EraseCaption, p.EraseAnchor = false, true, anchor
	}
}
