package game

// TitlePalette retains the original sixteen OCS colors and $95E fade divider.
// Hardware colors are copied before a title handler updates this working bank.
type TitlePalette struct {
	Colors  [16]uint16
	Divider uint16
}

// Fade follows $5720. Each eligible update moves the three color nibbles one
// step toward the target. Completion is detected on the following eligible
// update, when no channel changes, as in the original instruction sequence.
func (p *TitlePalette) Fade(target [16]uint16, interval uint16) bool {
	p.Divider = (p.Divider + 1) & 15
	if p.Divider < interval {
		return false
	}
	p.Divider = 0
	changed := false
	for i, current := range p.Colors {
		for shift := uint(0); shift < 12; shift += 4 {
			mask := uint16(15) << shift
			from, to := current&mask, target[i]&mask
			if from < to {
				current += uint16(1) << shift
				changed = true
			} else if from > to {
				current -= uint16(1) << shift
				changed = true
			}
		}
		p.Colors[i] = current
	}
	return !changed
}
