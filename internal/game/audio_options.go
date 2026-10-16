package game

// FilterLatch is the keyboard command's $630 latch. Music commands can change
// the actual CIA filter bit without changing this software toggle.
func (g *Game) FilterLatch() bool { return g.filterLatch }

func (g *Game) LEDFilterEnabled() bool {
	if g.audioEngine != nil {
		return g.audioEngine.LEDFilter()
	}
	return g.ledFilterEnabled
}

func (g *Game) toggleLEDFilter() {
	g.filterLatch = !g.filterLatch
	g.ledFilterEnabled = g.filterLatch
	if g.audioEngine != nil {
		g.audioEngine.SetLEDFilter(g.filterLatch)
	}
}

func (g *Game) MusicEnabled() bool { return g.audioEngine != nil && !g.musicDisabled }

func (g *Game) EffectsEnabled() bool { return g.audioControl != nil && !g.effectsDisabled }

func (g *Game) MasterVolume() uint16 {
	if g.audioEngine != nil {
		return g.audioEngine.MasterVolume()
	}
	if g.masterVolumeValue == 0 {
		return nativeGameplay.Defaults.MasterVolume
	}
	return g.masterVolumeValue - 1
}

func (g *Game) setMasterVolume(volume uint16) {
	g.masterVolumeValue = volume + 1
	if g.audioEngine != nil {
		g.audioEngine.SetMasterVolume(volume)
	}
}

// QueueSoundKey accepts the source F3/F4 and main/keypad +/- controls.
func (g *Game) QueueSoundKey(code byte) {
	switch code & 0x7f {
	case 0x3d, 0x3e, 0x0c, 0x0d, 0x4a, 0x4e:
		g.QueueSourceKey(code)
	}
}

func (g *Game) applySoundKey(key byte) bool {
	switch key {
	case 0x3d:
		if g.audioEngine != nil {
			if g.musicDisabled {
				g.audioEngine.StartMusic()
			} else {
				g.audioEngine.StopMusic()
			}
		}
		g.musicDisabled = !g.musicDisabled
		if !g.musicDisabled {
			g.setMasterVolume(15)
		}
		return true
	case 0x3e:
		g.effectsDisabled = !g.effectsDisabled
		if g.audioEngine != nil {
			g.audioEngine.EffectsEnabled = !g.effectsDisabled
		}
		return true
	case 0x0d, 0x4e:
		if g.MasterVolume() != 15 {
			g.setMasterVolume(g.MasterVolume() + 1)
			return true
		}
	case 0x0c, 0x4a:
		if g.MasterVolume() != 0 {
			g.setMasterVolume(g.MasterVolume() - 1)
			return true
		}
	}
	return false
}
