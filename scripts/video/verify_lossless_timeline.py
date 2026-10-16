#!/usr/bin/env python3
"""Verify every lossless montage frame against its declared native source."""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import subprocess
from fractions import Fraction
from pathlib import Path

from PIL import Image


ROOT = Path(__file__).resolve().parents[2]
FRAME_BYTES = 320 * 200 * 4


def decoded_hashes(ffmpeg: str, path: Path):
    command = [
        ffmpeg, "-hide_banner", "-loglevel", "error", "-threads", "2",
        "-i", str(path), "-pix_fmt", "rgba", "-fps_mode", "passthrough",
        "-f", "rawvideo", "pipe:1",
    ]
    process = subprocess.Popen(command, stdout=subprocess.PIPE)
    try:
        while True:
            frame = bytearray()
            while len(frame) < FRAME_BYTES:
                part = process.stdout.read(FRAME_BYTES - len(frame))
                if not part:
                    break
                frame.extend(part)
            if not frame:
                break
            if len(frame) != FRAME_BYTES:
                raise RuntimeError(f"Truncated lossless frame in {path}")
            yield hashlib.sha256(frame).hexdigest()
    finally:
        process.stdout.close()
        if process.wait():
            raise RuntimeError(f"Cannot decode {path}")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--timeline", type=Path, default=ROOT / "scripts/video/timeline-native-313.json")
    parser.add_argument("--metadata", type=Path, required=True, help="record-native JSON metadata")
    parser.add_argument("--source", type=Path, help="optional complete lossless record-native video")
    parser.add_argument("--work", type=Path, required=True, help="composition directory containing clip-NN.mkv")
    parser.add_argument("--title", type=Path, required=True, help="opening title PNG")
    parser.add_argument("--output", type=Path, help="optional JSON verification report")
    parser.add_argument("--ffmpeg", default=shutil.which("ffmpeg") or "ffmpeg")
    args = parser.parse_args()

    timeline = json.loads(args.timeline.read_text(encoding="utf-8"))
    metadata = json.loads(args.metadata.read_text(encoding="utf-8"))
    source_hashes = metadata["FrameSHA256"]
    source_rate = Fraction(str(timeline.get("source_fps", timeline["fps"])))
    title_hash = hashlib.sha256(Image.open(args.title).convert("RGBA").tobytes()).hexdigest()
    checked = gameplay = held = titles = 0

    decoded_source = 0
    if args.source:
        actual_source = list(decoded_hashes(args.ffmpeg, args.source))
        if actual_source != source_hashes:
            if len(actual_source) != len(source_hashes):
                raise RuntimeError(f"Source has {len(actual_source)} frames; metadata declares {len(source_hashes)}")
            mismatch = next(i for i, pair in enumerate(zip(actual_source, source_hashes)) if pair[0] != pair[1])
            raise RuntimeError(f"Lossless source differs from metadata at frame {mismatch}")
        decoded_source = len(actual_source)

    for index, clip in enumerate(timeline["clips"]):
        actual = list(decoded_hashes(args.ffmpeg, args.work / f"clip-{index:02d}.mkv"))
        if len(actual) != clip["frames"]:
            raise RuntimeError(f"Clip {index} has {len(actual)} frames; expected {clip['frames']}")
        if clip.get("kind") == "title":
            expected = [title_hash] * clip["frames"]
            titles += len(expected)
        elif clip.get("hold"):
            source = round(clip["source_start"] * float(source_rate))
            expected = [source_hashes[source]] * clip["frames"]
            held += len(expected)
        else:
            first, last = clip["source_tick_start"], clip["source_tick_end"]
            expected = source_hashes[first:last]
            gameplay += len(expected)
        if actual != expected:
            mismatch = next(i for i, pair in enumerate(zip(actual, expected)) if pair[0] != pair[1])
            raise RuntimeError(f"Clip {index} differs at local frame {mismatch}")
        checked += len(actual)

    if checked != timeline["frames"]:
        raise RuntimeError(f"Verified {checked} frames; timeline declares {timeline['frames']}")
    report = {
        "VerifiedFrames": checked,
        "GameplayFrames": gameplay,
        "TitleFrames": titles,
        "HeldFrames": held,
        "SourceFrames": metadata["Frames"],
        "DecodedSourceFrames": decoded_source,
        "SourceRevision": metadata.get("SourceRevision", ""),
        "Scope": "Every decoded RGBA frame from the lossless montage clips matches its declared native source frame or deliberate held card before H.264 encoding.",
    }
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
