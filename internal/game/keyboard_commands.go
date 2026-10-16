package game

// SourceFrozen reports the authored freeze command, which holds gameplay
// while keyboard and music updates continue.
func (g *Game) SourceFrozen() bool { return g.sourceFrozen }

// JoinRequested reports a pending human start for the frontend transition.
func (g *Game) JoinRequested() int { return g.sourceJoinPlayers }

// UsePhysicalJoystickPins enables CIA fire-input sampling for a complete
// physical-input timeline. Older decoded-input routine fixtures omit it.
func (g *Game) UsePhysicalJoystickPins() { g.physicalJoystickPins = true }

// SetSourceJoystick publishes input before scene keyboard polling.
func (g *Game) SetSourceJoystick(inputs [2]Input) {
	g.joystick = inputs
	if g.physicalJoystickPins && g.audioEngine != nil {
		g.audioEngine.SetFireButtons(inputs[0].Fire, inputs[1].Fire)
	}
	if holder := g.presentationState; holder != nil {
		for id, input := range inputs {
			value := byte(joystickNibble(input))
			if input.Fire {
				value |= 16
			}
			holder.Joystick[id] = value
		}
	}
}

// PausedAt identifies the native scene phase held by the freeze command.
func (g *Game) PausedAt() WaitPoint { return g.pausePoint }

// SetPresentationWait identifies the current native scene wait.
func (g *Game) SetPresentationWait(wait WaitPoint) { g.waitPoint = wait }

func (g *Game) arenaWaitPoint() WaitPoint {
	if g.Phase == BonusEnd {
		b := &g.Bonus
		if b.resultStage == bonusResultFailureHold || b.resultStage == bonusResultHold || b.resultStage == bonusResultFinalFrame {
			return WaitNextFrame
		}
	}
	if g.OriginalMode() == 5 {
		return WaitBallPicture
	}
	return WaitFightPicture
}

func (g *Game) selectorKey() byte {
	if g.presentationState != nil {
		return g.presentationState.Keys[0]
	}
	return g.speedKeys[0]
}

func (g *Game) fastMusic() {
	if g.audioEngine != nil {
		for range 3 {
			g.audioEngine.UpdateMusic(g.MusicEnabled())
		}
	}
}
