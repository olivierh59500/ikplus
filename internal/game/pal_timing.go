package game

import (
	"fmt"

	"github.com/olivierh59500/ikplus/internal/audio"
)

// SetPALFrameProfile selects the physical field before gameplay or audio starts.
// Direct game construction retains the controlled 312-line reference profile.
func (g *Game) SetPALFrameProfile(profile audio.PALFrameProfile) error {
	if profile.FrameClocks() == 0 {
		return fmt.Errorf("unknown PAL frame profile %d", profile)
	}
	if g.Frame != 0 {
		return fmt.Errorf("select the PAL frame profile before gameplay starts")
	}
	g.palFrameProfile = profile
	return nil
}

func (g *Game) PALFrameClocks() uint64 { return g.palFrameProfile.FrameClocks() }
