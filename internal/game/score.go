package game

// $7D44 adds to both BCD score stores, including bonus contacts/awards.
// Both six-digit domains wrap; a new scene resets only the round store.
func (f *Fighter) addOriginalScore(value int) {
	f.Score = (f.Score + value) % 1000000
	f.RoundScore = (f.RoundScore + value) % 1000000
}
