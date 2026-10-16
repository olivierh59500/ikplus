package game

var waitPointSamples = [...][4]byte{
	{0, 0, 0, 0},
	{0, 0, 10, 212},
	{0, 0, 101, 166},
	{0, 0, 119, 92},
	{0, 0, 119, 74},
	{0, 0, 37, 180},
	{0, 0, 162, 230},
}

func waitPointSample(point WaitPoint, shift uint8) byte {
	if int(point) >= len(waitPointSamples) {
		return 0
	}
	return waitPointSamples[point][3-int(shift)/8]
}
