#!/usr/bin/env python3
"""Extract the recovered random lookup vocabulary for the native Go mixer.

Source offsets belong only to this development extractor. The running game
uses immutable data and named Go state samples, without a mapped address space.
"""

import argparse
import json
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parent.parent


def extract():
    source = bytes(1536) + (ROOT / "assets/original/game.bin").read_bytes()
    fields = json.loads((ROOT / "tools/reference-data/presentation-fields.json").read_text())
    first = 0x785C - 32768
    length = 65536 + 100
    fields = [field for field in fields if field["address"] < first + length]
    audio_fields = [
        (0x864, "AudioPrevious", "byte", 3),
        (0x867, "AudioSelected", "byte", 1),
        (0x868, "AudioFrame", "byte", 1),
        (0x86A, "AudioFlags", "byte", 1),
        (0x86B, "AudioActor", "byte", 1),
        (0x86E, "AudioDelay", "byte", 1),
        (0x86F, "AudioDelayed", "byte", 1),
        (0x12CA, "AudioLength", "uint16", 1),
        (0x94E, "AudioBusy", "uint16", 1),
        (0x15D8, "AudioCommands", "uint16", 1),
        (0x15DA, "AudioMainPending", "uint16", 1),
        (0x15DC, "AudioActive", "uint16", 1),
        (0x15DE, "AudioMainID", "uint16", 1),
        (0x15E0, "AudioID", "uint16", 1),
        (0x15E2, "AudioObjectID", "uint16", 1),
        (0x15E4, "AudioObjectPending", "uint16", 1),
        (0x902, "AudioSubstep", "byte", 1),
        (0xD5D2, "AudioPitch", "uint16", 1),
        (0xD5D4, "AudioVoices", "byte", 248),
        (0x312C, "Particles", "uint16", 1800),
    ]
    for address, name, kind, count in audio_fields:
        fields.append(dict(address=address, name=name, kind=kind, count=count))
    for index in range(9):
        fields.append(dict(address=0x2D68 + index * 36, name=f"RainbowLine{index}", kind="uint16", count=1))
    fields.append(dict(address=0x96DC, name="CollisionEnabled", kind="uint16", count=1))
    names = list(dict.fromkeys(field["name"].replace("[].", "") for field in fields))
    tokens, plan = [], {}
    for field in fields:
        width = {"byte": 1, "uint16": 2, "uint32": 4}[field["kind"]]
        kind = names.index(field["name"].replace("[].", ""))
        for index in range(field["count"]):
            for byte in range(width):
                address = field["address"] + index * width + byte
                if first <= address < first + length:
                    plan[address] = 256 + len(tokens)
                    tokens.append((kind, index, (width - 1 - byte) * 8, address))
    noise = bytes(source[at] if 0 <= at < len(source) else 0 for at in range(first, first + length))
    indices = b"".join(struct.pack("<H", plan.get(at, 0)) for at in range(first, first + length))
    definitions = b"".join(struct.pack("<HHB", kind, index, shift) for kind, index, shift, _ in tokens)
    addresses = b"".join(struct.pack("<I", at) for _, _, _, at in tokens)
    restoration = b"".join(
        struct.pack(">I", 0 if bank == 0 else (0x1C1DC, 0x1C5DC)[bank - 1] + count * 6)
        for bank in range(3) for count in range(128)
    )
    return {
        "assets/original/random-noise.bin": noise,
        "assets/original/random-samples.bin": indices,
        "assets/original/random-inputs.bin": definitions,
        "assets/original/random-restoration.bin": restoration,
        "internal/game/testdata/native-random-sample-addresses.bin": addresses,
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="Verify extracted files without writing them.")
    options = parser.parse_args()
    mismatches = []
    for name, data in extract().items():
        path = ROOT / name
        if options.check:
            if not path.exists() or path.read_bytes() != data:
                mismatches.append(name)
        else:
            path.write_bytes(data)
    if mismatches:
        parser.error("Recovered random data differs: " + ", ".join(mismatches))
    print("Native random vocabulary verified." if options.check else "Native random vocabulary extracted.")


if __name__ == "__main__":
    main()
