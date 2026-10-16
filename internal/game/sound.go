package game

import "github.com/olivierh59500/ikplus/internal/audio"

// AudioPreparationSnapshot records native sample-service inputs for development
// comparison. Playback and gameplay never consult this optional journal.
type AudioPreparationSnapshot struct {
	Frame         uint64
	At            uint64
	EffectEnd     uint64
	Phase         int
	Mode          byte
	Before, After audio.ControlState
}

// TraceAudioPreparation enables the optional development journal.
func (g *Game) TraceAudioPreparation() { g.audioPreparationTrace = true }

// AudioPreparations remains valid until the next update begins.
func (g *Game) AudioPreparations() []AudioPreparationSnapshot {
	return g.audioPreparations
}

// EnableAudio attaches the original sample and music data to a native player.
// Gameplay chooses sounds; playback consumes completed commands and no RNG.
func (g *Game) EnableAudio() error {
	if g.audioEngine != nil {
		return nil
	}
	data, err := audio.LoadData()
	if err != nil {
		return err
	}
	player, err := audio.NewPlayer(data)
	if err != nil {
		return err
	}
	g.audioEngine, g.audioControl = player, player.Control
	player.MusicEnabled, player.EffectsEnabled = !g.musicDisabled, !g.effectsDisabled
	if g.masterVolumeValue != 0 {
		player.SetMasterVolume(g.masterVolumeValue - 1)
	}
	if g.demonstrating {
		g.resetArenaAudio()
	}
	return nil
}

// AudioFrame remains valid until the next completed update.
func (g *Game) AudioFrame() (audio.Frame, bool) {
	if g.audioEngine == nil {
		return audio.Frame{}, false
	}
	return g.audioEngine.Frame(), true
}

func (g *Game) beginSoundFrame() {
	g.audioPreparations = g.audioPreparations[:0]
	if g.audioEngine != nil {
		g.audioInFrame, g.audioEffectsDone = true, false
		g.audioEngine.BeginFrame()
	} else if g.audioControl != nil {
		g.audioControl.BeginFrame()
	}
}

func (g *Game) endSoundFrame() {
	if g.audioEngine == nil {
		return
	}
	g.updateSoundEffects()
	g.audioTicks++
	g.audioEngine.FinishFrame(g.audioTicks * g.PALFrameClocks())
	g.audioInFrame = false
}

func (g *Game) resetArenaAudio() {
	if g.audioEngine != nil {
		g.audioEngine.ResetArena()
	} else if g.audioControl != nil {
		g.audioControl.ClearArenaMemory()
	}
	g.soundCooldown = [2]byte{}
}

// updateSoundEffects preserves the original order of effect selection and
// gameplay RNG. The offsets are audio delays within one update, not raster
// events; no CPU instructions, display waits or interrupt service run here.
func (g *Game) updateSoundEffects() {
	if g.audioEngine == nil || g.audioEffectsDone {
		return
	}
	begin := g.audioTicks * g.PALFrameClocks()
	for phase, offset := range [...]uint64{15209, 19749, 24289} {
		var before audio.ControlState
		var effectEnd uint64
		if g.audioPreparationTrace {
			before = g.audioControl.State
			effectEnd = g.audioEngine.EffectEnd()
		}
		g.audioEngine.ProcessEffects(begin+offset, g.OriginalMode(), g.OriginalVBL(), byte(g.Bonus.Number), g.originalRandomByte)
		if g.audioPreparationTrace {
			g.audioPreparations = append(g.audioPreparations, AudioPreparationSnapshot{
				Frame: g.Frame, At: begin + offset, Phase: phase, Mode: g.OriginalMode(), EffectEnd: effectEnd,
				Before: before, After: g.audioControl.State,
			})
		}
	}
	g.audioEffectsDone = true
}

func (g *Game) tickSoundTimers() {
	for index := range g.soundCooldown {
		if g.soundCooldown[index] != 0 {
			g.soundCooldown[index]--
		}
	}
	if g.audioControl != nil {
		g.audioControl.TickDelayed(g.EffectsEnabled())
	}
}

func (g *Game) requestSound(id uint16, fromObject bool) {
	if g.audioControl != nil {
		g.audioControl.Request(id, g.EffectsEnabled(), fromObject)
	}
}

func (g *Game) dispatchSound() {
	if g.audioControl == nil {
		return
	}
	var poses [3]audio.Pose
	for id, fighter := range g.Fighters {
		poses[id] = audio.Pose{Frame: byte(fighter.SpriteFrame), Flags: fighter.computerAnimation.Flags, Hit: g.soundHits[id], Down: g.soundDown[id]}
	}
	g.audioControl.Dispatch(poses, g.EffectsEnabled())
}

// Floor sound suppression uses the original shared cooldown bytes.
func (g *Game) soundFloorBounce(before byte, after originalBallState) {
	if before != after.Bounces && g.soundCooldown == [2]byte{} {
		g.soundCooldown[1] = 10
		g.requestSound(3, true)
	}
}
