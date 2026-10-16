#!/usr/bin/env python3
"""Compare delivered fighter-region frame order with its lossless source."""

import argparse
import json
from pathlib import Path
import subprocess

from PIL import Image, ImageChops, ImageStat

WIDTH, HEIGHT = 102, 39
FRAME_SIZE = WIDTH * HEIGHT


def decode(path, video_filter):
    """Decode small grayscale signatures without changing frame timestamps."""
    command = [
        "ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", "2",
        "-i", str(path), "-vf", video_filter, "-pix_fmt", "gray",
        "-fps_mode", "passthrough", "-f", "rawvideo", "pipe:1",
    ]
    process = subprocess.Popen(command, stdout=subprocess.PIPE)
    try:
        while True:
            data = bytearray()
            while len(data) < FRAME_SIZE:
                part = process.stdout.read(FRAME_SIZE - len(data))
                if not part:
                    break
                data.extend(part)
            if not data:
                break
            if len(data) != FRAME_SIZE:
                raise RuntimeError("Truncated decoded frame")
            yield Image.frombytes("L", (WIDTH, HEIGHT), bytes(data))
    finally:
        process.stdout.close()
        if process.wait():
            raise RuntimeError("Video decode failed")


def main():
    root = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, default=root / ".cache/video/v2/gameplay.mkv")
    parser.add_argument("--video", type=Path, default=root / "videos/ikplus-showcase.mp4")
    parser.add_argument("--timeline", type=Path, default=root / "scripts/video/timeline.json")
    parser.add_argument("--output", type=Path, default=root / ".cache/verification/video-flicker/complete-mp4-order.json")
    args = parser.parse_args()
    timeline = json.loads(args.timeline.read_text())
    canvas_height = timeline.get("source_canvas", [320, 256])[1]
    source_y = 144 - (256 - canvas_height) // 2
    raw = list(decode(args.source, f"crop=204:78:70:{source_y},scale=102:39:flags=neighbor"))
    mapping, ranges = [], []
    for clip in timeline["clips"]:
        start = len(mapping)
        if clip.get("kind") == "title" or clip.get("hold"):
            mapping.extend([None] * clip["frames"])
        else:
            mapping.extend(range(clip["source_tick_start"], clip["source_tick_end"]))
        ranges.append((start, len(mapping)))

    bad, ties, compared, error, decoded = [], 0, 0, 0, 0
    for index, frame in enumerate(decode(args.video, "crop=612:234:690:552,scale=102:39:flags=neighbor")):
        if index >= len(mapping):
            raise RuntimeError("Encoded video exceeds its declared timeline")
        decoded += 1
        expected = mapping[index]
        if expected is None:
            continue
        # Keep nearby candidates within the same clip at editorial cuts.
        first, last = next(item for item in ranges if item[0] <= index < item[1])
        candidates = [mapping[n] for n in range(max(first, index - 3), min(last, index + 4))]
        scores = {
            n: ImageStat.Stat(ImageChops.difference(frame, raw[n])).sum[0]
            for n in candidates
        }
        best, current = min(scores.values()), scores[expected]
        compared += 1
        error += current / FRAME_SIZE
        if current > best:
            bad.append({
                "OutputFrame": index, "SourceFrame": expected,
                "Nearest": min(scores, key=scores.get),
                "ExpectedError": current / FRAME_SIZE, "BestError": best / FRAME_SIZE,
            })
        if sum(value == best for value in scores.values()) > 1:
            ties += 1
    if decoded != len(mapping) or not compared:
        raise RuntimeError("Encoded video is incomplete or has no gameplay")

    report = {
        "SourceFrames": len(raw), "OutputFrames": len(mapping),
        "GameplayFramesCompared": compared, "NearestFrameMismatches": bad,
        "RepeatedPoseTies": ties, "MeanLumaError": error / compared,
        "Scope": "Complete delivered gameplay excerpts, excluding title/held cards; each encoded fighter-region frame compared with original lossless source at declared timeline index and +/-3 nearby frames. Compression and repeated poses prevent pixel identity and unique identification of every stationary frame. No physical Amiga raster or viewer-device proof.",
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({
        key: len(value) if key == "NearestFrameMismatches" else value
        for key, value in report.items()
    }, indent=2))


if __name__ == "__main__":
    main()
