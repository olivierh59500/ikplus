package game

// AttachPresentation shares the native scene state while the match retains
// ownership of audio and randomness.
func (g *Game) AttachPresentation(state *PresentationState, speedChanged func()) {
	holder := state
	g.presentationState = state
	g.presentationSpeedChanged = speedChanged
	g.speedValue = holder.Speed + 1
	g.sourceFast = holder.Fast
	g.speedKeyDelay = holder.KeyDelay
	copy(g.speedKeys[:], holder.Keys[0:4])
	g.speedLastRaw = byte(holder.LastRawKey)
	g.speedKeyValid = g.speedLastRaw != 0
	g.speedKeyHistory = holder.KeyHistory
	g.sourceFrozen = holder.Frozen != 0
	g.pausePoint = holder.FreezeLocation
	g.sourceBootMarker = holder.BootMarker
	g.filterLatch = holder.Filter != 0
	g.readHintState(state)
	g.pauseDemo.sound = holder.PauseSound != 0
	g.waitPoint = 0
	g.readDisplayOptions(state)
}

func (g *Game) PresentationRandomWord() uint16 {
	value, next := RandomWord(g.random, g.OriginalVBL(), g.randomSample)
	g.random = next
	g.publishPresentationRandom()
	return value
}

func (g *Game) publishPresentationRandom() {
	if g.presentationState == nil {
		return
	}
	g.presentationState.SetRandomState(g.random)
}

// BeginColdPresentationUpdate advances the first title update without a CPU
// startup context. The frontend supplies the display and title effects.
func (g *Game) BeginColdPresentationUpdate(publishDisplay func()) {
	if g.Frame != 0 || g.OriginalMode() != 8 {
		panic("cold title update requires an unstarted presentation")
	}
	g.beginPresentationUpdate(publishDisplay)
}

func (g *Game) EndColdPresentationUpdate() {
	// Construction is not a completed displayed frame. Keep its audio changes
	// for the first regular update so a newly attached renderer starts at zero.
	g.audioInFrame = false
	g.publishPresentationRandom()
}

// BeginPresentationUpdate advances music, controls and presentation timers.
// The elapsed update count continues across scene changes.
func (g *Game) BeginPresentationUpdate() {
	g.beginPresentationUpdate(nil)
}

func (g *Game) beginPresentationUpdate(publishDisplay func()) {
	g.Frame++
	g.beginSoundFrame()
	g.publishDisplayOptions()
	if publishDisplay != nil {
		// Publish the completed scene after music and before input polling.
		publishDisplay()
	}
	if g.presentationState != nil && g.presentationState.BlackScreen != 0 {
		// The black transition holds scene controls while audio continues.
		g.publishPresentationRandom()
		return
	}
	g.pollSpeedKey()
	if g.RestartRequested() || g.JoinRequested() != 0 {
		return
	}
	if g.OriginalMode() != 8 && !g.SourceFrozen() {
		g.tickSoundTimers()
	}
	g.publishPresentationRandom()
}

// PresentationDisplayWait applies queued audio before the next scene update.
// Pure presentation timer waits do not advance effect selection.
func (g *Game) PresentationDisplayWait() {
	if g.audioInFrame {
		g.updateSoundEffects()
		g.publishPresentationRandom()
	}
}

func (g *Game) EndPresentationUpdate(effects bool) {
	if !effects {
		g.audioEffectsDone = true
	}
	g.endSoundFrame()
	g.publishPresentationRandom()
}

// DetachPresentation preserves settings, scenery and session history before
// control returns to the arena.
func (g *Game) DetachPresentation() {
	if g.presentationState == nil {
		return
	}
	holder := g.presentationState
	frame := holder.Frame
	g.sourceFrameOffset = frame - byte(g.Frame)
	g.inputRepeatCounter, g.sourceVBLWraps = holder.InputRepeat, holder.FrameWraps
	g.speedValue = holder.Speed + 1
	g.sourceFast = holder.Fast
	copy(g.speedKeys[:], holder.Keys[0:4])
	g.speedKeyDelay = holder.KeyDelay
	g.sourceFrozen = holder.Frozen != 0
	g.pausePoint = holder.FreezeLocation
	g.sourceBootMarker = holder.BootMarker
	g.filterLatch = holder.Filter != 0
	g.readHintState(holder)
	g.pauseDemo.sound = holder.PauseSound != 0
	g.readDisplayOptions(holder)
	g.ambient = holder.Ambient()
	g.ambient.Visible, g.ambient.ScriptVisible = false, false
	copy(g.reflectionPhases[:], holder.ReflectionPhases[0:21])
	g.reflectionUpdates, g.reflectionSlow = holder.WaterUpdates, holder.WaterSlow
	g.ambientReady, g.reflectionReady = true, true
	g.retainPresentationState(holder)
	g.presentationState = nil
	g.presentationSpeedChanged = nil
}
