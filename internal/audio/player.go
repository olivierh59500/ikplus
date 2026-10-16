package audio

// PALClock is the original pitch reference. It is only a unit for sample
// frequencies and durations; the player has no CPU, bus or interrupt scheduler.
const PALClock = uint64(3546895)

// Event changes one sample voice or the output filter at an audio timestamp.
type Event struct {
	At                                      uint64
	Voice                                   int
	Channel                                 Channel
	Restart, OneShot, Filter, FilterEnabled bool
}

// Frame contains immutable commands produced by a completed game update.
// Playback consumes these commands without calling gameplay or its RNG.
type Frame struct {
	Begin, End uint64
	Events     []Event
	Ownership  []Ownership
}

// Ownership records the effect reservation at an ordinary audio timestamp.
type Ownership struct {
	At   uint64
	Busy uint16
}

// EffectOwnershipAt inspects a completed update without advancing playback.
func (f Frame) EffectOwnershipAt(at uint64) (uint16, bool) {
	if at < f.Begin || at > f.End || len(f.Ownership) == 0 {
		return 0, false
	}
	busy := f.Ownership[0].Busy
	for _, change := range f.Ownership {
		if change.At > at {
			break
		}
		busy = change.Busy
	}
	return busy, true
}

// Player advances original musical sequences and effect selection once per
// game update. Effects finish from their sample length and pitch.
type Player struct {
	Control                            *Control
	Music                              *Music
	MusicEnabled, EffectsEnabled, Fast bool
	Substep                            byte
	clock, frameBegin                  uint64
	effectEnd                          uint64
	events                             []Event
	ownership                          []Ownership
	frame                              Frame
}

func NewPlayer(data *Data) (*Player, error) {
	control, err := NewControl(data)
	if err != nil {
		return nil, err
	}
	music, err := NewMusic(data, 44100)
	if err != nil {
		return nil, err
	}
	p := &Player{Control: control, Music: music, MusicEnabled: true, EffectsEnabled: true, ownership: []Ownership{{}}}
	music.onVoice = func(index int, channel Channel, restart bool) {
		p.events = append(p.events, Event{At: p.clock, Voice: index, Channel: channel, Restart: restart})
		if index == 3 {
			p.syncChannel(channel)
		}
	}
	music.onFilter = func(enabled bool) {
		p.events = append(p.events, Event{At: p.clock, Filter: true, FilterEnabled: enabled})
	}
	return p, nil
}

func (p *Player) syncChannel(c Channel) {
	p.Control.State.Channel = [4]uint32{c.Offset, uint32(c.Length), uint32(c.Period), uint32(c.Volume)}
}

func (p *Player) syncOwnership() {
	busy := p.Control.State.Busy
	if len(p.ownership) == 0 || p.ownership[len(p.ownership)-1].Busy != busy {
		p.ownership = append(p.ownership, Ownership{p.clock, busy})
	}
	p.Music.effectBusy = p.Control.State.Busy != 0
}

// BeginFrame accepts queued effects before advancing envelopes and notes.
func (p *Player) BeginFrame() {
	if p.Control.BeginFrame() {
		p.stopEffect()
	}
	p.syncOwnership()
	p.UpdateMusic(p.MusicEnabled)
}

func (p *Player) UpdateMusic(enabled bool) {
	p.Music.work = MusicWork{}
	if p.Fast {
		p.Substep++
		if int8(p.Substep) >= 6 {
			p.Substep = 0
			return
		}
	}
	if enabled {
		p.Music.Tick()
	}
}

// ProcessEffects follows the original three-stage effect handoff. Its owner
// supplies audio offsets, without simulating raster registers or interrupts.
func (p *Player) ProcessEffects(at uint64, mode, frame, number byte, random func() byte) {
	p.AdvanceTo(at)
	s := &p.Control.State
	if s.Active != 0 {
		switch {
		case s.CopperCount == 0:
			p.stopEffect()
		case s.CopperCount == 1 || s.LengthCounter != 1 || p.effectEnd == 0:
			channel := p.Control.effect(mode, frame, number, random)
			p.Music.channels[3] = voice{offset: channel.Offset, length: channel.Length, period: channel.Period, volume: channel.Volume, enabled: true}
			p.syncChannel(channel)
			s.LengthCounter = 1
			start := at + effectCommandDelay(mode)
			p.effectEnd = start + uint64(sampleLength(channel.Length))*uint64(channel.Period)
			p.events = append(p.events, Event{At: start, Voice: 3, Channel: channel, Restart: true, OneShot: true})
		}
		if s.CopperCount > 1 {
			s.ID, s.Active = 0, 0
		}
	}
	s.CopperCount++
	p.syncOwnership()
}

// Sample selection has a measured preparation delay after its command batch.
// Mode-specific pitch selection uses less work in the two bonus stages. These
// are ordinary event timestamps; playback has no processor or IRQ service.
func effectCommandDelay(mode byte) uint64 {
	switch mode {
	case 5:
		return 326
	case 7:
		return 278
	default:
		return 708
	}
}

// AdvanceTo expires a finite sample. This is an audio duration calculation,
// independent of the amount of PCM requested by an output device.
func (p *Player) AdvanceTo(at uint64) {
	if p.effectEnd != 0 && p.effectEnd <= at {
		p.clock = p.effectEnd
		p.stopEffect()
		// Retain the authored silent sample selection for deterministic
		// randomness and the next musical handoff. Playback remains stopped.
		c := &p.Music.channels[3]
		c.offset, c.length, c.volume = p.Music.data.SilentOffset, 1, 64
		p.syncChannel(Channel{c.offset, c.length, c.period, c.volume, false})
		p.Control.State.LengthCounter, p.Control.State.Busy = 0, 0
		p.syncOwnership()
	}
	p.clock = at
}

func (p *Player) stopEffect() {
	p.effectEnd = 0
	c := &p.Music.channels[3]
	c.enabled, c.volume, c.period = false, 0, 1
	p.syncChannel(Channel{c.offset, c.length, c.period, c.volume, false})
	p.Music.publishVoice(c, false)
}

func (p *Player) ResetArena() {
	p.stopEffect()
	p.Control.ClearArenaMemory()
	p.Substep = 0
	p.syncOwnership()
}

func (p *Player) Restart() {
	p.ResetArena()
	p.Fast = false
	p.MusicEnabled = true
	p.SetMasterVolume(15)
	p.SetLEDFilter(false)
}

func (p *Player) StopMusic() {
	// Cutting playback cancels its completion. The original retains effect
	// ownership until a replacement effect or an arena reset releases it.
	// An unfinished handoff may still restart the effect at its next stage.
	p.effectEnd = 0
	for index := range p.Music.channels {
		c := &p.Music.channels[index]
		c.volume, c.period, c.enabled = 0, 1, false
		p.Music.publishVoice(c, false)
	}
	p.MusicEnabled = false
}

func (p *Player) StartMusic()               { p.MusicEnabled = true; p.SetMasterVolume(15) }
func (p *Player) SetLEDFilter(enabled bool) { p.Music.SetFilter(enabled) }
func (p *Player) LEDFilter() bool           { return p.Music.filter }
func (p *Player) SetFireButtons(white, red bool) {
	p.Music.fireButtons = [2]bool{white, red}
}
func (p *Player) MasterVolume() uint16          { return p.Music.masterVolume }
func (p *Player) SetMasterVolume(volume uint16) { p.Music.masterVolume = volume }

// Frame returns commands until the next FinishFrame call. Between-update
// changes are retained and become part of the next completed frame.
func (p *Player) Frame() Frame { return p.frame }

// EffectEnd reports the finite sample's current playback endpoint. Inspecting
// it does not advance voices, completion, ownership or gameplay randomness.
func (p *Player) EffectEnd() uint64 { return p.effectEnd }

func (p *Player) FinishFrame(end uint64) {
	p.AdvanceTo(end)
	p.frame = Frame{Begin: p.frameBegin, End: end, Events: append(p.frame.Events[:0], p.events...), Ownership: append(p.frame.Ownership[:0], p.ownership...)}
	p.events = p.events[:0]
	p.ownership = append(p.ownership[:0], Ownership{end, p.Control.State.Busy})
	p.frameBegin = end
}

func (p *Player) ReservedVoice() Channel {
	c := p.Music.dummy
	return Channel{c.offset, c.length, c.period, c.volume, c.enabled}
}

func (p *Player) PitchScratch() uint16 { return p.Music.pitchChanges }
