#!/usr/bin/env python3
"""Decode fighter runs and named sprite cells for the native Go renderer."""

import argparse
import base64
import hashlib
import json
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parent.parent


def extract():
    raw = (ROOT / "assets/original/game.bin").read_bytes()
    source = bytes(0x600) + raw
    runtime = bytearray(source)
    word = lambda at: struct.unpack_from(">H", source, at)[0]
    long = lambda at: struct.unpack_from(">I", source, at)[0]
    # Recover the authored bitmap layout once, outside the running game.
    for base in (0x1CA5C, 0x248CA, 0x2C5CA):
        for y in range(200):
            for x in range(20):
                for plane in range(4):
                    start = base + y * 160 + x * 8 + plane * 2
                    target = base + y * 160 + plane * 40 + x * 2
                    runtime[target:target + 2] = source[start:start + 2]

    sprites = [dict(rows=[])]
    sprite_ids = {}

    def sprite(at, height):
        key = (at, height)
        if key not in sprite_ids:
            sprite_ids[key] = len(sprites)
            sprites.append(dict(rows=[[struct.unpack_from(">H", runtime, at + y * 160 + plane * 40)[0]
                                       for plane in range(4)] for y in range(height)]))
        return sprite_ids[key]

    fighters = []
    for index in range(88):
        runs = []
        relative = long(0x48220 + index * 4)
        at = 0x48220 + relative
        if relative:
            for _ in range(100):
                offset, height = word(at), word(at + 2) & 127
                at += 4
                if height == 0:
                    break
                rows = [list(struct.unpack_from(">3H", source, at + row * 6)) for row in range(height)]
                runs.append(dict(x=(offset % 160) // 4 * 8, y=offset // 160, rows=rows))
                at += height * 6
            else:
                raise ValueError(f"fighter {index} lacks a run terminator")
        shadow = [[word(0x483A0 + index * 80 + row * 10 + x * 2) for x in range(5)] for row in range(8)]
        fighters.append(dict(runs=runs, shadow=shadow))

    def words(at, count):
        return [word(at + i * 2) for i in range(count)]

    def rectangle(at, rows):
        return [[list(struct.unpack_from(">10H", runtime, at + row * 160 + plane * 40))
                 for plane in range(4)] for row in range(rows)]

    def hint(at):
        if at == 0:
            return dict(valid=True, text=[])
        if at >= len(source):
            return dict(valid=False)
        end = source.index(0x9B, at)
        return dict(valid=True, text=list(source[at:end + 1]))

    motions = []
    for tables, count in (((0x52D2, 0x52EE, 0x530A, 0x5326), 14), ((0x5342, 0x5360, 0x537E, 0x539C), 15)):
        motions.append([dict(column=word(tables[0] + i * 2), row=word(tables[1] + i * 2),
                             span=word(tables[2] + i * 2), action=word(tables[3] + i * 2)) for i in range(count)])
    particle_colors = []
    for i in range(128):
        value = word(0x2C74 + i * 2)
        particle_colors.append(value)
        if value == 0:
            break
    else:
        raise ValueError("unterminated particle color sequence")
    title = dict(
        black=words(0x2A1A, 16), particlePalette=words(0x2A3A, 8),
        particleTail=words(0x2C54, 8), particleExit=words(0x2C64, 8), particleColors=particle_colors,
        fades=[words(at, 16) for at in (0x4E36, 0x4E6A, 0x4F02, 0x4F32, 0x4F62, 0x4F92, 0x4FE2, 0x5012)],
        instructionPalette=words(0x5042, 16), letterPalette=words(0x5116, 16),
        rainbow=[[word(0x2D6E + band * 36 + color * 4) for color in range(8)] for band in range(8)],
        particleOrigins=[words(long(0x2F0C + i * 4), 8) for i in range(4)],
        particleVelocity=[[words(long(0x2FDA + group * 4) + i * 4, 2) for i in range(4)] for group in range(2)],
        letterSteps=words(0x5186, 4), letterOrder=[list(source[at:at + 60]) for at in (0x518E, 0x51CA)],
        motions=motions,
        finalColors=[dict(colors=words(0x578E + i * 2, 8), rotations=word(0x579C + i * 2) >> 12) for i in range(154)],
        logo=rectangle(0x248CA, 121), finalLogo=rectangle(0x248DE, 120),
        legends=[rectangle(at, 41) for at in (0x293CA, 0x2ACCA)], instructions=rectangle(0x293CA, 81),
        letters=[[[struct.unpack_from(">H", runtime, 0x293DE + (i // 10) * 13 * 160 + (i % 10) * 2 + row * 160 + plane * 40)[0]
                   for plane in range(4)] for row in range(13)] for i in range(60)],
        slots=[dict(delay=struct.unpack_from(">b", source, 0x276E + i)[0],
                    mirrored=source[0x2785 + i] >= 128, rows=source[0x279B + i],
                    offset=long(0x27B2 + i * 4) - 0x70000) for i in range(22)],
    )

    data = dict(
        version=2, sourceSHA256=hashlib.sha256(raw).hexdigest(), fighters=fighters,
        shields=[[sprite(at + col * 2, 23) for col in range(2)]
                 for at in (0x225B4, 0x225B6, 0x225B8, 0x2399C)],
        balls=[[sprite(0x1E2BC + color * 0x6E0 + frame * 2, 11) for frame in range(9)] for color in range(8)],
        fragments=[[sprite(0x1DC94 + variant * 0x640 + frame * 2, 9) for frame in range(8)] for variant in range(4)],
        bombFuses=[sprite(long(0xCB60 + frame * 4), 13) for frame in range(36)],
        bombBody=sprite(0x22E7C, 10), captionTail=sprite(0x22E7E, 7), kickMarker=sprite(0x22DBC, 1),
        speedLabel=[sprite(0x2317C + col * 2, 5) for col in range(2)],
        speeds=[[sprite(long(0x6C0E + speed * 4) + col * 2, 5) for col in range(2)] for speed in range(5)],
        hands=[[sprite(long(table + owner * 4), 15) for owner in range(2)] for table in (0x6CD8, 0x6CD0)],
        handX=[long(0x6CC8 + owner * 4) for owner in range(2)],
        stars=[[[sprite(0x234AC + color * 0x640 + frame * 4 + col * 2, 9)
                  for col in range(2)] for frame in range(6)] for color in range(3)],
        drool=[[[sprite(long(0x91A6 + pose * 4 + col * 24) + frame * 4, 12)
                  for col in range(2)] for frame in range(5)] for pose in range(6)],
        droolPosition=[[struct.unpack_from(">h", source, 0x918E + pose * 2)[0], word(0x919A + pose * 2)]
                       for pose in range(6)],
        puffs=[[sprite(0x23860 + frame * 4 + col * 2, 24) for col in range(2)] for frame in range(3)],
        fish=[[sprite(0x1CA5C + frame * 2 + direction * 0x8C0, 14) for frame in range(20)] for direction in range(2)],
        ripples=[sprite(0x1DDBC + frame * 2, 4) for frame in range(12)],
        scriptStamp=sprite(0x22FAC, 2),
        pac=[[sprite(long(table + frame * 4), 8) for frame in range(10)] for table in (0x19E88, 0x19EB0)],
        pacFacing=[word(0x19E68 + frame * 2) < 32768 for frame in range(4)],
        peri=[sprite(long(0x1A00C + pose * 4), 9) for pose in range(22)],
        periFeet=[sprite(long(0x1A064 + frame * 4), 4) for frame in range(7)],
        border=[sprite(0x22A9C + frame * 2, 3) for frame in range(8)],
        foreground=list(runtime[0x1A324 - 16 * 40:0x1A324 + 184 * 40 + 4]),
        edgeMasks=[long(0x19696 + edge * 4) for edge in range(3)],
        projectileShadows=[[word(0xC14C - height * 2 + row * 2) for row in range(8)] for height in range(8)],
        hudFont=[list(source[0x8E9E + glyph * 8:0x8E9E + (glyph + 1) * 8]) for glyph in range(64)],
        scoreFont=[list(source[0x909E + glyph * 5:0x909E + (glyph + 1) * 5]) for glyph in range(64)],
        hintColors=list(source[0x2484:0x24A4]),
        birds=[[word(0x1944A + frame * 10 + row * 2) for row in range(5)] for frame in range(16)],
        outlines=[dict(offset=word(0x1985A + frame * 2),
                       rows=[word(0x1981A + ((frame & 1) ^ 1) * 32 + word(0x198AC + frame * 2) + row * 2)
                             for row in range(7)]) for frame in range(41)],
        water=dict(
            jitter=list(source[0x19190:0x191A0]),
            waveform=list(source[0x18A60:0x18A80]),
            rowOffsets=list(source[0x18A92:0x18AA7]),
            patterns=list(source[0x18AAA:0x18BAA]),
            leftMasks=list(source[0x18A80:0x18A89]), rightMasks=list(source[0x18A89:0x18A92]),
            edges=[dict(offset=long(0x18D56 + edge * 4), phase=word(0x18D9E + edge * 2)) for edge in range(18)],
            glitter=[[word(table + i * 2) for i in range(count)]
                     for table, count in ((0x18E04, 64), (0x18E84, 32), (0x18EC4, 64))],
            ink=[[long(0x18BB2 + pattern * 16 + col * 4) for col in range(4)] for pattern in range(9)],
        ),
        sprites=sprites,
        title=title,
        score=dict(
            headers=[list(source[at:at + 27]) for at in (0xAB32, 0x1BA22)],
            beltLabels=[[list(source[at + i * 8:at + (i + 1) * 8]) for i in range(24)] for at in (0xAA12, 0x1BA3E)],
            beltThresholds=words(0xA9E4, 23), beltColors=[words(at, 24) for at in (0xAAD2, 0xAB02)],
            nameRows=words(0xA6DE, 6), hints=[[hint(long(table + i * 4)) for i in range(32)] for table in (0x200E, 0x1B9A4)],
            initialNames=[list(source[0x16F2 + i * 3:0x16F5 + i * 3]) for i in range(51)],
            germanClock=list(source[0x1BA1C:0x1BA21]), speedDelay=list(source[0x65A0:0x65AA]),
        ),
        initialWater=list(source[0x18A04:0x18A19]), graphicsColors=words(0xEBC, 4),
        arenaPolicy=dict(defence=source[0x6EED], activity=source[0x6EF5], target=source[0x6EFD], idle=source[0x9E0A + 28]),
    )
    # Retained spin bytes select adjacent authored words rather than wrapping
    # into the nine ordinary pictures. Append these decoded views after the
    # existing cells so all established native sprite IDs remain unchanged.
    for color, cells in enumerate(data["balls"]):
        cells.extend(sprite(0x1E2BC + color * 0x6E0 + frame * 2, 11)
                     for frame in range(9, 256))
    background = bytearray(runtime[0x2C5CA:0x2C5CA + 32768])
    for row in range(192, 200):
        background[row * 160:row * 160 + 40] = bytes(40)
    data["background"] = base64.b64encode(background).decode()
    return (json.dumps(data, separators=(",", ":")) + "\n").encode(), len(sprites), sum(len(f["runs"]) for f in fighters)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="verify checked-in native graphics")
    args = parser.parse_args()
    content, sprites, runs = extract()
    path = ROOT / "assets/original/graphics.json"
    if args.check:
        if not path.exists() or path.read_bytes() != content:
            raise SystemExit("Native graphics differ; run tools/extract_native_graphics.py")
    else:
        path.write_bytes(content)
    print(f"Native graphics {'verified' if args.check else 'extracted'}: 88 poses, {runs} runs, {sprites - 1} sprite cells")


if __name__ == "__main__":
    main()
