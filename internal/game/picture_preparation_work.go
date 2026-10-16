package game

// A fixed work scale keeps preparation independent of host rendering speed.
const picturePreparationQuota = 65536

// The fixed allowance includes the later ground-shadow setup. Keeping that
// tail separate identifies the earlier body rows without changing total work.
const (
	actorPreparationSetupWork = 1475
	actorGroundShadowWork     = 1000
)

// Ripples restore five rows and paint four every update. The measured baseline
// uses alignment five; sixteen long-word rotations account for other positions.
// Pose advances and the additional wrap step retain their separate work.
func ripplePreparationWork(a AmbientState) int {
	work := 1800 + (int(a.X&7)-5)*16*2*picturePreparationQuota/(227*313)
	if a.Counter&3 == 0 {
		work += 63
		if a.Frame == 0 {
			work += 19
		}
	}
	return work
}

// The authored RLE runs describe the work independently of GPU/host speed.
func authoredPictureRows(frame int) (rows, runs int) {
	if frame < 0 || frame >= 88 {
		return 0, 0
	}
	for _, run := range fighterArtwork.Fighters[frame].Runs {
		rows += len(run.Rows)
		runs++
	}
	return rows, runs
}

// actorRotationWords describes actual artwork operations, independent of the
// provisional duration estimate. A ground shadow rotates five words per row.
func actorRotationWords(frame int, shadow bool) int {
	if frame < 0 || frame >= len(fighterArtwork.Fighters) {
		return 0
	}
	rows, _ := authoredPictureRows(frame)
	words := rows * 3
	if shadow {
		words += 8 * 5
	}
	return words
}

func actorPreparationWork(frame, rawX int) int {
	rows, runs := authoredPictureRows(frame)
	work := actorPreparationSetupWork + rows*150 + rows*3/2 + runs*100 + rows*(rawX&7)*4
	if frame >= 36 && frame <= 47 {
		work += 1310
	}
	return work
}

func pictureRestorationGeometry(p PreparedPicture) (rows, runs int) {
	if !p.Valid {
		return 0, 0
	}
	for id, fighter := range p.Fighters {
		if p.Active >= 0 && p.Active != id {
			continue
		}
		height, spans := authoredPictureRows(fighter.SpriteFrame)
		rows += height + spans
		runs += spans
		VisitFighterEffects(fighter, func(effect FighterEffect) {
			if restored := spriteRectangleRows(effect.X, effect.Y, len(effect.Sprite.Rows), false); restored != 0 {
				rows += restored
				runs++
			}
		})
		if p.Mode == 7 && id == p.Active {
			if x, y, active := KickMarker(fighter); active {
				if restored := spriteRectangleRows(x+32, y, 1, true); restored != 0 {
					rows += restored
					runs++
				}
			}
		}
	}
	if p.ShieldVisible && p.Active >= 0 {
		f := p.Fighters[p.Active]
		boards := 1
		if f.ShieldType == 1 {
			boards = 2
		}
		for board := 0; board < boards; board++ {
			if height := shieldRectangleRows(int(f.ShieldX)+32+board*16, int(f.ShieldY)); height != 0 {
				rows += height
				runs++
			}
		}
	}
	if p.Caption.Visible {
		height, spans := authoredPictureRows(int(p.Caption.RefereeFrame))
		rows += height + spans
		runs += spans
		if p.CaptionText && p.Caption.Previous < 40 {
			rows += 8
			runs++
		}
		for _, drawn := range p.Caption.Draw {
			if drawn {
				rows += 6
				runs++
			}
		}
	}
	for _, blast := range p.Blasts {
		if blast.Active && blast.Stage == 5 {
			height, spans := authoredPictureRows(blast.BlastFrame)
			rows += height + spans
			runs += spans
		}
	}
	return rows, runs
}

// The packed shield painter queues only its visible rows. AE4A restores the
// saved height inclusively, adding one row; a rejected picture queues nothing.
func shieldRectangleRows(x, y int) int {
	return spriteRectangleRows(x, y, 23, true)
}

func spriteRectangleRows(x, y, height int, pixelX bool) int {
	if pixelX {
		x = (x & 511) >> 1
	} else {
		x &= 255
	}
	aligned := x & 248
	y &= 4095
	if aligned < 8 || aligned >= 176 || y >= Height {
		return 0
	}
	return min(height, Height-y) + 1
}

// Caption text interprets the original command stream only to count authored
// glyph rows. Names, double-height text and numeric substitutions retain their
// actual sizes instead of using a fixed duration for every message.
func captionTextPreparationWork(s CaptionState) int {
	return captionTextWorkWithRows(s, 105)
}

func captionTextWorkWithRows(s CaptionState, glyphRowWork int) int {
	if !s.Visible || s.Previous >= 40 {
		return 64
	}
	drawing := captionDrawingRows(s)
	return drawing.PanelRows*167 + drawing.GlyphRows*glyphRowWork + 2000
}

// Caption drawing follows authored panel and glyph rows. These geometric
// quantities stay separate from the provisional duration weights.
type captionDrawingGeometry struct{ PanelRows, GlyphRows int }

func captionDrawingRows(s CaptionState) captionDrawingGeometry {
	if !s.Visible || s.Previous >= 40 {
		return captionDrawingGeometry{}
	}
	language := 0
	if s.Alternate {
		language = 1
	}
	rows, height := 0, 5
	for _, ch := range nativeGameplay.Caption.Messages[language][s.Previous] {
		switch {
		case ch < 10:
			rows += int(ch) * height
		case ch < 13:
			rows += int(nativeGameplay.Caption.NameLengths[s.Order[ch-10]]) * height
		case ch >= 48 && ch < 95:
			rows += height
		case ch == 0x9a:
			height = 10
		case ch == 0x9d:
			rows += 2 * height
		case ch == 0x9e:
			height = 5
		case ch == 0x9b:
			panelRows := 0
			if s.Width > 2 {
				panelRows = int(s.Width) * 21
			}
			return captionDrawingGeometry{PanelRows: panelRows, GlyphRows: rows}
		}
	}
	panic("unterminated original caption command stream")
}
