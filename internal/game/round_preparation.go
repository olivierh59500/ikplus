package game

// Round preparation is native work spread across measured game updates.
// Fighter placement precedes ready publication; bonus entry has no jitter.
type roundPreparationStage byte

const (
	roundPreparationNone roundPreparationStage = iota
	roundPreparationPublish
	roundPreparationFighters
	roundPreparationBonus
	roundPreparationBonusReturn
)

func (g *Game) tickRoundPreparation() {
	switch g.roundPreparation {
	case roundPreparationFighters:
		g.roundPreparation = roundPreparationPublish
		g.advanceRound()
	case roundPreparationPublish:
		g.roundPreparation = roundPreparationNone
		g.rememberFightBoundary()
	case roundPreparationBonus:
		g.roundPreparation = roundPreparationNone
		g.advanceRound()
	case roundPreparationBonusReturn:
		hold := g.Bonus.kind == 0 && g.Bonus.Success
		g.roundPreparation = roundPreparationNone
		g.advanceRound()
		// The returning arena publishes READY immediately, then retains the
		// prepared fight picture through one ordinary update before its next
		// dispatcher visit.
		g.fightBonusReturnHold = hold
	}
}
