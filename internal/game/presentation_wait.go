package game

// WaitPoint describes the native scene phase retained by a pause command.
// Resuming uses the scene's existing Go controller, never a saved CPU context.
type WaitPoint uint32

const (
	WaitNone WaitPoint = iota
	WaitArenaSetup
	WaitFightPicture
	WaitBallPicture
	WaitNextFrame
	WaitTitleActors
	WaitScoresExit
)
