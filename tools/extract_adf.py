#!/usr/bin/env python3
"""Recover IK+ assets from the supplied Amiga OFS disk, without third-party packages.

The loader contains a backwards ByteKiller stream. The game's own display and
sprite routines document the planar layouts decoded below. All addresses refer
to the decompressed game's original Amiga memory map unless otherwise stated.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import struct
import wave
import zlib

DISK_SHA256 = "d1662b6b1704ca211d0cd0fd9899a9b6e92b01a892c7766110e5730ddee3e266"
GAME_BASE = 0x600
SPRITE_TABLE = 0x48220
CELL_WIDTH, CELL_HEIGHT, ATLAS_COLUMNS = 96, 80, 12


def u16(data: bytes, offset: int) -> int:
    """Read a 68000 big-endian word."""
    return struct.unpack_from(">H", data, offset)[0]


def u32(data: bytes, offset: int) -> int:
    """Read a 68000 big-endian longword."""
    return struct.unpack_from(">I", data, offset)[0]


def read_ofs_file(disk: bytes, name: str) -> bytes:
    """Follow OFS directory hashes and data block chains, checking checksums."""
    if len(disk) != 901120 or disk[:4] != b"DOS\0":
        raise ValueError("Expected the supplied 880 KiB Amiga OFS image")

    def block(number: int) -> bytes:
        if not 2 <= number < 1760:
            raise ValueError(f"Invalid OFS block {number}")
        result = disk[number * 512:(number + 1) * 512]
        if sum(struct.unpack(">128I", result)) & 0xFFFFFFFF:
            raise ValueError(f"OFS checksum mismatch in block {number}")
        return result

    visited: set[int] = set()
    headers = [880]
    while headers:
        number = headers.pop()
        if number in visited:
            continue
        visited.add(number)
        header = block(number)
        kind = u32(header, 508)
        filename = header[433:433 + header[432]].decode("latin1")
        if kind == 0xFFFFFFFD and filename == name:
            length, next_block = u32(header, 324), u32(header, 16)
            result = bytearray()
            data_seen: set[int] = set()
            while next_block:
                if next_block in data_seen:
                    raise ValueError("Cyclic OFS data block chain")
                data_seen.add(next_block)
                data = block(next_block)
                if u32(data, 0) != 8 or u32(data, 12) > 488:
                    raise ValueError("Malformed OFS data block")
                result.extend(data[24:24 + u32(data, 12)])
                next_block = u32(data, 16)
            if len(result) != length:
                raise ValueError("OFS file size does not match its header")
            return bytes(result)
        if kind in (1, 2):
            headers.extend(pointer for pointer in struct.unpack_from(">72I", header, 24) if pointer)
        if u32(header, 496):
            headers.append(u32(header, 496))
    raise ValueError(f"OFS file {name!r} was not found")


def hunk_code(hunk: bytes) -> bytes:
    """Read this loader's single HUNK_CODE segment, excluding relocation data."""
    if [u32(hunk, at) for at in (0, 4, 8, 12, 16, 24)] != [1011, 0, 1, 0, 0, 1001]:
        raise ValueError("Unexpected Amiga executable header")
    length = u32(hunk, 28) * 4
    return hunk[32:32 + length]


def bytekiller(code: bytes, start: int = 0x99BC) -> bytes:
    """Translate the original 0x98D4 backwards depacker, including its checksum."""
    packed, length, checksum = struct.unpack_from(">III", code, start)
    source, target = start + 12 + packed, length
    result = bytearray(length)
    word = 0

    def next_bit() -> int:
        nonlocal source, word, checksum
        bit = word & 1
        word >>= 1
        if not word:
            source -= 4
            if source < start + 12:
                raise ValueError("ByteKiller bitstream exhausted")
            word = u32(code, source)
            checksum ^= word
            bit = word & 1
            word = (word >> 1) | 0x80000000
        return bit

    def bits(count: int) -> int:
        value = 0
        for _ in range(count):
            value = (value << 1) | next_bit()
        return value

    def literal(count: int) -> None:
        nonlocal target
        if count > target:
            raise ValueError("ByteKiller literal exceeds output")
        for _ in range(count):
            target -= 1
            result[target] = bits(8)

    def copy(count: int, offset: int) -> None:
        nonlocal target
        if count > target or offset == 0:
            raise ValueError("Invalid ByteKiller backreference")
        for _ in range(count):
            target -= 1
            if target + offset >= length:
                raise ValueError("ByteKiller backreference exceeds output")
            result[target] = result[target + offset]

    source -= 4
    word = u32(code, source)
    checksum ^= word
    while target:
        if not next_bit():
            if not next_bit():
                literal(bits(3) + 1)
            else:
                copy(2, bits(8))
        else:
            kind = bits(2)
            if kind == 3:
                literal(bits(8) + 9)
            elif kind == 2:
                count = bits(8) + 1
                copy(count, bits(12))
            else:
                copy(kind + 3, bits(9 + kind))
    if checksum != 0 or source != start + 12:
        raise ValueError("ByteKiller checksum or stream boundary mismatch")
    return bytes(result)


def rgb(amiga: int) -> tuple[int, int, int]:
    """Expand OCS 12-bit RGB to an exact 8-bit channel representation."""
    return ((amiga >> 8 & 15) * 17, (amiga >> 4 & 15) * 17, (amiga & 15) * 17)


def png(path: Path, width: int, height: int, rgba: bytes | bytearray) -> None:
    """Write an RGBA PNG using only the standard library."""
    if len(rgba) != width * height * 4:
        raise ValueError("PNG pixel buffer has the wrong dimensions")

    def chunk(kind: bytes, body: bytes) -> bytes:
        return struct.pack(">I", len(body)) + kind + body + struct.pack(">I", zlib.crc32(kind + body))

    scanlines = b"".join(b"\0" + rgba[y * width * 4:(y + 1) * width * 4] for y in range(height))
    path.write_bytes(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0))
                    + chunk(b"IDAT", zlib.compress(scanlines, 9)) + chunk(b"IEND", b""))


def copper_palettes(memory: bytes) -> list[list[int]]:
    """Apply the game's copper COLOR writes at their original display rows."""
    events: list[tuple[int, int, int]] = []
    row = -52
    for at in range(0xBF0, 0xE74, 4):
        register, value = u16(memory, at), u16(memory, at + 2)
        if register & 1:
            row = (register >> 8) - 52
        elif 0x180 <= register <= 0x19E:
            events.append((row, (register - 0x180) // 2, value))
    rows = []
    for y in range(200):
        colors = [0] * 16
        for row, register, value in events:
            if row <= y:
                colors[register] = value
        rows.append(colors)
    return rows


def planar_indices(memory: bytes, address: int) -> bytearray:
    """Decode a 320x200 image stored as four successive plane words per 16 pixels."""
    pixels = bytearray(320 * 200)
    for y in range(200):
        for word in range(20):
            planes = struct.unpack_from(">4H", memory, address + y * 160 + word * 8)
            for bit in range(16):
                pixels[y * 320 + word * 16 + bit] = sum(((value >> (15 - bit)) & 1) << p
                                                        for p, value in enumerate(planes))
    return pixels


def color_image(indices: bytes | bytearray, palettes: list[list[int]]) -> bytearray:
    """Color a full screen, retaining the original scanline palette changes."""
    return bytearray(channel for y in range(200) for x in range(320)
                     for channel in (*rgb(palettes[y][indices[y * 320 + x]]), 255))


def recover_sprites(memory: bytes, destination: Path, palette: list[int]) -> list[dict]:
    """Decode 88 column-run frames and the game's three plane routing variants."""
    width, height = ATLAS_COLUMNS * CELL_WIDTH, 8 * CELL_HEIGHT
    atlases = {name: bytearray(width * height * 4) for name in ("white", "red", "blue")}
    metadata = []
    belts = bytearray(width * height * 4)
    sprite_indices = bytearray(88 * CELL_WIDTH * CELL_HEIGHT)
    for index in range(88):
        relative = u32(memory, SPRITE_TABLE + index * 4)
        at = SPRITE_TABLE + relative
        pixels: dict[tuple[int, int], tuple[int, int, int]] = {}
        for _ in range(100):
            offset, count = u16(memory, at), u16(memory, at + 2)
            at += 4
            if count == 0:
                break
            # Startup 0x1E2A converts the word-interleaved X offset to a plane-row offset.
            x, y = (offset % 160) // 4 * 8, offset // 160 - 20
            for row in range(count & 127):
                values = struct.unpack_from(">3H", memory, at)
                at += 6
                for column in range(16):
                    planes = tuple(value >> (15 - column) & 1 for value in values)
                    if any(planes):
                        if not (0 <= x + column < CELL_WIDTH and 0 <= y + row < CELL_HEIGHT):
                            raise ValueError(f"Sprite {index} exceeds its atlas cell")
                        pixels[x + column, y + row] = planes
        else:
            raise ValueError("Sprite run terminator is missing")
        for (x, y), planes in pixels.items():
            sprite_indices[(index * CELL_HEIGHT + y) * CELL_WIDTH + x] = sum(plane << bit for bit, plane in enumerate(planes))
            if planes == (0, 1, 1):
                at = (((index // ATLAS_COLUMNS * CELL_HEIGHT + y) * width)
                      + index % ATLAS_COLUMNS * CELL_WIDTH + x) * 4
                belts[at:at + 4] = bytes((255, 255, 255, 255))
        for name, image in atlases.items():
            for (x, y), (plane0, plane1, plane2) in pixels.items():
                color = plane0 * 2 + plane2 * 8
                color += plane1 * {"white": 4, "red": 5, "blue": 1}[name]
                atlas_x, atlas_y = index % ATLAS_COLUMNS * CELL_WIDTH + x, index // ATLAS_COLUMNS * CELL_HEIGHT + y
                offset = (atlas_y * width + atlas_x) * 4
                image[offset:offset + 4] = bytes((*rgb(palette[color]), 255))
        metadata.append({"id": index, "index": index, "x": index % ATLAS_COLUMNS * CELL_WIDTH,
                         "y": index // ATLAS_COLUMNS * CELL_HEIGHT, "width": CELL_WIDTH,
                         "height": CELL_HEIGHT, "offset_x": 32, "offset_y": 78,
                         "source_address": f"0x{SPRITE_TABLE + relative:05x}",
                         "visible": bool(pixels)})
    png(destination / "belts.png", width, height, belts)
    (destination / "sprites.indices").write_bytes(sprite_indices)
    (destination / "fighter-shadows.bin").write_bytes(memory[0x483A0:0x483A0 + 88 * 80])
    for name, data in atlases.items():
        png(destination / f"sprites-{name}.png", width, height, data)
    return metadata


def recover_ui(memory: bytes, destination: Path, palette: list[int]) -> None:
    """Recover separate fonts, speech tables and direct plane-row object cells."""
    font = bytearray(128 * 20 * 4)
    for glyph in range(48):
        for y in range(5):
            value = memory[0x909E + glyph * 5 + y]
            for x in range(8):
                if value >> (7 - x) & 1:
                    at = (((glyph // 16 * 5 + y) * 128) + glyph % 16 * 8 + x) * 4
                    font[at:at + 4] = bytes((255, 255, 255, 255))
    png(destination / "font.png", 128, 20, font)
    (destination / "font.json").write_text(json.dumps({"glyph_width": 8, "glyph_height": 5,
        "columns": 16, "count": 48, "first_character": 48, "mask": 63,
        "source_address": "0x909e", "space_is_empty": True}, indent=2) + "\n")

    # $7C32 uses a separate 64-cell eight-row font at $8E9E. The five-row
    # font above belongs to speech/text routines, not the scoreboard.
    hud_font = bytearray(128 * 32 * 4)
    for glyph in range(64):
        for y in range(8):
            value = memory[0x8E9E + glyph * 8 + y]
            for x in range(8):
                if value >> (7 - x) & 1:
                    at = (((glyph // 16 * 8 + y) * 128) + glyph % 16 * 8 + x) * 4
                    hud_font[at:at + 4] = bytes((255, 255, 255, 255))
    png(destination / "hud-font.png", 128, 32, hud_font)
    hud_palette = copper_palettes(memory)[0]
    bank_indices = planar_indices(memory, 0x1CA5C)

    def hud_cell(address: int, width: int, height: int) -> bytearray:
        offset = address - 0x1CA5C
        source_y, source_x = offset // 160, offset % 160 * 8
        result = bytearray(width * height * 4)
        for y in range(height):
            for x in range(width):
                index = bank_indices[(source_y + y) * 320 + source_x + x]
                if index:
                    at = (y * width + x) * 4
                    result[at:at + 4] = bytes((*rgb(hud_palette[index]), 255))
        return result

    png(destination / "hud-speed-label.png", 32, 5, hud_cell(0x2317C, 32, 5))
    for speed in range(5):
        address = u32(memory, 0x6C0E + speed * 4)
        png(destination / f"hud-speed-{speed}.png", 32, 5, hud_cell(address, 32, 5))
    for player in range(2):
        address = u32(memory, 0x6CD0 + player * 4)
        png(destination / f"hud-owner-{player}.png", 16, 15, hud_cell(address, 16, 15))
    (destination / "hud.json").write_text(json.dumps({
        "font_source": "0x8e9e", "glyph_width": 8, "glyph_height": 8,
        "glyph_count": 64, "columns": 16,
        "palette": hud_palette,
        "point_columns": list(memory[0x7D38:0x7D3B]),
        "score_columns": list(memory[0x7D3B:0x7D3E]),
        "player_colors": list(memory[0x7CC6:0x7CC9]),
        "empty_point_colors": list(memory[0x7CC9:0x7CCC]),
        "countdown_colors": list(memory[0x6BB8:0x6BBA]),
        "pause_colors": list(memory[0x6B30:0x6B32]),
    }, indent=2) + "\n")

    def cell(address: int, width: int, height: int) -> bytearray:
        # Runtime startup has already deinterleaved the banks before these routines use them.
        # Translate the source plane-row address back to its original word-interleaved bank.
        offset = address - 0x1CA5C
        source_y, source_x = offset // 160, offset % 160 * 8
        indices = planar_indices(memory, 0x1CA5C)
        result = bytearray(width * height * 4)
        for y in range(height):
            for x in range(width):
                if source_y + y >= 200:
                    continue
                index = indices[(source_y + y) * 320 + source_x + x]
                if index:
                    at = (y * width + x) * 4
                    result[at:at + 4] = bytes((*rgb(palette[index]), 255))
        return result

    def indexed_cell(address: int, width: int, height: int) -> bytearray:
        offset = address - 0x1CA5C
        source_y, source_x = offset // 160, offset % 160 * 8
        return bytearray(bank_indices[(source_y + y) * 320 + source_x + x]
                         for y in range(height) for x in range(width))

    # Speech scripts retain their control bytes for player names, line breaks,
    # doubled rows and BCD awards. The Go renderer interprets them natively.
    caption_messages = []
    for pointer_table in (0x8926, 0x1B86E):
        messages = []
        for message in range(40):
            address = u32(memory, pointer_table + message * 4)
            script = memory[address:address + 128]
            messages.append(list(script[:script.index(0x9B) + 1]))
        caption_messages.append(messages)
    (destination / "captions.json").write_text(json.dumps({
        "messages": caption_messages,
        "colors": list(memory[0x89CE:0x89CE + 40]),
        "name_lengths": list(memory[0x8324:0x8327]),
        "name_offsets": list(memory[0x8327:0x832A]),
        "names": [list(memory[0x832A:0x8339]), list(memory[0x1BA0A:0x1BA0A + 15])],
        "name_colors": list(memory[0x8339:0x833C]),
        "row_y": [u32(memory, 0x833C + row * 4) // 160 for row in range(4)],
    }, separators=(",", ":")) + "\n")
    (destination / "caption-tail.indices").write_bytes(indexed_cell(0x22E7E, 16, 7))
    sparkle_cells = bytearray(64 * 5)
    for frame in range(4):
        source = indexed_cell(0x22DC0 + frame * 2, 16, 5)
        for y in range(5):
            sparkle_cells[y * 64 + frame * 16:y * 64 + frame * 16 + 16] = source[y * 16:y * 16 + 16]
    (destination / "caption-sparkles.indices").write_bytes(sparkle_cells)

    # 0xC2C6 selects nine animation columns and eight color rows; the original
    # palette cycle at 0xBCD4 includes the final color index 7.
    ball_atlas = bytearray(144 * 96 * 4)
    for color in range(8):
        for frame in range(9):
            data = cell(0x1E2BC + color * 0x6E0 + frame * 2, 16, 12)
            for y in range(12):
                at = ((color * 12 + y) * 144 + frame * 16) * 4
                ball_atlas[at:at + 64] = data[y * 64:(y + 1) * 64]
    png(destination / "balls.png", 144, 96, ball_atlas)
    png(destination / "ball.png", 16, 12, cell(0x1E2BC + 3 * 0x6E0, 16, 12))
    # 0xCF22 selects four ten-row fragments. Source 0x1DC94 is the first.
    fragment_atlas = bytearray(128 * 40 * 4)
    for variant in range(4):
        for frame in range(8):
            data = cell(0x1DC94 + variant * 0x640 + frame * 2, 16, 10)
            for y in range(10):
                at = ((variant * 10 + y) * 128 + frame * 16) * 4
                fragment_atlas[at:at + 64] = data[y * 64:(y + 1) * 64]
    png(destination / "ball-fragments.png", 128, 40, fragment_atlas)
    bomb_atlas = bytearray(576 * 13 * 4)
    for frame in range(36):
        address = u32(memory, 0xCB60 + frame * 4)
        data = cell(address, 16, 13)
        for y in range(13):
            at = (y * 576 + frame * 16) * 4
            bomb_atlas[at:at + 64] = data[y * 64:(y + 1) * 64]
    png(destination / "bombs.png", 576, 13, bomb_atlas)
    # Reproduce startup $1BF8's plane-row layout before reading the exact
    # addresses selected by the original shield and bomb drawing routines.
    runtime = bytearray(memory)
    for y in range(200):
        at = 0x1CA5C + y * 160
        words = struct.unpack_from(">80H", memory, at)
        struct.pack_into(">80H", runtime, at,
                         *(words[column * 4 + plane] for plane in range(4) for column in range(20)))

    def drawn_cell(address: int, width: int, height: int) -> bytearray:
        # $1BC26 reads four plane words 40 bytes apart and advances 160 bytes
        # per row. Index zero is transparent; a black nonzero index is opaque.
        # Read past the graphics bank when the original routine does so: the
        # alternate neutral shield's final row comes from unmodified code.
        result = bytearray(width * height * 4)
        for y in range(height):
            for x in range(width):
                index = sum(((u16(runtime, address + y * 160 + plane * 40 + x // 16 * 2)
                              >> (15 - x % 16)) & 1) << plane for plane in range(4))
                if index:
                    at = (y * width + x) * 4
                    result[at:at + 4] = bytes((*rgb(palette[index]), 255))
        return result

    # $B9AE..$BA00 chooses type 0/2/1 in that order. Only neutral type 1
    # draws a second 16-pixel board; $1BAFE preserves A0, so it starts at A0+2.
    png(destination / "shield-left.png", 16, 23, drawn_cell(0x225B4, 16, 23))
    png(destination / "shield-right.png", 16, 23, drawn_cell(0x225B6, 16, 23))
    png(destination / "shield-neutral.png", 32, 23, drawn_cell(0x225B8, 32, 23))
    png(destination / "shield-neutral-alternate.png", 32, 23, drawn_cell(0x2399C, 32, 23))
    # $C88A draws this one-row marker for shin kicks and sweeps while $802<3.
    # The same opaque pixels enter the original weapon collision plane.
    png(destination / "kick-marker.png", 16, 1, drawn_cell(0x22DBC, 16, 1))
    for obsolete in ("shield.png", "shield-wide.png", "shield-fist.png", "shield-alternate.png"):
        (destination / obsolete).unlink(missing_ok=True)

    # $D0F8 calls $1BEF0 to clear all four destination planes under a
    # single-plane mask. This opaque black body is separate from the
    # thirteen-row fuse/highlight animation drawn eight pixels above it.
    body = bytearray(16 * 10 * 4)
    for y in range(10):
        mask = u16(runtime, 0x22E7C + y * 160)
        for x in range(16):
            if mask >> (15 - x) & 1:
                body[(y * 16 + x) * 4 + 3] = 255
    png(destination / "bomb-body.png", 16, 10, body)
    # $CF1C/$CE46 draw/test the first plane only, as opaque black fragments.
    fragments = bytearray(128 * 36 * 4)
    for variant in range(4):
        for frame in range(8):
            address = 0x1DC94 + variant * 0x640 + frame * 2
            for y in range(9):
                mask = u16(runtime, address + y * 160)
                for x in range(16):
                    if mask >> (15 - x) & 1:
                        fragments[((variant * 9 + y) * 128 + frame * 16 + x) * 4 + 3] = 255
    png(destination / "bomb-fragments.png", 128, 36, fragments)
    bomb_preview = bytearray(16 * 18 * 4)
    bomb_preview[8 * 16 * 4:] = body
    fuse = drawn_cell(u32(memory, 0xCB60), 16, 13)
    for at in range(0, len(fuse), 4):
        if fuse[at + 3]:
            bomb_preview[at:at + 4] = fuse[at:at + 4]
    png(destination / "bomb.png", 16, 18, bomb_preview)
    png(destination / "shadow.png", 16, 11, cell(0x22E7C, 16, 11))


def recover_scenery(memory: bytes, destination: Path, palettes: list[list[int]]) -> None:
    """Recover the reflection patterns and exact phase/lookup tables for its animator."""
    width, height = 9 * 64, 21
    pixels = bytearray(width * height * 4)
    for pattern in range(9):
        data = memory[0x18BB2 + pattern * 16:0x18BC2 + pattern * 16]
        plane2, plane3 = int.from_bytes(data[:8], "big"), int.from_bytes(data[8:], "big")
        for y in range(height):
            for x in range(64):
                index = 2 + 4 * (plane2 >> (63 - x) & 1) + 8 * (plane3 >> (63 - x) & 1)
                at = (y * width + pattern * 64 + x) * 4
                pixels[at:at + 4] = bytes((*rgb(palettes[78 + y][index]), 255))
    png(destination / "reflection-rows.png", width, height, pixels)
    reflection = {"x": 128, "y": 78, "width": 64, "height": 21, "pattern_width": 64,
        "pattern_count": 9, "update_interval": 2, "phase_mask": 31, "phase_step": 31,
        "initial_phases": list(memory[0x18A04:0x18A19]),
        "waveform": list(memory[0x18A60:0x18A80]),
        "row_offsets": list(memory[0x18A92:0x18AA7]),
        "pattern_lookup": list(memory[0x18AAA:0x18BAA]),
        "jitter": list(memory[0x19190:0x191A0]),
        "source_routines": ["0x18c7e", "0x18ce2", "0x191a0"]}
    (destination / "reflection.json").write_text(json.dumps(reflection, indent=2) + "\n")
    edges = []
    edge_pixels = bytearray(72 * 18 * 4)
    background = planar_indices(memory, 0x2C5CA)
    left_masks = list(memory[0x18A80:0x18A89])
    right_masks = list(memory[0x18A89:0x18A92])
    for point in range(18):
        offset, phase_index = u32(memory, 0x18D56 + point * 4), u16(memory, 0x18D9E + point * 2)
        x, y = (offset % 160) * 8, offset // 160
        masks = left_masks if phase_index < 21 else right_masks
        edges.append({"x": x, "y": y, "phase_index": phase_index % 21, "right_mask": phase_index >= 21})
        for pattern in range(9):
            for bit in range(8):
                color = 6 if masks[pattern] >> (7 - bit) & 1 else background[y * 320 + x + bit] & 9
                at = (point * 72 + pattern * 8 + bit) * 4
                edge_pixels[at:at + 4] = bytes((*rgb(palettes[y][color]), 255))
    png(destination / "water-edge.png", 72, 18, edge_pixels)
    (destination / "water-edge.json").write_text(json.dumps(edges, indent=2) + "\n")
    bank = planar_indices(memory, 0x1CA5C)
    bird_indices = bytearray(320 * 28)
    for y in range(28):
        bird_indices[y * 320:(y + 1) * 320] = bank[y * 320:(y + 1) * 320]
    (destination / "ambient-birds.indices").write_bytes(bird_indices)
    fish = bytearray(192 * 4)
    for y in range(4):
        fish[y * 192:(y + 1) * 192] = bank[(31 + y) * 320:(31 + y) * 320 + 192]
    (destination / "ambient-fish.indices").write_bytes(fish)
    ambient = {"dispatch_routine": "0x19214", "spawn_routine": "0x1922c",
        "mode_handlers": [f"0x{u32(memory, 0x191F4 + index * 4):x}" for index in range(8)],
        "object_types": list(memory[0x19366:0x1936A]),
        "default_delays": list(memory[0x1936A:0x1936E]),
        "x_positions": list(memory[0x19376:0x19396]),
        "y_positions": list(memory[0x19396:0x193B6]),
        "palette_indices": list(memory[0x193B6:0x193D6]),
        "jumping_fish": {"source": "0x1ca5c", "frame_width": 16, "frame_height": 14,
            "frames": 20, "directions": 2, "first_direction_y": 0, "second_direction_y": 14,
            "routine": "0x19a04", "advance_delay": [3, 4]},
        "ripples": {"source": "0x1ddbc", "frame_width": 16, "frame_height": 4,
            "frames": 12, "routine": "0x1992c", "advance_interval": 4, "x_step": 3},
        "bird": {"source": "0x1944a", "frame_width": 16, "frame_height": 5,
            "frames": 4, "variants": 4, "routine": "0x193e6", "advance_interval": 4},
        "forced_state_flags": {"0x18912": 6, "0x18914": 4, "0x18916": 1, "0x18918": 7},
        "source_key_sequences": {"0x999e2e": "0x18912", "0xa1979f23": "0x18914",
            "0xb0979320": "0x18916", "0x99929397": "0x18918"}}
    (destination / "ambient.json").write_text(json.dumps(ambient, indent=2) + "\n")
    (destination / "palettes.json").write_text(json.dumps(palettes, separators=(",", ":")) + "\n")


def recover_bonus_controls(memory: bytes, destination: Path) -> None:
    """Recover the seven-state shield controller and its original pose/position tables."""
    states = []
    for state in range(7):
        states.append({"state": state, "shield_type": memory[0xC50D + state],
            "shield_x_byte": memory[0xC514 + state] & 248,
            "fighter_x_byte": memory[0xC51B + state] & 248,
            "shield_y": 96 + memory[0xC522 + state],
            "facing": memory[0xC529 + state], "frame": memory[0xC530 + state]})
    controls = {"routine": "0xc3ec", "initial_state": 3, "neutral_direction": 15,
        "joystick_directions": list(memory[0xC4BE:0xC4CE]),
        "transitions": [list(memory[0xC4CE + state * 9:0xC4CE + (state + 1) * 9]) for state in range(7)],
        "states": states,
        "notes": "Values 128 and 130 in the transition table turn to the opposite facing and retain height; active-low neutral input is 15 and retains initial state 3."}
    (destination / "bonus-controls.json").write_text(json.dumps(controls, indent=2) + "\n")
    spawn = {"ball": {"routine": "0xbf3e", "accumulator_bits": 8, "accumulator_target": 44,
            "initial_increment": 128, "increment_per_launch": 8,
            "level_increment_limits": list(memory[0xC134:0xC13C]),
            "heights": list(memory[0xC124:0xC12C]), "initial_wait": 100, "goal": 64},
        "bomb": {"routine": "0xc99e", "initial_interval": 25,
            "difficulty_intervals": list(memory[0xC5A4:0xC5AC]),
            "interval_decrement": 8, "minimum_interval": 8,
            "first_positions": list(memory[0xC994:0xC99A]),
            "position_multiplier": 8, "position_offset": 16,
            "initial_fuse": 48, "fuse_decrement": 2, "fuse_zero_after": 27,
            "initial_wait": 150, "goal": u16(memory, 0xC9A0)}}
    (destination / "bonus-spawn.json").write_text(json.dumps(spawn, indent=2) + "\n")


def recover_combat(memory: bytes, destination: Path) -> None:
    """Export strike scoring and knockdown tables independently identified in the code."""
    def bcd(value: int) -> int:
        return (value >> 4) * 10 + (value & 15)
    records = []
    for index in range(10):
        records.append({"frame": memory[0x74B0 + index],
            "front_score": bcd(memory[0x74D9 + index]) * 100,
            "rear_score": bcd(memory[0x74E3 + index]) * 100,
            "front_knockdown": memory[0x7501 + index],
            "rear_knockdown": memory[0x750B + index]})
    (destination / "combat-tables.json").write_text(json.dumps(records, indent=2) + "\n")
    def bcd_word(value: int) -> int:
        return sum(((value >> (digit * 4)) & 15) * (10 ** digit) for digit in range(4))
    labels = [memory[0xAA12 + tier * 8:0xAA1A + tier * 8].decode("ascii") for tier in range(24)]
    belts = {"routine": "0xa9bc", "thresholds": [bcd_word(u16(memory, 0xA9E4 + tier * 2)) * 100 for tier in range(23)],
        "source_labels": labels, "labels": [label.replace("<", " ").strip() for label in labels],
        "white_colors": [u16(memory, 0xAAD2 + tier * 2) for tier in range(24)],
        "red_colors": [u16(memory, 0xAB02 + tier * 2) for tier in range(24)],
        "computer_colors": {"white": 0xC00, "red": 0xE6E, "blue": 0xCC0}}
    (destination / "belt-tables.json").write_text(json.dumps(belts, indent=2) + "\n")


def extract(source: Path, destination: Path) -> dict:
    """Recover the executable, display banks, sprite frames, tables and PCM effects."""
    disk = source.read_bytes()
    digest = hashlib.sha256(disk).hexdigest()
    if digest != DISK_SHA256:
        raise ValueError("This extractor is verified only for the supplied SIGN IK+ ADF")
    destination.mkdir(parents=True, exist_ok=True)
    hunk = read_ofs_file(disk, "ik+++")
    code = hunk_code(hunk)
    game = bytekiller(code)
    memory = bytes(GAME_BASE) + game
    (destination / "ikplus.hunk").write_bytes(hunk)
    (destination / "game.bin").write_bytes(game)
    palettes = copper_palettes(memory)
    png(destination / "background.png", 320, 200, color_image(planar_indices(memory, 0x2C5CA), palettes))
    (destination / "background.indices").write_bytes(planar_indices(memory, 0x2C5CA))
    png(destination / "graphics-bank.png", 320, 200, color_image(planar_indices(memory, 0x1CA5C), palettes))
    png(destination / "title-bank.png", 320, 200, color_image(planar_indices(memory, 0x248CA), palettes))
    # 0x5974 copies the left 160x120 logo to (80,40); 0x2596 removes plane 3.
    title_indices, title = planar_indices(memory, 0x248CA), bytearray(320 * 200 * 4)
    menu_palette = list(struct.unpack_from(">16H", memory, 0x2A3A))
    for y in range(200):
        for x in range(320):
            color = 0
            if 80 <= x < 240 and 40 <= y < 160:
                color = title_indices[(y - 40) * 320 + x - 80] & 7
            at = (y * 320 + x) * 4
            title[at:at + 4] = bytes((*rgb(menu_palette[color]), 255))
    png(destination / "title.png", 320, 200, title)
    recover_scenery(memory, destination, palettes)
    recover_combat(memory, destination)
    recover_bonus_controls(memory, destination)
    recover_ui(memory, destination, palettes[140])
    sprites = recover_sprites(memory, destination, palettes[140])
    (destination / "sprites.json").write_text(json.dumps(sprites, indent=2) + "\n")
    starts = list(memory[0x9E0A:0x9E34])
    animations = []
    for action, (start, end) in enumerate(zip(starts, starts[1:])):
        animations.append({"action": action, "start": start, "end": end,
                           "frames": list(memory[0x9E34 + start:0x9E34 + end]),
                           "x_deltas": [value if value < 128 else value - 256 for value in memory[0x9F28 + start:0x9F28 + end]],
                           "flags": list(memory[0xA01C + start:0xA01C + end])})
    (destination / "animations.json").write_text(json.dumps(animations, indent=2) + "\n")
    sounds = []
    for index in range(18):
        start = u32(memory, 0x342CA + index * 4)
        end = u32(memory, 0x3434A + index * 4)
        pcm = memory[0x343CA + start:0x343CA + end]
        # Paula consumes signed 8-bit PCM. A WAV stores 8-bit PCM unsigned.
        with wave.open(str(destination / f"sound-{index:02d}.wav"), "wb") as audio:
            audio.setnchannels(1)
            audio.setsampwidth(1)
            audio.setframerate(round(3546895 / 358))
            audio.writeframes(bytes(value ^ 128 for value in pcm))
        sounds.append({"index": index, "source_address": f"0x{0x343CA + start:05x}",
                       "length": len(pcm), "period": 358, "sample_rate": round(3546895 / 358)})
    (destination / "sounds.json").write_text(json.dumps(sounds, indent=2) + "\n")
    sound_events = []
    for index, frame in enumerate(memory[0x145C:0x146D]):
        event = {"frame": frame}
        for name, base in (("normal", 0x146D), ("normal_delay", 0x147E),
                           ("normal_delayed", 0x148F), ("hit", 0x14A0),
                           ("hit_delay", 0x14B1), ("hit_delayed", 0x14C2),
                           ("recovering", 0x14D3)):
            value = memory[base + index]
            event[name] = value if value < 128 else -1
        sound_events.append(event)
    (destination / "sound-events.json").write_text(json.dumps(sound_events, indent=2) + "\n")
    generated_names = {"ikplus.hunk", "game.bin", "background.png", "graphics-bank.png", "title-bank.png",
                       "title.png", "sprites.json", "animations.json", "sounds.json", "sound-events.json",
                       "font.png", "font.json", "balls.png", "ball.png", "ball-fragments.png", "bombs.png", "bomb.png", "bomb-body.png", "bomb-fragments.png",
                       "shield-left.png", "shield-right.png", "shield-neutral.png", "shield-neutral-alternate.png", "kick-marker.png", "shadow.png", "belts.png",
                       "reflection-rows.png", "reflection.json", "combat-tables.json", "bonus-controls.json",
                       "water-edge.png", "water-edge.json", "bonus-spawn.json",
                       "ambient-birds.indices", "ambient-fish.indices", "ambient.json", "palettes.json", "belt-tables.json"}
    generated_names.update({"hud-font.png", "hud.json", "hud-speed-label.png", "hud-owner-0.png", "hud-owner-1.png"})
    generated_names.update(f"hud-speed-{speed}.png" for speed in range(5))
    generated_names.update({"captions.json", "caption-tail.indices", "caption-sparkles.indices"})
    generated_names.update({"background.indices", "sprites.indices", "fighter-shadows.bin"})
    generated_names.update(f"sprites-{name}.png" for name in ("white", "red", "blue"))
    generated_names.update(f"sound-{index:02d}.wav" for index in range(18))
    manifest = {"source": source.name, "source_sha256": digest, "game_base": "0x600",
                "game_length": len(game), "loader_length": len(hunk), "bytekiller_checksum": 0,
                "display": {"width": 320, "height": 200, "ground_y": 194},
                "sprite_count": 88, "animation_count": len(animations), "sound_count": len(sounds),
                "files": {path.name: hashlib.sha256(path.read_bytes()).hexdigest()
                          for path in (destination / name for name in sorted(generated_names))}}
    (destination / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    return manifest


def main() -> None:
    """Use the project's reference disk by default or accept explicit paths."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--adf", type=Path,
                        default=Path(".cache/reference/disks") /
                        "IK+ (1988-12-20)(System 3)(de-en)[cr][h Sign].adf")
    parser.add_argument("--output", type=Path, default=Path("assets/original"))
    args = parser.parse_args()
    source = args.adf
    if not source.is_file():
        parser.error(f"Reference disk is missing: {source}. Supply a local disk with --adf PATH.")
    manifest = extract(source, args.output)
    print(f"Recovered {manifest['game_length']} game bytes, {manifest['sprite_count']} sprites, "
          f"{manifest['animation_count']} animation sequences and {manifest['sound_count']} PCM effects.")


if __name__ == "__main__":
    main()
