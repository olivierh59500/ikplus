package game

import (
	"encoding/json"

	"github.com/olivierh59500/ikplus/assets"
)

// gameplayData contains decoded authored values. Controllers use native table
// indices and state; no loaded executable or original memory layout is needed.
type gameplayData struct {
	Version      int
	SourceSHA256 string
	Animation    struct {
		Starts, Ends, Frames, Motion, Flags, Widths [256]byte
		Recovery, GetUp                             [6]byte
	}
	AI    struct{ CounterClasses, Reactions [256]byte }
	Input struct {
		Commands        [4][32]byte
		DefenceRelation [8]byte
		KeyDecode       [128]byte
		HintSignatures  [20]uint32
		HintCounts      [24]byte
	}
	Collision struct {
		Frames, Tips, Masks, Widths []byte
		Poses, Bodies, Rear         []byte
		Reactions                   []byte
		FrontScores, RearScores     []byte
		FrontStuns, RearStuns       []byte
	}
	Bonus struct {
		FailureActions                     [2]byte
		ReboundColors                      [8]byte
		BallHeights, BallLimits, BallRates [8]byte
		BombDelays                         [8]byte
		BombGoal                           int
		BombPositions                      [6]byte
		FragmentMotion                     [4][4]byte
		FragmentX, FragmentY               [4]byte
		FragmentVX, FragmentVY             [16]uint16
		BlastFrames                        [16]byte
	}
	Ambient struct {
		InitialClock, InitialDelay         byte
		Initial                            bool
		BirdDirection, BirdEnd             [8]byte
		OutlineEnd, RippleEnd              [8]byte
		SelectionData, Modes               [4]byte
		X, Y, Colors                       [32]byte
		Scripts, FishX, FishY, FishRepeats [16]byte
		DropX, DropY, DropMotion           [16]uint16
		PacDirection, PacX, PacEnd, PacY   [4]uint16
		PeriScripts                        [8][]byte
		PeriX, PeriY                       [8]uint16
	}
	Caption struct {
		Widths             [2][40]byte
		Speaking           [128]byte
		Referees           [4]byte
		SparkleX, SparkleY [32]byte
		Messages           [2][40][]byte
		NameLengths        [3]byte
	}
	Judge struct {
		Choices [28]byte
		Idle    [32]byte
		Timers  [6]byte
	}
	Pause struct {
		Positions [3]byte
		Actions   [13]byte
	}
	HitStars [2][10]byte
	Defaults struct {
		MasterVolume, Graphics, Ground uint16
		Belts                          [3]uint16
	}
}

var nativeGameplay = func() gameplayData {
	raw, err := assets.Read("original/gameplay.json")
	if err != nil {
		panic(err)
	}
	var data gameplayData
	if err = json.Unmarshal(raw, &data); err != nil {
		panic(err)
	}
	if data.Version != 1 || len(data.Collision.Frames) != 10 ||
		len(data.Collision.Poses) != 73 || len(data.Collision.Bodies) != 216 ||
		len(data.Collision.FrontScores) != 10 || len(data.Collision.RearScores) != 10 ||
		len(data.Collision.FrontStuns) != 10 || len(data.Collision.RearStuns) != 10 ||
		len(data.Ambient.PeriScripts[0]) == 0 || len(data.Caption.Messages[0][0]) == 0 {
		panic("invalid native gameplay data")
	}
	return data
}()
