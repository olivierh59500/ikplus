package game

// OriginalSpeed is the current $60F byte. Bomb turns temporarily replace it;
// the displayed label changes only when the original $6C22 routine runs.
func (g *Game) OriginalSpeed() byte {
	if g.presentationState != nil {
		return g.presentationState.Speed
	}
	if g.speedValue == 0 {
		return 2
	}
	return g.speedValue - 1
}

func (g *Game) DisplaySpeed() int {
	if g.displaySpeedValue == 0 {
		return 2
	}
	return int(g.displaySpeedValue - 1)
}

func (g *Game) OriginalFast() byte {
	if g.presentationState != nil {
		return g.presentationState.Fast
	}
	return g.sourceFast
}

func (g *Game) waitThreshold() byte {
	return fighterArtwork.Score.SpeedDelay[g.OriginalSpeed()+g.OriginalFast()]
}

// QueueSpeedKey accepts the decoded F6..F10 press/release bytes from $1ADE.
// The VBL poll applies source debounce and mode restrictions, not the UI tick.
func (g *Game) QueueSpeedKey(code byte) {
	if code&0x7f < 0x40 || code&0x7f > 0x44 {
		return
	}
	g.QueueSourceKey(code)
}

// QueueSourceKey accepts decoded keys from $1ADE. Unhandled keys still enter
// the original history, so they can interrupt a language or scenery command.
// Platform attack keys remain separate joystick input.
func (g *Game) QueueSourceKey(code byte) {
	if code&0x7f == 0 {
		return
	}
	raw := byte(0)
	found := false
	for index := 0; index < 128; index++ {
		if nativeGameplay.Input.KeyDecode[index] == code&0x7f {
			raw = byte(index) | code&0x80
			found = true
			break
		}
	}
	if !found {
		return
	}
	if g.speedKeyValid && raw == g.speedLastRaw {
		return
	}
	g.speedLastRaw, g.speedKeyValid = raw, true
	if code&0x80 == 0 {
		copy(g.speedKeys[1:], g.speedKeys[:3])
	}
	g.speedKeys[0] = code
	g.publishSpeedKeys()
}

func (g *Game) publishSpeedKeys() {
	if holder := g.presentationState; holder != nil {
		copy(holder.Keys[0:4], g.speedKeys[:])
		holder.KeyDelay = g.speedKeyDelay
		holder.LastRawKey = uint16(g.speedLastRaw)
	}
}

func (g *Game) pollSpeedKey() {
	mode, frame := g.OriginalMode(), g.OriginalVBL()
	if (mode == 5 || mode == 7) && frame&1 == 0 {
		return
	}
	if holder := g.presentationState; holder != nil {
		copy(g.speedKeys[:], holder.Keys[0:4])
		g.speedKeyDelay = holder.KeyDelay
	}
	g.speedKeyHistory = uint32(g.speedKeys[3])<<24 | uint32(g.speedKeys[2])<<16 | uint32(g.speedKeys[1])<<8 | uint32(g.speedKeys[0])
	if holder := g.presentationState; holder != nil {
		holder.KeyHistory = g.speedKeyHistory
	}
	wasFrozen := g.sourceFrozen
	keyBefore, delayBefore := g.speedKeys[0], g.speedKeyDelay
	ready := !wasFrozen && g.speedKeyHistory != 0xa19392ac && int8(delayBefore) >= 3
	if g.sourceFrozen {
		if g.speedKeys[0] != 0 {
			g.sourceFrozen = false
			g.speedKeys[0], g.speedKeyDelay = 0, 0
		}
	} else if g.speedKeyHistory == 0xa19392ac {
		g.sourceFrozen = true
		g.speedKeys[0], g.speedKeyDelay = 0, 0
	} else if int8(g.speedKeyDelay) < 3 {
		g.speedKeyDelay++
	} else {
		if g.speedKeyHistory == 0x94979426 {
			// $680C branches out of the old IRQ/main context through TRAP0.
			g.restartRequested = true
			if holder := g.presentationState; holder != nil {
				holder.KeyHistory = g.speedKeyHistory
			}
			return
		}
		if g.speedKeyHistory == 0xa19e9f14 {
			g.fastMusic()
		}
		if g.speedKeyHistory == 0xa197a614 {
			g.toggleLEDFilter()
			g.speedKeys[0], g.speedKeyDelay = 0, 0
		} else if g.applyAmbientKey(g.speedKeyHistory) {
			g.speedKeys[0], g.speedKeyDelay = 0, 0
		} else if g.applyDisplayKey(g.speedKeys[0], mode) {
			g.speedKeys[0], g.speedKeyDelay = 0, 0
		} else if g.applySoundKey(g.speedKeys[0]) {
			g.speedKeys[0], g.speedKeyDelay = 0, 0
		} else if mode == 6 && g.speedKeys[0] == 0x19 {
			g.pauseDemo.sound = !g.pauseDemo.sound
			g.speedKeys[0], g.speedKeyDelay = 0, 0
		} else if mode != 5 && mode != 7 && mode != 8 && g.speedKeys[0] >= 0x40 && g.speedKeys[0] <= 0x44 {
			g.speedValue = g.speedKeys[0] - 0x40 + 1
			g.displaySpeedValue = g.speedValue - g.OriginalFast()
			g.speedKeys[0], g.speedKeyDelay = 0, 0
			if g.presentationSpeedChanged != nil {
				g.presentationSpeedChanged()
			}
		} else if g.applyHintKey(g.speedKeys[0], mode) {
			if g.restartRequested {
				return
			}
			g.speedKeys[0], g.speedKeyDelay = 0, 0
		}
	}
	if !wasFrozen && g.sourceFrozen {
		g.pausePoint = g.waitPoint
	}
	g.pollSourceJoin(keyBefore != 0 && g.speedKeys[0] == 0, ready)
	if holder := g.presentationState; holder != nil {
		holder.FreezeLocation = g.pausePoint
		holder.Frozen = 0
		if g.sourceFrozen {
			holder.Frozen = 1
		}
		holder.Graphics, holder.Referee = g.graphicsIndex, g.refereeVariant
		if g.germanText {
			holder.German = 1
		} else {
			holder.German = 0
		}
		if g.alternateShield {
			holder.AlternateShield = 1
		} else {
			holder.AlternateShield = 0
		}
		holder.GroundShade = g.GroundShade()
		holder.PauseSound = 0
		if g.pauseDemo.sound {
			holder.PauseSound = 1
		}
		if g.MusicEnabled() {
			holder.Music = 1
		} else {
			holder.Music = 0
		}
		if g.EffectsEnabled() {
			holder.Effects = 1
		} else {
			holder.Effects = 0
		}
		holder.Volume = g.MasterVolume()
		holder.Filter = 0
		if g.filterLatch {
			holder.Filter = 1
		}
		holder.Speed = g.speedValueOrDefault()
		g.publishSpeedKeys()
		holder.KeyHistory = g.speedKeyHistory
	}
}

func (g *Game) speedValueOrDefault() byte {
	if g.speedValue == 0 {
		return 2
	}
	return g.speedValue - 1
}

// ValidSourceKey checks the recovered decoder table, including the original
// HELP key and letters used by authored hint/scenery words.
func ValidSourceKey(code byte) bool {
	if code == 0 {
		return true
	}
	if code&0x7f == 0 {
		return false
	}
	for index := 0; index < 128; index++ {
		if nativeGameplay.Input.KeyDecode[index] == code&0x7f {
			return true
		}
	}
	return false
}

func (g *Game) clearSpeedKeys() {
	g.speedKeys = [4]byte{}
	g.speedLastRaw, g.speedKeyHistory, g.speedKeyValid = 0, 0, false
	g.sourceFrozen = false
}
