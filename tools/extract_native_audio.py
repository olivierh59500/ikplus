#!/usr/bin/env python3
"""Extract native music commands, instruments and PCM without runtime addresses."""

import argparse
import hashlib
import json
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parent.parent


class AudioExtractor:
    def __init__(self):
        self.raw = (ROOT / "assets/original/game.bin").read_bytes()
        self.source = bytes(0x600) + self.raw
        self.patterns = [[]]
        self.pattern_sources = [[]]
        self.orders = [[]]
        self.order_sources = [0]
        self.envelopes = [[]]
        self.envelope_sources = [0]
        self.instruments = [{}]
        self.instrument_sources = [0]
        self.pattern_ids = {}
        self.order_ids = {}
        self.envelope_ids = {}
        self.instrument_ids = {}
        self.samples = bytearray(4)
        self.sample_offsets = {0: 0}
        self.sample_sources = [(0, 0, 0)]

    def word(self, at):
        return struct.unpack_from(">H", self.source, at)[0]

    def long(self, at):
        return struct.unpack_from(">I", self.source, at)[0]

    def sample(self, address, words):
        aligned = address & ~1
        if address == 0:
            return 0
        size = words * 2 if words else 131072
        if address in self.sample_offsets:
            offset = self.sample_offsets[address]
            old = next(item[2] for item in self.sample_sources if item[0] == offset)
            if size > old:
                raise ValueError(f"sample {address:x} has inconsistent spans: {old}/{size}")
            return offset
        if aligned < 0x600 or aligned + size > len(self.source):
            raise ValueError(f"sample {address:x}+{size} is outside the source")
        offset = len(self.samples)
        self.sample_offsets[address] = offset
        self.samples.extend(self.source[aligned:aligned + size])
        self.sample_sources.append((offset, address, size))
        return offset

    def envelope(self, address):
        if address == 0:
            return 0
        if address in self.envelope_ids:
            return self.envelope_ids[address]
        index = len(self.envelopes)
        self.envelope_ids[address] = index
        self.envelope_sources.append(address)
        values = []
        at = address
        while self.word(at) != 255:
            values.append(self.word(at))
            at += 2
            if len(values) > 4096:
                raise ValueError("unterminated volume envelope")
        self.envelopes.append(values)
        return index

    def instrument(self, address):
        if address in self.instrument_ids:
            return self.instrument_ids[address]
        index = len(self.instruments)
        self.instrument_ids[address] = index
        self.instrument_sources.append(address)
        mode = self.word(address)
        if mode == 0:
            words = self.word(address + 6)
            instrument = dict(mode=mode, attack=dict(offset=self.sample(self.long(address + 2), words), length=words))
        elif mode == 1:
            attack_words, loop_words = self.word(address + 6), self.word(address + 12)
            instrument = dict(mode=mode,
                              attack=dict(offset=self.sample(self.long(address + 2), attack_words), length=attack_words),
                              loop=dict(offset=self.sample(self.long(address + 8), loop_words), length=loop_words))
        else:
            raise ValueError(f"unsupported instrument mode {mode}")
        self.instruments.append(instrument)
        return index

    def order(self, address):
        if address == 0:
            return 0
        if address in self.order_ids:
            return self.order_ids[address]
        index = len(self.orders)
        self.order_ids[address] = index
        self.order_sources.append(address)
        self.orders.append([])
        while self.long(address) != 0:
            self.orders[index].append(self.pattern(self.long(address)))
            address += 4
            if len(self.orders[index]) > 1024:
                raise ValueError("unterminated song order")
        return index

    def pattern(self, address):
        if address == 0:
            return 0
        if address in self.pattern_ids:
            return self.pattern_ids[address]
        index = len(self.patterns)
        self.pattern_ids[address] = index
        self.patterns.append([])
        self.pattern_sources.append([])
        for _ in range(4096):
            self.pattern_sources[index].append(address)
            value = self.word(address)
            address += 2
            command = {}
            if value > 100:
                command = dict(kind="note", period=value, duration=self.word(address))
                address += 2
            elif value == 0:
                command = dict(kind="stop")
            elif value == 4:
                command = dict(kind="instrument", instrument=self.instrument(self.long(address)))
                address += 4
            elif value == 8:
                command = dict(kind="pattern")
            elif value == 12:
                command = dict(kind="envelope", envelope=self.envelope(self.long(address)))
                address += 4
            elif value == 16:
                mode, step, target, delay = (self.word(address + i * 2) for i in range(4))
                command = dict(kind="slide", slide=mode, step=step, target=target, delay=delay)
                address += 8
            elif value == 20:
                delay, interval, fall, rise, extra, progress = (self.word(address + i * 2) for i in range(6))
                command = dict(kind="vibrato", delay=delay, interval=interval, fall=fall, rise=rise, extra=extra, progress=progress)
                address += 12
            elif value == 24:
                command = dict(kind="slide-off")
            elif value == 28:
                command = dict(kind="vibrato-off")
            elif value == 32:
                command = dict(kind="rest", duration=self.word(address))
                address += 2
            elif value in (36, 40):
                command = dict(kind="filter", enabled=value == 36)
            else:
                raise ValueError(f"unsupported music command {value} at {address - 2:x}")
            self.patterns[index].append(command)
            if command["kind"] in ("pattern", "stop"):
                self.pattern_sources[index].append(address)
                return index
        raise ValueError("unterminated music pattern")

    def extract(self):
        silent = self.sample(0x12C6, 1)
        silent_envelope = self.envelope(0xD4C2)
        tracks = []
        for index in range(4):
            at = 0xD5D4 + index * 62
            if any(self.word(at + offset) for offset in list(range(0, 22, 2)) + list(range(34, 62, 2))):
                raise ValueError("unexpected nonzero initial music controls")
            tracks.append(dict(sequence=dict(id=self.pattern(self.long(at + 22))),
                               order=dict(id=self.order(self.long(at + 26))),
                               repeat=dict(id=self.order(self.long(at + 30)))))
        effects = []
        for index in range(18):
            start, end = self.long(0x342CA + index * 4), self.long(0x3434A + index * 4)
            length = ((end - start) >> 1) & 0x7FFE
            effects.append(dict(offset=self.sample((0x343CA + start) & ~1, length), length=length))
        trigger_names = ["frames", "ordinary", "ordinaryDelays", "ordinaryDelayed", "hits", "hitDelays", "hitDelayed", "down"]
        triggers = dict(zip(trigger_names, [list(self.source[0x145C + i * 17:0x145C + (i + 1) * 17]) for i in range(8)]))
        native = dict(version=1, sourceSHA256=hashlib.sha256(self.raw).hexdigest(), tracks=tracks,
                      patterns=self.patterns, orders=self.orders, envelopes=self.envelopes,
                      instruments=self.instruments, effects=effects, triggers=triggers,
                      silentOffset=silent, silentEnvelope=silent_envelope,
                      afterTracks=self.word(0xD6CC))
        reference = dict(patterns=self.pattern_sources, orders=self.order_sources,
                         envelopes=self.envelope_sources, instruments=self.instrument_sources,
                         samples=self.sample_sources, sourceSHA256=native["sourceSHA256"])
        # Pointer-shaped values are immutable vocabulary for the recovered random
        # mixer only. Runtime cursors and samples use native indices and offsets.
        fields = [
            ("disabled", 2), ("duration", 2), ("mode", 2), ("period", 2),
            ("slide", 2), ("slide-step", 2), ("slide-target", 2), ("slide-delay", 2),
            ("vibrato", 2), ("envelope", 4), ("sequence", 4), ("order", 4),
            ("repeat", 4), ("template", 4), ("attack", 4), ("attack-length", 2),
            ("loop", 4), ("loop-length", 2), ("vibrato-delay", 2),
            ("vibrato-interval", 2), ("vibrato-fall", 2), ("vibrato-rise", 2),
            ("vibrato-extra", 2), ("progress", 2),
        ]
        requests = [dict(voice=voice, field=name, shift=(width - 1 - byte) * 8)
                    for voice in range(4) for name, width in fields for byte in range(width)]
        assert len(requests) == 248
        native["entropy"] = dict(
            requests=requests,
            patterns=[[list(struct.pack(">I", value)) for value in values] for values in self.pattern_sources],
            orders=[[list(struct.pack(">I", address + i * 4)) for i in range(len(order) + 1)]
                    for address, order in zip(self.order_sources, self.orders)],
            envelopes=[[list(struct.pack(">I", address + i * 2)) for i in range(len(values) + 1)]
                       for address, values in zip(self.envelope_sources, self.envelopes)],
            samples=[dict(offset=offset, value=list(struct.pack(">I", address))) for offset, address, _ in self.sample_sources],
        )
        return native, bytes(self.samples), reference


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    options = parser.parse_args()
    native, samples, reference = AudioExtractor().extract()
    outputs = {
        "assets/original/audio.json": json.dumps(native, separators=(",", ":")).encode() + b"\n",
        "assets/original/audio.pcm": samples,
        "internal/referenceaudio/testdata/native-audio-layout.json": json.dumps(reference, separators=(",", ":")).encode() + b"\n",
    }
    for name, data in outputs.items():
        path = ROOT / name
        if options.check:
            if not path.exists() or path.read_bytes() != data:
                parser.error(f"extracted audio differs: {name}")
        else:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
    print(f"Native audio: {len(native['patterns']) - 1} patterns, {len(native['instruments']) - 1} instruments, {len(samples)} PCM bytes.")


if __name__ == "__main__":
    main()
