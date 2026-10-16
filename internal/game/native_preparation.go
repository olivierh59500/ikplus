package game

import "github.com/olivierh59500/ikplus/internal/audio"

// UseNativePlayback applies the shared desktop/mobile controller and picture
// publication policy after selecting a PAL profile, before starting gameplay.
func (g *Game) UseNativePlayback() {
	g.UseNativePreparation()
	g.UsePhysicalJoystickPins()
	g.UseDisplayLatch()
}

// UseNativePreparation selects the native bonus work queues for normal PAL
// playback. Declared historical frame profiles retain their reference cadence.
func (g *Game) UseNativePreparation() {
	if g.PALFrameClocks() == audio.Supplied313.FrameClocks() {
		g.UseNativeBallPreparation()
		g.UseNativeBombPreparation()
	}
}
