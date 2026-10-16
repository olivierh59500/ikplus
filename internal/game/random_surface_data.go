package game

// Recovered lookup vocabulary for surface identities. These are read-only
// mixer inputs and never addresses used by a renderer or game controller.
var pictureBufferSample = [...][4]byte{
	{0, 0, 0, 0},
	{0, 7, 128, 0},
	{0, 7, 0, 0},
}
var coveragePlaneSample = [...][4]byte{
	{0, 0, 0, 0},
	{0, 6, 110, 32},
	{0, 6, 142, 32},
	{0, 6, 174, 32},
	{0, 6, 206, 32},
}
var restorationSamples = loadRandomAsset("original/random-restoration.bin", 1536)

func restorationSample(cursor RestorationCursor, shift uint8) byte {
	if cursor.Buffer > PictureSecond || cursor.Count >= 128 {
		return 0
	}
	return restorationSamples[(int(cursor.Buffer)*128+int(cursor.Count))*4+3-int(shift)/8]
}
