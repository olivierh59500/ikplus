package presentation

import "github.com/olivierh59500/ikplus/internal/game"

func presentationAmbient(p *sceneryPlanes) game.AmbientState         { return p.state.Ambient() }
func storePresentationAmbient(p *sceneryPlanes, a game.AmbientState) { p.state.SetAmbient(a) }

// sceneryVBL follows the same $18C42 update order as the arena: sample water
// masks, apply phase jitter, paint retained scenery, then select its next pose.
// The shared RNG runs during the logical tick and is never owned by a renderer.
func (h *highScoreSequence) sceneryVBL() {
	p := h.planes
	for i := 0; i < 4; i++ {
		h.ambient.SetForceWord(i, p.state.AmbientForces[i])
	}
	var phases [21]byte
	phases = p.state.ReflectionPhases
	if p.state.Frame&1 == 0 {
		for index := range phases {
			phases[index] = (phases[index] + 31) & 31
			if p.waterUpdates == 255 {
				choice := ((index & 3) + int((p.waterSlow+1)>>4)) & 15
				phases[index] = (phases[index] + p.artwork.Water.Jitter[choice]) & 31
			}
		}
	}
	before := h.ambient
	if before.Mode == 0 && p.state.BorderEnabled != 0 {
		// $19246 suppresses a new scenery selection during name entry.
		h.ambient.Counter++
		h.ambient.Visible, h.ambient.ScriptVisible = false, false
	} else {
		h.ambient.Tick(p.state.Frame, 4, phases, func() byte { return byte(h.randomWord()) })
	}
	bank := 0
	if p.state.Background == game.PictureSecond {
		bank = 1
	}
	p.tick(before, h.ambient, p.state.Frame, phases, bank, true)
	copy(p.state.ReflectionPhases[0:21], phases[:])
	p.state.WaterUpdates, p.state.WaterSlow = p.waterUpdates, p.waterSlow
	p.state.WaterChoice = p.waterSlow >> 4
	storePresentationAmbient(p, h.ambient)
}
