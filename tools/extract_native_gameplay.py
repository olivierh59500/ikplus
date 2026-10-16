#!/usr/bin/env python3
"""Extract named native gameplay tables without runtime executable addresses."""

import argparse
import hashlib
import json
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parent.parent


def extract():
    raw = (ROOT / "assets/original/game.bin").read_bytes()
    source = bytes(0x600) + raw

    def bytes_at(at, length):
        return list(source[at:at + length])

    def words(at, count):
        return list(struct.unpack_from(f">{count}H", source, at))

    def script(at):
        end = source.index(255, at)
        return bytes_at(at, end - at + 1)

    def caption(at):
        end = source.index(0x9B, at, at + 128)
        return bytes_at(at, end - at + 1)
    data = dict(
        version=1, sourceSHA256=hashlib.sha256(raw).hexdigest(),
        # Keep each byte-indexed animation/control domain, including authored
        # aliases beyond the smaller human-facing move catalog.
        animation=dict(starts=bytes_at(0x9E0A, 256), ends=bytes_at(0x9E0B, 256),
                       frames=bytes_at(0x9E34, 256), motion=bytes_at(0x9F28, 256),
                       flags=bytes_at(0xA01C, 256), widths=bytes_at(0x98A2, 256),
                       recovery=bytes_at(0x98F2, 6), getUp=bytes_at(0x78CC, 6)),
        ai=dict(counterClasses=bytes_at(0x72D2, 256), reactions=bytes_at(0x7327, 256)),
        input=dict(commands=[bytes_at(at, 32) for at in (0x6E6C, 0x6E8C, 0x6EAC, 0x6ECC)],
                   defenceRelation=bytes_at(0x6E64, 8), keyDecode=bytes_at(0x1ADE, 128),
                   hintSignatures=list(struct.unpack_from(">20I", source, 0x6A68)),
                   hintCounts=bytes_at(0x6ABC, 24)),
        collision=dict(frames=bytes_at(0x74B0, 10), tips=bytes_at(0x74BB, 10),
                       masks=bytes_at(0x74C5, 10), widths=bytes_at(0x74CF, 10),
                       poses=bytes_at(0x7544, 73), bodies=bytes_at(0x758D, 216),
                       rear=bytes_at(0x72C2, 8), reactions=bytes_at(0x751C, 40),
                       frontScores=bytes_at(0x74D9, 10), rearScores=bytes_at(0x74E3, 10),
                       frontStuns=bytes_at(0x7501, 10), rearStuns=bytes_at(0x750B, 10)),
        bonus=dict(failureActions=bytes_at(0xBC5E, 2), reboundColors=bytes_at(0xBCD4, 8),
                   ballHeights=bytes_at(0xC124, 8), ballLimits=bytes_at(0xC12C, 8),
                   ballRates=bytes_at(0xC134, 8), bombDelays=bytes_at(0xC5A4, 8),
                   bombGoal=source[0xC9A1], bombPositions=bytes_at(0xC994, 6),
                   fragmentMotion=[bytes_at(0xCE36 + i * 4, 4) for i in range(4)],
                   fragmentX=bytes_at(0xCF84, 4), fragmentY=bytes_at(0xCF88, 4),
                   fragmentVX=words(0xCF8C, 16), fragmentVY=words(0xCFAC, 16),
                   blastFrames=bytes_at(0xCCF2, 16)),
        ambient=dict(initialClock=source[0x18866], initialDelay=source[0x18867], initial=source[0x18868] != 0,
                     birdDirection=bytes_at(0x193D6, 8), birdEnd=bytes_at(0x193DE, 8),
                     outlineEnd=bytes_at(0x1936E, 8), rippleEnd=bytes_at(0x199CC, 8),
                     selectionData=bytes_at(0x1936A, 4), modes=bytes_at(0x19366, 4),
                     x=bytes_at(0x19376, 32), y=bytes_at(0x19396, 32), colors=bytes_at(0x193B6, 32),
                     scripts=bytes_at(0x19C74, 16), fishX=bytes_at(0x199D4, 16),
                     fishY=bytes_at(0x199E4, 16), fishRepeats=bytes_at(0x199F4, 16),
                     dropX=words(0x19CA4, 16), dropY=words(0x19CC4, 16), dropMotion=words(0x19C84, 16),
                     pacDirection=words(0x19E68, 4), pacX=words(0x19E70, 4),
                     pacEnd=words(0x19E78, 4), pacY=words(0x19E80, 4),
                     periScripts=[script(struct.unpack_from(">I", source, 0x1A142 + i * 4)[0]) for i in range(8)],
                     periX=words(0x1A162, 8), periY=words(0x1A172, 8)),
        caption=dict(widths=[bytes_at(at, 40) for at in (0x88FD, 0x1B844)],
                     speaking=bytes_at(0x89FE, 128), referees=bytes_at(0x8190, 4),
                     sparkleX=bytes_at(0x8218, 32), sparkleY=bytes_at(0x8239, 32),
                     messages=[[caption(struct.unpack_from(">I", source, table + i * 4)[0])
                                for i in range(40)] for table in (0x8926, 0x1B86E)],
                     nameLengths=bytes_at(0x8324, 3)),
        judge=dict(choices=bytes_at(0x5CD0, 28), idle=bytes_at(0x5CEC, 32), timers=bytes_at(0x5D0C, 6)),
        pause=dict(positions=bytes_at(0x5B30, 3), actions=bytes_at(0x5B33, 13)),
        hitStars=[bytes_at(at, 10) for at in (0x74ED, 0x74F7)],
        defaults=dict(masterVolume=words(0x626, 1)[0], graphics=words(0xDAA, 1)[0],
                      ground=words(0xE62, 1)[0], belts=[words(at, 1)[0] for at in (0xE4E, 0xE52, 0xE42)]),
    )
    return (json.dumps(data, separators=(",", ":")) + "\n").encode()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="verify the checked-in native gameplay data")
    args = parser.parse_args()
    path = ROOT / "assets/original/gameplay.json"
    content = extract()
    if args.check:
        if not path.exists() or path.read_bytes() != content:
            raise SystemExit("Native gameplay data differ; run tools/extract_native_gameplay.py")
    else:
        path.write_bytes(content)
    print(f"Native gameplay {'verified' if args.check else 'extracted'}: animation, input, collisions, bonuses, scenery and captions")


if __name__ == "__main__":
    main()
