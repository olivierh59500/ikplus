#!/usr/bin/env python3
"""Build the current full-success showcase from recorded native milestones."""

import argparse
import hashlib
import json
from fractions import Fraction
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--recording", type=Path, required=True, help="record-native JSON metadata")
    parser.add_argument("--out", type=Path, default=ROOT / "scripts/video/timeline-native-313.json")
    parser.add_argument("--seed", type=int, default=18, help="documented deterministic tournament seed")
    args = parser.parse_args()
    source = json.loads(args.recording.read_text())
    rate = Fraction(source["PALClock"], source["FrameClocks"])
    seconds = lambda frames: float(Fraction(frames, 1) / rate)

    def state(round_number, phase):
        return next(c for c in source["Milestones"] if c["Round"] == round_number and c["Phase"] == phase)

    last = state(13, "READY")
    if source["Frames"] < last["VideoFrame"] + 35 or last["WhiteScore"] != 61100 or last["WhiteBelt"] != "BLACK  3":
        raise ValueError("This edit requires the complete current BLACK 3 round-13 recording.")
    for round_number in (3, 9):
        result = state(round_number, "BONUS RESULT")
        if not result["BonusSuccess"] or result["Deflected"] != 64:
            raise ValueError("The selected recording must retain both complete shield results.")
    for round_number, kicks in ((6, 22), (12, 20)):
        result = state(round_number, "BONUS RESULT")
        if not result["BonusSuccess"] or result["BonusNumber"] != 32 or result["Deflected"] != kicks:
            raise ValueError("The selected recording must retain both complete bomb results.")

    cards = round(8 * rate)
    clips = [{"kind": "title", "frames": cards, "duration": seconds(cards)}]

    def excerpt(first, end):
        if not 0 <= first < end <= source["Frames"]:
            raise ValueError("A selected interval falls outside the source recording.")
        clips.append(dict(source_start=seconds(first), frames=end-first, duration=seconds(end-first),
                          source_tick_start=first, source_tick_end=end))

    excerpt(0, state(2, "JUDGE")["VideoFrame"] + 75)
    end = state(4, "READY")["VideoFrame"] - 37
    excerpt(end - 1050, end)
    excerpt(state(4, "READY")["VideoFrame"], state(5, "JUDGE")["VideoFrame"] + 75)
    end = state(7, "READY")["VideoFrame"] - 31
    excerpt(end - 930, end)
    excerpt(state(7, "READY")["VideoFrame"], state(8, "JUDGE")["VideoFrame"] + 75)
    end = state(10, "READY")["VideoFrame"] - 19
    excerpt(end - 850, end)
    excerpt(state(10, "READY")["VideoFrame"], state(11, "JUDGE")["VideoFrame"] + 75)
    end = last["VideoFrame"] + 34
    excerpt(end - 1785, end)
    clips.append(dict(source_start=seconds(end - 1), frames=cards, duration=seconds(cards), hold=True))

    titles = [
        ("THE AMIGA CLASSIC", "Original artwork / Native Go + Ebitengine"),
        ("OPENING BOUTS", "Rounds 1-2 / Ordinary joystick controls"),
        ("SHIELD REFLEXES", "Round 3 / Closing highlights and all 64 deflections"),
        ("FIGHTING FOR POSITION", "Rounds 4-5 / The same uninterrupted run"),
        ("BURNING FUSES", "Round 6 / All 32 bombs survived with 22 kicks"),
        ("PURPLE BELT PRESSURE", "Rounds 7-8 / Both complete fights"),
        ("THE SECOND WAVE", "Round 9 / Another 64 deflections"),
        ("BLACK BELT BOUTS", "Rounds 10-11 / Both complete fights"),
        ("THE FINAL BOMB STAGE", "Round 12 / All 32 bombs survived with 20 kicks"),
        ("ROUND 13 REACHED", "BLACK 3 / 61,100 points / Native port preview"),
    ]
    chapters, cues = [], []
    offset = 0
    for clip, (title, detail) in zip(clips, titles):
        chapters.append(dict(start=seconds(offset), end=seconds(offset + clip["frames"]), title=title, detail=detail))
        offset += clip["frames"]

    def cue(chapter, start, end, text):
        c = chapters[chapter]
        if not 0 <= start < end <= c["end"]-c["start"]:
            raise ValueError("A caption falls outside its chapter.")
        cues.append(dict(start=c["start"]+start, end=c["start"]+end, text=text))

    cue(0, .7, 7.2, "The Amiga classic, rebuilt in Go + Ebitengine.\nA skilled simulated player controls white.")
    cue(1, .8, 8, "One human player, two computer opponents.\nEvery move uses ordinary joystick controls.")
    cue(1, 18, 25, "Timing and distance control the exchange.\nBoth opening fights are retained in full.")
    cue(1, 35, 42, "Keep both opponents in view.\nThe first shield challenge comes next.")
    cue(2, .6, 7, "High, middle, low: follow the incoming balls.\nKeep the shield in the path of the next attack.")
    cue(2, 10, 16, "Left and right, the pressure keeps building.\nClosing highlights from the first shield stage.")
    cue(2, 17.9, 20.8, "64 deflections.\nBack to the tournament.")
    cue(3, .8, 7.8, "More points, tougher exchanges.\nBoth complete fights remain in the cut.")
    cue(3, 20, 27, "A strike on one side can leave the other side open.\nWatch the space between all three fighters.")
    cue(3, 33, 38.1, "Blue takes the fifth bout.\nWhite stays in the tournament.")
    cue(4, .6, 7, "Burning fuses replace the shield challenge.\nKick danger away, then move clear.")
    cue(4, 12.8, 18.2, "All 32 bombs are survived, with 22 clean kicks.\nThe uninterrupted tournament continues.")
    cue(5, .8, 7.8, "Later in the same run, both rivals remain dangerous.\nWhite works toward the purple belt.")
    cue(5, 20, 27, "Blue wins one exchange, but white stays in contention.\nThe next bout begins without a restart.")
    cue(5, 31, 37, "White answers with a clear win.\nAnother wave of balls follows.")
    cue(6, .6, 7, "Another shield challenge, from both sides.\nHigh and low: keep adjusting.")
    cue(6, 13.6, 16.8, "64 deflections again.\nThe brown belt is earned.")
    cue(7, .8, 7.8, "Brown turns to black as the pressure rises.\nBoth complete fights remain in the cut.")
    cue(7, 15, 22, "When the clock runs down, every point matters.\nStay out of reach and choose the next opening.")
    cue(7, 27, 33.5, "Round eleven ends with white in first place.\nThe last bomb challenge comes next.")
    cue(8, .8, 7.8, "Round twelve: closing bomb-stage highlights.\nKick, move, and watch the blast.")
    cue(8, 15, 22, "Fuses keep burning after a kick.\nMoving away gives the next attack room to pass.")
    cue(8, 32.25, 35.5, "All 32 bombs survived, with 20 kicks.\nRound thirteen reached.")
    cue(9, .5, 7.3, "Round 13. Black 3. 61,100 points.\nA native Go + Ebitengine preview.")

    fingerprint = hashlib.sha256()
    for path in sorted((ROOT / "internal").rglob("*.go")):
        if not path.name.endswith("_test.go"):
            fingerprint.update(path.relative_to(ROOT).as_posix().encode())
            fingerprint.update(path.read_bytes())
    result = dict(version=3, title="IK+", subtitle="THE AMIGA CLASSIC / GO + EBITENGINE", fps=str(rate), source_fps=str(rate),
                  source_canvas=[320, 200], source_frames=source["Frames"], duration=seconds(offset), frames=offset, seed=args.seed,
                  source_commit=source.get("SourceRevision", ""), source_worktree_modified=source.get("SourceModified", False),
                  source_code_sha256=fingerprint.hexdigest(),
                  source_code_sha256_definition="SHA-256 over each sorted internal/**/*.go path followed by its bytes, excluding *_test.go",
                  source_recording=str(args.recording.with_suffix("")), source_video_sha256=source["VideoSHA256"], source_audio_sha256=source["AudioSHA256"],
                  display_profile="Normal native 313-line playback, shared software scene compositor before GPU upload and application overlays",
                  editorial_notes=[f"One uninterrupted seed-{args.seed} tournament controlled by ordinary joystick input from the observable-state pilot.",
                                   "All eight combat fights through round eleven are retained in full; judgement and bonus sections are excerpted chronologically.",
                                   "Both ball stages complete all 64 deflections. Both bomb stages survive all 32 bombs, with 22 and 20 kicks.",
                                   "White advances through wins and qualifying places; the edit does not claim eight first-place finishes.",
                                   "No speed changes, restarts, score edits or forced outcomes. Title and final achievement cards are deliberate stills.",
                                   "This is a current native preview. Complete fidelity to the original remains under verification."],
                  clips=clips, chapters=chapters, cues=cues)
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(result, indent=2) + "\n")
    print(f"Wrote {offset} frames / {seconds(offset):.3f} seconds / {len(chapters)} chapters: {args.out}")


if __name__ == "__main__":
    main()
