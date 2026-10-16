package pilot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/olivierh59500/ikplus/internal/audio"
	"github.com/olivierh59500/ikplus/internal/game"
)

// Transition records natural tournament transitions, including failed bonuses
// and defeats. No failure is hidden by restarting the game.
type Transition struct {
	Frame       uint64 `json:"frame"`
	Round       int    `json:"round"`
	Phase       string `json:"phase"`
	Points      [3]int `json:"points"`
	RoundScores [3]int `json:"round_scores"`
	Ranks       [3]int `json:"ranks"`
	Score       int    `json:"score"`
	Belt        string `json:"belt"`
}

// BonusResult keeps completion separate from the number of successful contacts.
// A bomb stage can be survived by kicking some bombs and avoiding other blasts.
type BonusResult struct {
	Round    int    `json:"round"`
	Kind     string `json:"kind"`
	Frame    uint64 `json:"frame"`
	Spawned  int    `json:"spawned"`
	Total    int    `json:"total"`
	Contacts int    `json:"contacts"`
	Success  bool   `json:"success"`
}

// BeltChange records every earned original belt grade during the campaign.
type BeltChange struct {
	Frame uint64 `json:"frame"`
	Round int    `json:"round"`
	Score int    `json:"score"`
	Belt  string `json:"belt"`
}

// CampaignReport is deterministic evidence from one uninterrupted human match.
type CampaignReport struct {
	PALFrameLines  int           `json:"pal_frame_lines"`
	AudioEnabled   bool          `json:"audio_enabled"`
	Seed           uint64        `json:"seed"`
	TargetRound    int           `json:"target_round"`
	ReachedTarget  bool          `json:"reached_target"`
	Status         string        `json:"status"`
	Frames         uint64        `json:"frames"`
	FinalRound     int           `json:"final_round"`
	FinalPhase     string        `json:"final_phase"`
	Score          int           `json:"score"`
	Belt           string        `json:"belt"`
	CombatRounds   int           `json:"combat_rounds"`
	CombatWins     int           `json:"combat_wins"`
	HitsGiven      int           `json:"hits_given"`
	HitsTaken      int           `json:"hits_taken"`
	Transitions    []Transition  `json:"transitions"`
	Bonuses        []BonusResult `json:"bonuses"`
	Belts          []BeltChange  `json:"belts"`
	TraceSHA256    string        `json:"trace_sha256"`
	InvariantError string        `json:"invariant_error,omitempty"`
}

// TraceFrame contains legal replay input and visible state after one PAL tick.
// The trace deliberately excludes random state, internal action commands and
// private animation/control flags. It can be compared to a recorded Amiga run.
type TraceFrame struct {
	Frame         uint64            `json:"frame"`
	Round         int               `json:"round"`
	Phase         string            `json:"phase"`
	Input         game.Input        `json:"input"`
	Fighters      [3]VisibleFighter `json:"fighters"`
	BonusContacts int               `json:"bonus_contacts"`
	BonusSpawned  int               `json:"bonus_spawned"`
	Balls         []VisibleBall     `json:"balls,omitempty"`
	Bombs         []VisibleBomb     `json:"bombs,omitempty"`
}

// VisibleFighter is the compact state available from the rendered match.
type VisibleFighter struct {
	X, Y                            float64
	Facing                          int
	Move                            string
	SpriteFrame                     int
	Points, RoundScore, Score, Rank int
	Human, Eliminated               bool
	ShieldX, ShieldY                float64
	ShieldState, ShieldSide         int
}

// VisibleBall records public projectile telemetry without its private source
// integrator state. Only active projectiles appear in a frame's trace.
type VisibleBall struct {
	X, Y, VX, VY        float64
	Frame, Color        int
	Deflected, Flashing bool
}

// VisibleBomb preserves the public fuse telemetry used by the tactical player.
// Exact stage timers are stronger observations than the current rendered fuse.
type VisibleBomb struct {
	X, Y, VX, VY                       float64
	Fuse, Stage, StageTicks, Explosion int
	Kicked                             bool
	Fragment                           bool
	Frame, Variant, BlastFrame         int
}

// RunCampaign retains the controlled historical profile used by original
// input recordings. RunNativeCampaign exercises current desktop/mobile setup.
func RunCampaign(seed uint64, targetRound, maxTicks int, trace func(TraceFrame)) CampaignReport {
	g := game.New(seed)
	return runCampaign(g, seed, targetRound, maxTicks, trace)
}

// RunNativeCampaign starts once and supplies only observable-state human input
// under the normal native playback profile, without restarts after defeat.
func RunNativeCampaign(seed uint64, targetRound, maxTicks int, trace func(TraceFrame)) CampaignReport {
	g := game.New(seed)
	if err := g.SetPALFrameProfile(audio.Supplied313); err != nil {
		return CampaignReport{Seed: seed, TargetRound: targetRound, InvariantError: err.Error()}
	}
	g.UseNativePlayback()
	return runCampaign(g, seed, targetRound, maxTicks, trace)
}

func runCampaign(g *game.Game, seed uint64, targetRound, maxTicks int, trace func(TraceFrame)) CampaignReport {
	g.Start(1)
	if err := g.EnableAudio(); err != nil {
		return CampaignReport{Seed: seed, TargetRound: targetRound, InvariantError: err.Error()}
	}
	p := New(0)
	report := CampaignReport{Seed: seed, TargetRound: targetRound, AudioEnabled: true, PALFrameLines: int(g.PALFrameClocks() / 227)}
	digest := sha256.New()
	previousPhase, previousRound := game.Attract, 0
	previousBelt := ""
	bonusKind := game.BallBonus
	previousScore := [3]int{}
	for tick := 0; tick < maxTicks && g.Round < targetRound && g.Phase != game.GameOver; tick++ {
		in := p.Input(g)
		g.Tick([2]game.Input{in})
		state := makeTrace(g, in)
		// This fixed JSON layout makes the same run independently repeatable.
		encoded, _ := json.Marshal(state)
		digest.Write(encoded)
		digest.Write([]byte{'\n'})
		if trace != nil {
			trace(state)
		}
		if err := checkCampaign(g, previousRound, previousScore); err != nil {
			// Preserve the input and result that triggered a consistency error,
			// allowing the failing tick itself to be replayed and investigated.
			report.InvariantError = err.Error()
			break
		}
		if g.Phase != previousPhase || g.Round != previousRound {
			transition := Transition{Frame: g.Frame, Round: g.Round, Phase: g.Phase.String(), Score: g.Fighters[0].Score, Belt: g.Fighters[0].Belt()}
			for i, f := range g.Fighters {
				transition.Points[i], transition.RoundScores[i], transition.Ranks[i] = f.Points, f.RoundScore, f.Rank
			}
			report.Transitions = append(report.Transitions, transition)
			if g.Phase == game.Judge {
				report.CombatRounds++
				if g.Fighters[0].Rank == 1 {
					report.CombatWins++
				}
			}
			if g.Phase == game.BallBonus || g.Phase == game.BombBonus {
				bonusKind = g.Phase
			}
			if g.Phase == game.BonusEnd {
				report.Bonuses = append(report.Bonuses, BonusResult{Round: g.Round, Kind: bonusKind.String(), Frame: g.Frame, Spawned: g.Bonus.Number, Total: g.Bonus.Total, Contacts: g.Bonus.Deflected, Success: g.Bonus.Success})
			}
		}
		if belt := g.Fighters[0].Belt(); belt != previousBelt {
			report.Belts = append(report.Belts, BeltChange{Frame: g.Frame, Round: g.Round, Score: g.Fighters[0].Score, Belt: belt})
			previousBelt = belt
		}
		for _, e := range g.Events {
			if e.Kind != game.HitEvent {
				continue
			}
			if e.Fighter == 0 {
				report.HitsGiven++
			}
			if e.Target == 0 {
				report.HitsTaken++
			}
		}
		previousPhase, previousRound = g.Phase, g.Round
		for i, f := range g.Fighters {
			previousScore[i] = f.Score
		}
	}
	report.Frames, report.FinalRound, report.FinalPhase = g.Frame, g.Round, g.Phase.String()
	report.Score, report.Belt = g.Fighters[0].Score, g.Fighters[0].Belt()
	report.ReachedTarget = g.Round >= targetRound && g.Fighters[0].Human && report.InvariantError == ""
	report.Status = "tick_limit"
	if report.ReachedTarget {
		report.Status = "reached_target"
	}
	if g.Phase == game.GameOver {
		report.Status = "eliminated"
	}
	if report.InvariantError != "" {
		report.Status = "invariant_failure"
	}
	report.TraceSHA256 = hex.EncodeToString(digest.Sum(nil))
	return report
}

func makeTrace(g *game.Game, in game.Input) TraceFrame {
	t := TraceFrame{Frame: g.Frame, Round: g.Round, Phase: g.Phase.String(), Input: in, BonusContacts: g.Bonus.Deflected, BonusSpawned: g.Bonus.Number}
	for i, f := range g.VisibleFighters() {
		t.Fighters[i] = VisibleFighter{X: f.X, Y: f.Y, Facing: f.Facing, Move: f.Move.String(), SpriteFrame: f.SpriteFrame, Points: f.Points, RoundScore: f.RoundScore, Score: f.Score, Rank: f.Rank, Human: f.Human, Eliminated: f.Eliminated, ShieldX: f.ShieldX, ShieldY: f.ShieldY, ShieldState: f.ShieldState, ShieldSide: f.ShieldSide}
	}
	for _, b := range g.Bonus.Balls {
		if b.Active {
			t.Balls = append(t.Balls, VisibleBall{X: b.X, Y: b.Y, VX: b.VX, VY: b.VY, Frame: b.Frame, Color: b.Color, Deflected: b.Deflected, Flashing: b.Flashing})
		}
	}
	for _, b := range g.Bonus.Bombs {
		if b.Active {
			t.Bombs = append(t.Bombs, VisibleBomb{X: b.X, Y: b.Y, VX: b.VX, VY: b.VY, Fuse: b.Fuse, Stage: b.Stage, StageTicks: b.StageTicks, Explosion: b.Explosion, Kicked: b.Kicked, Fragment: b.Fragment, Frame: b.Frame, Variant: b.Variant, BlastFrame: b.BlastFrame})
		}
	}
	return t
}

func checkCampaign(g *game.Game, previousRound int, previousScore [3]int) error {
	if previousRound > 0 && (g.Round < previousRound || g.Round > previousRound+1) {
		return fmt.Errorf("round jumped from %d to %d", previousRound, g.Round)
	}
	for i, f := range g.Fighters {
		if f.Score < 0 || f.Score >= 1000000 {
			return fmt.Errorf("fighter %d score is outside the original six-digit store", i)
		}
		// Both combat and bonus awards can wrap the BCD score, not only the
		// judge's 100-point increments. A perfect bonus is 5000; allow its
		// accompanying contacts while rejecting a reset or negative award.
		if f.Score < previousScore[i] && f.Score+1000000-previousScore[i] > 6000 {
			return fmt.Errorf("fighter %d score decreased", i)
		}
		if f.Points < 0 || f.Points > 7 {
			return fmt.Errorf("fighter %d has %d points", i, f.Points)
		}
		if f.Human && f.Eliminated && g.Phase != game.Judge && g.Phase != game.GameOver {
			return fmt.Errorf("fighter %d remains human after elimination", i)
		}
	}
	if (g.Phase == game.BallBonus || g.Phase == game.BombBonus) && !g.BonusPreparing() {
		if g.Round%3 != 0 {
			return fmt.Errorf("bonus outside displayed third round: %d", g.Round)
		}
		kind, total := game.BallBonus, 64
		if g.Round/3%2 == 0 {
			kind, total = game.BombBonus, 32
		}
		if g.Round == 99 {
			// The displayed round saturates while the source cycle continues.
			kind = g.Phase
			if kind == game.BombBonus {
				total = 32
			} else {
				total = 64
			}
		}
		if g.Phase != kind || g.Bonus.Total != total {
			return fmt.Errorf("wrong bonus kind or quota at round %d", g.Round)
		}
		if g.Bonus.Number > g.Bonus.Total || g.Bonus.Deflected > g.Bonus.Number {
			return fmt.Errorf("invalid bonus contacts/quota at round %d", g.Round)
		}
	}
	return nil
}
