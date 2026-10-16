package audio

// PALFrameProfile selects a rational game update duration for source replays.
// These constants do not create scanlines or a hardware scheduler.
type PALFrameProfile byte

const (
	// Controlled312 preserves the existing source-opcode comparison profile.
	Controlled312 PALFrameProfile = iota
	// Supplied313 follows the supplied disk's observed noninterlaced PAL field.
	Supplied313
)

// FrameClocks returns zero for an unknown profile.
func (p PALFrameProfile) FrameClocks() uint64 {
	switch p {
	case Controlled312:
		return 227 * 312
	case Supplied313:
		return 227 * 313
	default:
		return 0
	}
}

const PALFrameClocks = uint64(227 * 312)
