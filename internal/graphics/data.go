// Package graphics provides recovered artwork with native pose and sprite IDs.
// Source executable addresses belong only to the offline extractor and tests.
package graphics

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/olivierh59500/ikplus/assets"
)

// Sprite is a sixteen-pixel column with explicit four-plane rows. Combining
// these authored bits preserves transparent pixels and palette substitutions.
type Sprite struct {
	Rows [][4]uint16
}

// Run is an authored vertical strip. X and Y are coordinates within the pose.
type Run struct {
	X, Y int
	Rows [][3]uint16
}

// Pose contains drawable strips and the independent ground shadow.
type Pose struct {
	Runs   []Run
	Shadow [8][5]uint16
}

// Outline is one retained seven-row step in the growing scenery silhouette.
type Outline struct {
	Offset int
	Rows   [7]uint16
}

// Water contains the authored wave and sparkle tables as native values.
type Water struct {
	Jitter     [16]byte
	Waveform   [32]byte
	RowOffsets [21]byte
	Patterns   [256]byte
	LeftMasks  [9]byte
	RightMasks [9]byte
	Edges      [18]struct{ Offset, Phase int }
	Glitter    [3][]uint16
	Ink        [9][4]uint32
}

// TitleMotion is one authored letter move, expressed in cells and pixels.
type TitleMotion struct {
	Column, Row, Span, Action uint16
}

// Title holds palette targets, choreography and decoded logo/instruction rows.
type Title struct {
	Black              [16]uint16
	ParticlePalette    [8]uint16
	ParticleTail       [8]uint16
	ParticleExit       [8]uint16
	ParticleColors     []uint16
	Fades              [8][16]uint16
	InstructionPalette [16]uint16
	LetterPalette      [16]uint16
	Rainbow            [8][8]uint16
	ParticleOrigins    [4][8]uint16
	ParticleVelocity   [2][4][2]uint16
	LetterSteps        [4]uint16
	LetterOrder        [2][60]byte
	Motions            [2][]TitleMotion
	FinalColors        [154]struct {
		Colors    [8]uint16
		Rotations uint16
	}
	Logo         [121][4][10]uint16
	FinalLogo    [120][4][10]uint16
	Legends      [2][41][4][10]uint16
	Instructions [81][4][10]uint16
	Letters      [60][13][4]uint16
	Slots        [22]struct {
		Delay        int16
		Mirrored     bool
		Rows, Offset int
	}
}

// Hint identifies valid authored text; invalid source entries are not commands.
type Hint struct {
	Valid bool
	Text  []byte
}

// Score contains ranked-table labels and native text/animation settings.
type Score struct {
	Headers        [2][27]byte
	BeltLabels     [2][24][8]byte
	BeltThresholds [23]uint16
	BeltColors     [2][24]uint16
	NameRows       [6]uint16
	Hints          [2][32]Hint
	InitialNames   [51][3]byte
	GermanClock    [5]byte
	SpeedDelay     [10]byte
}

// Data is shared immutable artwork. IDs select decoded cells, never memory.
type Data struct {
	Version           int
	SourceSHA256      string
	Fighters          [88]Pose
	Sprites           []Sprite
	Shields           [4][2]int
	Balls             [8][256]int
	Fragments         [4][8]int
	BombFuses         [36]int
	BombBody          int
	CaptionTail       int
	KickMarker        int
	SpeedLabel        [2]int
	Speeds            [5][2]int
	Hands             [2][2]int
	HandX             [2]int
	Stars             [3][6][2]int
	Drool             [6][5][2]int
	DroolPosition     [6][2]int
	Puffs             [3][2]int
	Fish              [2][20]int
	Ripples           [12]int
	ScriptStamp       int
	Pac               [2][10]int
	PacFacing         [4]bool
	Peri              [22]int
	PeriFeet          [7]int
	Border            [8]int
	Foreground        []byte
	EdgeMasks         [3]uint32
	ProjectileShadows [8][8]uint16
	HUDFont           [64][8]byte
	ScoreFont         [64][5]byte
	HintColors        [32]byte
	Birds             [16][5]uint16
	Outlines          [41]Outline
	Water             Water
	Title             Title
	Score             Score
	Background        []byte
	InitialWater      [21]byte
	GraphicsColors    [4]uint16
	ArenaPolicy       struct{ Defence, Activity, Target, Idle byte }
}

var recovered = sync.OnceValues(func() (*Data, error) {
	raw, err := assets.Read("original/graphics.json")
	if err != nil {
		return nil, err
	}
	var data Data
	if err = json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if data.Version != 2 || len(data.Sprites) < 2 || len(data.Foreground) != 200*40+4 ||
		len(data.Background) != 32768 || len(data.Title.ParticleColors) != 33 ||
		len(data.Title.Motions[0]) != 14 || len(data.Title.Motions[1]) != 15 {
		return nil, fmt.Errorf("invalid native graphics data")
	}
	for _, cells := range data.Balls {
		for _, id := range cells {
			if id <= 0 || id >= len(data.Sprites) || len(data.Sprites[id].Rows) != 11 {
				return nil, fmt.Errorf("incomplete retained ball artwork")
			}
		}
	}
	return &data, nil
})

// Load returns the shared native artwork decoded once from embedded assets.
func Load() (*Data, error) { return recovered() }

// MustLoad is used by game components whose embedded artwork is mandatory.
func MustLoad() *Data {
	data, err := Load()
	if err != nil {
		panic(err)
	}
	return data
}
