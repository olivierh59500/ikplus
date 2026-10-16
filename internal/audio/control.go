// Package audio plays IK+'s original sample and music data with native Go updates.
package audio

import (
	"fmt"
)

// Pose supplies the bytes read by the original frame-trigger dispatcher.
type Pose struct {
	Frame, Flags, Hit, Down byte
}

// ControlState retains the original sound selection bytes for deterministic
// gameplay and reference comparisons. Historical slot names describe the
// original data layout; no interrupts or Copper processor run in this player.
type ControlState struct {
	Previous                        [3]byte
	Selected, Frame, Flags, Actor   byte
	Delay, Delayed                  byte
	CopperCount                     uint16
	MainPending, Active, MainID, ID uint16
	IRQID, IRQPending               uint16
	Busy, LengthCounter             uint16
	Channel                         [4]uint32
}

// Triggers returns the gameplay-facing selection state. Channel registers and
// sample completion counters belonged to the old hardware comparison profile.
func (s ControlState) Triggers() ControlState {
	s.Channel = [4]uint32{}
	s.Busy, s.LengthCounter, s.CopperCount = 0, 0, 0
	return s
}

// Control selects effects from fighter poses and two priority request slots.
type Control struct {
	State ControlState
	data  *Data
}

func NewControl(data *Data) (*Control, error) {
	if data == nil || len(data.Effects) != 18 {
		return nil, fmt.Errorf("native sound tables are missing")
	}
	return &Control{data: data}, nil
}

// ClearArenaMemory clears pose triggers while retaining pending requests.
func (c *Control) ClearArenaMemory() {
	s := &c.State
	s.Previous = [3]byte{}
	s.Selected, s.Frame, s.Flags, s.Actor = 0, 0, 0, 0
	s.Delay, s.Delayed, s.Busy = 0, 0, 0
}

// Request keeps the original priority: a fighter request wins over an object
// request at the next update. Later requests replace earlier ones in each slot.
func (c *Control) Request(id uint16, enabled, fromObject bool) {
	id &= 31
	if id >= 18 || !enabled {
		return
	}
	if fromObject {
		c.State.IRQID, c.State.IRQPending = id, 1
	} else {
		c.State.MainID, c.State.MainPending = id, 1
	}
}

func (c *Control) Dispatch(poses [3]Pose, enabled bool) {
	if !enabled {
		return
	}
	s := &c.State
	s.Actor, s.Selected = 2, 255
	for actor := 2; actor >= 0; actor-- {
		pose, previous := poses[actor], s.Previous[actor]
		s.Previous[actor] = pose.Frame
		if pose.Frame != previous {
			s.Frame, s.Flags = pose.Frame, pose.Flags
			for index := 0; index < 17; index++ {
				if c.data.Triggers.Frames[index] != pose.Frame {
					continue
				}
				selected := byte(255)
				switch {
				case pose.Hit != 0:
					selected = c.data.Triggers.Hits[index]
					if delay := c.data.Triggers.HitDelays[index]; int8(delay) > 0 {
						s.Delay, s.Delayed = delay, c.data.Triggers.HitDelayed[index]
					}
				case pose.Down != 0:
					selected = c.data.Triggers.Down[index]
				case pose.Flags&0x4d != 0:
					selected = c.data.Triggers.Ordinary[index]
					if delay := c.data.Triggers.OrdinaryDelays[index]; int8(delay) > 0 {
						s.Delay, s.Delayed = delay, c.data.Triggers.OrdinaryDelayed[index]
					}
				}
				if int8(selected) >= 0 {
					s.Selected = selected
				}
				break
			}
		}
		s.Actor--
	}
	if int8(s.Selected) >= 0 {
		c.Request(uint16(s.Selected), true, false)
	}
}

// BeginFrame promotes at most one queued effect before the music update.
func (c *Control) BeginFrame() bool {
	s := &c.State
	s.CopperCount = 0
	s.Active = s.MainPending
	if s.Active != 0 {
		s.MainPending, s.ID = 0, s.MainID
	} else {
		s.Active = s.IRQPending
		if s.Active == 0 {
			return false
		}
		s.IRQPending, s.ID = 0, s.IRQID
	}
	s.Busy = 0xffff
	return true
}

// TickDelayed preserves delayed pose sounds, including their one-update queue.
func (c *Control) TickDelayed(enabled bool) {
	if c.State.Delay == 0 {
		return
	}
	c.State.Delay--
	if c.State.Delay == 0 {
		c.Request(uint16(c.State.Delayed), enabled, true)
	}
}

func (c *Control) effect(mode, frame, number byte, random func() byte) Channel {
	id := c.State.ID & 31
	sample := c.data.Effects[id]
	period := uint16(358)
	if mode == 5 {
		period = 420 - uint16(number)*2 + uint16(frame&7)
	} else if mode != 7 {
		first, second := random()&63, random()&15
		period = 314 + uint16(first) + uint16(second)
	}
	return Channel{Offset: sample.Offset, Length: sample.Length, Period: period, Volume: 64, Enabled: true}
}
