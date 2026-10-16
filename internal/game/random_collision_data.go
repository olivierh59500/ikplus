package game

// These bytes are authored random-table values selected by the picture phase.
var collisionEntropy = [2][4][4]byte{
	{
		{0, 6, 110, 32},
		{0, 6, 174, 32},
		{0, 6, 142, 32},
		{0, 6, 206, 32},
	},
	{
		{0, 6, 142, 32},
		{0, 6, 206, 32},
		{0, 6, 110, 32},
		{0, 6, 174, 32},
	},
}
