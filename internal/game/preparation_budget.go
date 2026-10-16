package game

// preparationWorkRate selects the remaining interval and the native drawing
// throughput. All queues use the same region boundaries and unit scale.
func preparationWorkRate(position, drawingRate int) (limit, rate int) {
	if drawingRate == 0 {
		return picturePreparationQuota, 1000
	}
	if position < 52*picturePreparationQuota/313 {
		return 52 * picturePreparationQuota / 313, drawingRate
	}
	if position < 252*picturePreparationQuota/313 {
		return 252 * picturePreparationQuota / 313, 1000
	}
	return picturePreparationQuota, drawingRate
}

// advancePreparationDebt preserves fractional work across task and update
// boundaries. The unit is an ordinary native animation budget, not CPU state.
func advancePreparationDebt(debt, remainder *int, position, limit, rate int) int {
	available := limit - position
	capacity := (available*rate + *remainder) / 1000
	used := min(*debt, capacity)
	elapsed := max(0, (used*1000-*remainder+rate-1)/rate)
	*remainder += elapsed*rate - used*1000
	*debt -= used
	return position + elapsed
}
