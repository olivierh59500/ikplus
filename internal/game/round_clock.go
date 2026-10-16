package game

// originalRoundClock translates the original arena countdown from $11B8 and
// the timeout window at $6200. The first displayed second uses 25 PAL ticks;
// later seconds use 50. Ready action 28 pauses this eligible-tick counter.
func originalRoundClock(eligibleTicks int) (seconds, subsecond int, continuing bool) {
	if eligibleTicks < 25 {
		return 30, 25 - eligibleTicks, true
	}
	elapsed := eligibleTicks - 25
	seconds = max(0, 29-elapsed/50)
	subsecond = 50 - elapsed%50
	return seconds, subsecond, seconds != 0 || subsecond >= 20
}
