#!/usr/bin/env python3
"""Compose a subtitled IK+ pixel-art showcase from real game recordings.

Pillow renders the surrounding editorial graphics, so FFmpeg does not need
libass or drawtext. The native 320-pixel scene stays untouched inside a 3x
nearest-neighbor viewport. Source frame indices and rational PAL timestamps
keep video and sampled audio on the same clock.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import subprocess
import wave
from fractions import Fraction
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[2]
WIDTH, HEIGHT = 1920, 1080
GAME_BOX = (480, 120, 1440, 888)
NAVY = (10, 15, 27)
PANEL = (17, 24, 40)
GOLD = (234, 197, 103)
WHITE = (240, 242, 247)
MUTED = (150, 164, 188)


def frame_rate(timeline: dict, source: bool = False) -> Fraction:
    value = timeline.get("source_fps", timeline["fps"]) if source else timeline["fps"]
    return Fraction(str(value))


def source_canvas(timeline: dict) -> tuple[int, int]:
    """Support window captures and the shared compositor's exact arena view."""
    size = tuple(timeline.get("source_canvas", (320, 256)))
    if size not in ((320, 256), (320, 200)):
        raise ValueError("Source canvas must be 320 x 256 or 320 x 200.")
    return size


def clip_frames(clip: dict, rate: Fraction) -> int:
    return int(clip.get("frames", round(clip["duration"] * float(rate))))


def run(command: list[str]) -> None:
    """Fail immediately if a media tool cannot produce the requested artifact."""
    print(" ".join(str(part) for part in command), flush=True)
    subprocess.run(command, check=True)


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    """Find a conventional sans-serif font on macOS or Linux."""
    name = "Arial Bold.ttf" if bold else "Arial.ttf"
    linux = "DejaVuSans-Bold.ttf" if bold else "DejaVuSans.ttf"
    candidates = [
        Path("/System/Library/Fonts/Supplemental") / name,
        Path("/usr/share/fonts/truetype/dejavu") / linux,
        Path("/usr/share/fonts/TTF") / linux,
    ]
    for candidate in candidates:
        if candidate.exists():
            return ImageFont.truetype(str(candidate), size)
    raise RuntimeError("Install Arial or DejaVu Sans to render the captions.")


def lines_for(draw: ImageDraw.ImageDraw, text: str, face, width: int) -> list[str]:
    """Wrap prose without changing explicit line breaks."""
    lines = []
    for paragraph in text.split("\n"):
        current = ""
        for word in paragraph.split():
            trial = f"{current} {word}".strip()
            if current and draw.textlength(trial, font=face) > width:
                lines.append(current)
                current = word
            else:
                current = trial
        lines.append(current)
    return lines


def wrapped(draw, text, xy, face, fill, width, leading=1.3) -> int:
    x, y = xy
    for line in lines_for(draw, text, face, width):
        draw.text((x, y), line, fill=fill, font=face)
        y += round(face.size * leading)
    return y


def original_logo() -> Image.Image:
    """Reuse the recovered red mark, making only its black backing transparent."""
    logo = Image.open(ROOT / "assets/original/title.png").convert("RGBA")
    pixels = logo.load()
    for y in range(logo.height):
        for x in range(logo.width):
            r, g, b, _ = pixels[x, y]
            if r + g + b == 0:
                pixels[x, y] = (0, 0, 0, 0)
    logo = logo.crop(logo.getbbox())
    target_height = 66
    return logo.resize(
        (round(logo.width * target_height / logo.height), target_height),
        Image.Resampling.NEAREST,
    )


def active(items: list[dict], time: float, rate: Fraction) -> dict | None:
    tick = round(time * float(rate))
    return next((item for item in items if round(item["start"] * float(rate)) <= tick < round(item["end"] * float(rate))), None)


def graphic(timeline: dict, time: float) -> Image.Image:
    """Keep all titles, chapter notes and subtitles outside the game canvas."""
    image = Image.new("RGB", (WIDTH, HEIGHT), NAVY)
    draw = ImageDraw.Draw(image)
    draw.line((64, 98, WIDTH - 64, 98), GOLD, 2)
    logo = original_logo()
    image.paste(logo, (68, 20), logo)
    draw.text((180, 25), "INTERNATIONAL KARATE +", font=font(36, True), fill=WHITE)
    draw.text((182, 67), "THE AMIGA CLASSIC / A SKILLED PLAYER SHOWCASE", font=font(18), fill=MUTED)
    draw.text((1855, 37), "A SKILLED PLAYER SHOWCASE", font=font(21, True), fill=GOLD, anchor="ra")
    draw.rectangle((GAME_BOX[0] - 2, GAME_BOX[1] - 2, GAME_BOX[2] + 1, GAME_BOX[3] + 1), outline=(52, 65, 87), width=2)
    draw.rectangle(GAME_BOX, fill=(0, 0, 0))

    rate = frame_rate(timeline)
    chapter = active(timeline.get("chapters", []), time, rate)
    draw.text((68, 166), "THE JOURNEY", font=font(18, True), fill=GOLD)
    draw.line((68, 202, 394, 202), (52, 65, 87), 2)
    if chapter:
        y = wrapped(draw, chapter.get("title", ""), (68, 226), font(35, True), WHITE, 326, 1.2)
        wrapped(draw, chapter.get("detail", ""), (68, y + 25), font(24), MUTED, 326, 1.4)
    else:
        wrapped(draw, "An Amiga classic, rebuilt in Go.", (68, 226), font(35, True), WHITE, 326, 1.2)

    draw.text((1524, 166), "FOLLOW THE WHITE SUIT", font=font(18, True), fill=GOLD)
    draw.line((1524, 202, 1850, 202), (52, 65, 87), 2)
    y = wrapped(draw, "WHITE", (1524, 226), font(48, True), WHITE, 326)
    y = wrapped(draw, "Skilled simulated player", (1524, y + 15), font(25), MUTED, 326, 1.4)
    draw.rectangle((1524, y + 32, 1554, y + 38), fill=(238, 51, 85))
    draw.rectangle((1570, y + 32, 1600, y + 38), fill=(68, 68, 238))
    y = wrapped(draw, "Two computer opponents", (1524, y + 60), font(25), MUTED, 326, 1.4)
    draw.line((1524, y + 24, 1850, y + 24), (52, 65, 87), 2)
    wrapped(draw, "PAL / NORMAL SPEED\nChronological highlights", (1524, y + 48), font(22), GOLD, 326, 1.5)
    outro = time >= timeline["chapters"][-1]["start"]
    draw.text((68, 821), "GO + EBITENGINE" if outro else "ORIGINAL PIXEL ART", font=font(19, True), fill=WHITE)
    draw.text((68, 853), "Desktop / Android" if outro else "Original game audio", font=font(22), fill=MUTED)
    draw.text((1850, 850), "SYSTEM 3 / IK+", font=font(18), fill=MUTED, anchor="ra")

    draw.line((480, 919, 1440, 919), (52, 65, 87), 1)
    cue = active(timeline.get("cues", []), time, rate)
    if cue:
        subtitle_lines = lines_for(draw, cue["text"], font(34), 1640)
        if len(subtitle_lines) > 2:
            raise ValueError(f"Caption needs more than two lines: {cue['text']}")
        y = 957 if len(subtitle_lines) == 1 else 936
        for line in subtitle_lines:
            draw.text((960, y), line, font=font(34), fill=WHITE, anchor="ma")
            y += 47
    draw.text((1850, 1033), "IK+ / NATIVE PORT", font=font(15), fill=(101, 117, 143), anchor="ra")
    return image


def quote_concat(path: Path) -> str:
    return "'" + str(path.resolve()).replace("'", "'\\''") + "'"


def write_graphics(timeline: dict, work: Path) -> Path:
    """Use sparse stills so an FFmpeg build without text filters is sufficient."""
    rate = frame_rate(timeline)
    boundaries = {0.0, float(timeline["duration"])}
    for item in timeline.get("cues", []) + timeline.get("chapters", []):
        boundaries.update((round(float(item["start"]) * float(rate)) / float(rate), round(float(item["end"]) * float(rate)) / float(rate)))
    boundaries = sorted(boundaries)
    manifest = work / "graphics.ffconcat"
    entries = ["ffconcat version 1.0"]
    for index, (start, end) in enumerate(zip(boundaries, boundaries[1:])):
        path = work / f"graphics-{index:03d}.png"
        graphic(timeline, start + 0.0001).save(path)
        entries.extend((f"file {quote_concat(path)}", f"option framerate {rate}", f"duration {end - start:.9f}"))
    entries.extend((f"file {quote_concat(path)}", f"option framerate {rate}"))
    manifest.write_text("\n".join(entries) + "\n", encoding="utf-8")
    return manifest


def srt_time(seconds: float) -> str:
    milliseconds = round(seconds * 1000)
    hours, remainder = divmod(milliseconds, 3_600_000)
    minutes, remainder = divmod(remainder, 60_000)
    seconds, milliseconds = divmod(remainder, 1000)
    return f"{hours:02}:{minutes:02}:{seconds:02},{milliseconds:03}"


def write_subtitles(timeline: dict, path: Path) -> None:
    entries = []
    for index, cue in enumerate(timeline.get("cues", []), 1):
        entries.append(f"{index}\n{srt_time(cue['start'])} --> {srt_time(cue['end'])}\n{cue['text']}\n")
    path.write_text("\n".join(entries), encoding="utf-8")


def metadata_value(text: str) -> str:
    return text.replace("\\", "\\\\").replace("=", "\\=").replace(";", "\\;").replace("#", "\\#").replace("\n", " ")


def write_metadata(timeline: dict, path: Path) -> None:
    rows = [";FFMETADATA1", "title=IK+ - A skilled player showcase", "comment=Native Go/Ebitengine footage. Tool-assisted player. Chronological highlights with editorial cuts."]
    for chapter in timeline.get("chapters", []):
        rows.extend(("[CHAPTER]", "TIMEBASE=1/1000", f"START={round(chapter['start'] * 1000)}", f"END={round(chapter['end'] * 1000)}", "title=" + metadata_value(chapter["title"])))
    path.write_text("\n".join(rows) + "\n", encoding="utf-8")


def validate(timeline: dict) -> None:
    source_canvas(timeline)
    rate, source_rate = frame_rate(timeline), frame_rate(timeline, source=True)
    if rate <= 0 or source_rate <= 0:
        raise ValueError("Frame rates must be positive.")
    frames = sum(clip_frames(clip, rate) for clip in timeline["clips"])
    if abs(float(Fraction(frames, 1) / rate) - timeline["duration"]) > 0.000001:
        raise ValueError("Clip frame counts do not match the requested duration.")
    for clip in timeline["clips"]:
        if "source_tick_start" in clip:
            start = round(clip["source_start"] * float(source_rate))
            end = start + round(clip["duration"] * float(source_rate))
            if (start, end) != (clip["source_tick_start"], clip["source_tick_end"]):
                raise ValueError(f"Source seconds and source tick interval disagree: {clip}")
    for item in timeline.get("cues", []) + timeline.get("chapters", []):
        if not 0 <= item["start"] < item["end"] <= timeline["duration"]:
            raise ValueError(f"Invalid timeline interval: {item}")


def verify_sources(timeline: dict, args) -> None:
    """Reject mismatched Android canvases or incomplete source recordings."""
    for field, path in (("source_video_sha256", args.video), ("source_audio_sha256", args.audio)):
        expected = timeline.get(field)
        if expected:
            digest = hashlib.sha256()
            with path.open("rb") as source:
                for chunk in iter(lambda: source.read(1 << 20), b""):
                    digest.update(chunk)
            if digest.hexdigest() != expected:
                raise ValueError(f"Source hash differs from the selected recording: {path}")
    probe = subprocess.check_output([
        str(Path(args.ffmpeg).with_name("ffprobe")), "-v", "error",
        "-select_streams", "v:0", "-count_frames", "-show_entries", "stream=width,height,r_frame_rate,nb_read_frames",
        "-show_entries", "format=duration", "-of", "json", str(args.video),
    ], text=True)
    data = json.loads(probe)
    stream = data["streams"][0]
    source_rate = frame_rate(timeline, source=True)
    # Matroska stores integer-nanosecond default frame durations, so ffprobe
    # can report a nearby rational rather than the exact requested PAL ratio.
    canvas = source_canvas(timeline)
    if (stream["width"], stream["height"]) != canvas or abs(float(Fraction(stream["r_frame_rate"]) - source_rate)) > 0.00001:
        raise ValueError(f"Expected the native {canvas[0]} x {canvas[1]} canvas at {source_rate} fps: {stream}")
    if "source_frames" in timeline and int(stream["nb_read_frames"]) != timeline["source_frames"]:
        raise ValueError("Source recording frame count differs from the input run.")
    source_end = max(clip.get("source_start", 0) + (float(1 / source_rate) if clip.get("hold") else clip["duration"]) for clip in timeline["clips"] if clip.get("kind") != "title")
    if float(data["format"]["duration"]) + 0.001 < source_end:
        raise ValueError("The source video ends before the final selected gameplay frame.")
    music_end = max((clip.get("audio_start", 0 if clip.get("kind") == "title" else 8) + clip["duration"] for clip in timeline["clips"] if clip.get("kind") == "title" or clip.get("hold")), default=0)
    for path, minimum in ((args.audio, source_end), (args.music, music_end)):
        with wave.open(str(path), "rb") as audio:
            if (audio.getframerate(), audio.getnchannels(), audio.getsampwidth()) != (44100, 2, 2):
                raise ValueError(f"Expected 44.1 kHz stereo PCM16: {path}")
            if audio.getnframes() < round(minimum * 44100):
                raise ValueError(f"The source audio is incomplete: {path}")
    with Image.open(args.title) as still:
        if still.size != canvas:
            raise ValueError(f"The opening title screenshot must use the native {canvas[0]} x {canvas[1]} canvas.")


def segment(clip: dict, index: int, args, work: Path, timeline: dict) -> Path:
    """Render small lossless intermediates rather than buffering the whole source."""
    result = work / f"clip-{index:02d}.mkv"
    rate, source_rate = frame_rate(timeline), frame_rate(timeline, source=True)
    frames = clip_frames(clip, rate)
    duration = float(Fraction(frames, 1) / rate)
    start_tick = clip.get("source_tick_start", round(clip.get("source_start", 0) * float(source_rate)))
    command = [args.ffmpeg, "-hide_banner", "-loglevel", "warning", "-y", "-threads", "2"]
    if clip.get("kind") == "title" or clip.get("hold"):
        still = args.title
        if clip.get("kind") != "title":
            still = work / f"hold-{index:02d}.png"
            run(command + ["-i", str(args.video), "-vf", f"trim=start_frame={start_tick}:end_frame={start_tick+1}", "-frames:v", "1", "-update", "1", str(still)])
        if not still.exists():
            raise FileNotFoundError(f"Missing title screenshot: {still}")
        audio_start = clip.get("audio_start", 0 if clip.get("kind") == "title" else 8)
        command += ["-loop", "1", "-framerate", str(rate), "-i", str(still), "-i", str(args.music)]
        audio_first = round(audio_start * 44100)
        audio_last = audio_first + round(duration * 44100)
        video_filter = f"trim=end_frame={frames},setpts=N*{rate.denominator}/({rate.numerator}*TB)"
    else:
        command += ["-i", str(args.video), "-i", str(args.audio)]
        end_tick = clip.get("source_tick_end", start_tick + round(duration * float(source_rate)))
        first = Fraction(start_tick * 44100, 1) / source_rate
        last = Fraction(end_tick * 44100, 1) / source_rate
        audio_first = -(-first.numerator // first.denominator)
        audio_last = -(-last.numerator // last.denominator)
        video_filter = f"trim=start_frame={start_tick}:end_frame={end_tick},setpts=N*{source_rate.denominator}/({source_rate.numerator}*TB)"
    envelope = f"afade=t=in:st=0:d=0.025,afade=t=out:st={duration - 0.025:.6f}:d=0.025"
    audio_filter = f"atrim=start_sample={audio_first}:end_sample={audio_last},asetpts=PTS-STARTPTS,{envelope}"
    command += ["-map", "0:v:0", "-map", "1:a:0", "-vf", f"{video_filter},format=bgra,setsar=1", "-af", audio_filter, "-c:v", "ffv1", "-level", "3", "-threads", "2", "-c:a", "pcm_s16le", "-ar", "44100", "-ac", "2", "-r", str(rate), "-t", f"{duration:.9f}", str(result)]
    run(command)
    return result


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--timeline", type=Path, default=ROOT / "scripts/video/timeline.json")
    parser.add_argument("--video", type=Path, default=ROOT / ".cache/video/gameplay.mkv")
    parser.add_argument("--audio", type=Path, default=ROOT / ".cache/video/gameplay.wav")
    parser.add_argument("--music", type=Path, default=ROOT / ".cache/video/music.wav")
    parser.add_argument("--title", type=Path, default=ROOT / "captures/video-title.png")
    parser.add_argument("--output", type=Path, default=ROOT / "media/ikplus-presentation.webm")
    parser.add_argument("--work", type=Path, default=ROOT / ".cache/video/composition")
    parser.add_argument("--ffmpeg", default=shutil.which("ffmpeg") or "ffmpeg")
    parser.add_argument("--prepare", action="store_true", help="Render graphics and subtitles without encoding.")
    parser.add_argument("--reuse-segments", action="store_true", help="Reuse complete lossless clips after changing presentation graphics.")
    args = parser.parse_args()
    timeline = json.loads(args.timeline.read_text(encoding="utf-8"))
    validate(timeline)
    args.work.mkdir(parents=True, exist_ok=True)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    subtitles = args.output.with_suffix(".srt")
    metadata = args.work / "chapters.ffmetadata"
    graphics = write_graphics(timeline, args.work)
    write_subtitles(timeline, subtitles)
    write_metadata(timeline, metadata)
    if args.prepare:
        print(f"Graphics: {graphics}\nSubtitles: {subtitles}")
        return
    verify_sources(timeline, args)
    clips = []
    for index, clip in enumerate(timeline["clips"]):
        existing = args.work / f"clip-{index:02d}.mkv"
        clips.append(existing if args.reuse_segments and existing.exists() else segment(clip, index, args, args.work, timeline))
    montage = args.work / "montage.ffconcat"
    montage.write_text("ffconcat version 1.0\n" + "\n".join(f"file {quote_concat(path)}" for path in clips) + "\n", encoding="utf-8")
    duration = float(timeline["duration"])
    rate = frame_rate(timeline)
    scaled_height = source_canvas(timeline)[1] * 3
    top_padding = (768 - scaled_height) // 2
    graph = (
        f"[0:v]setpts=N*{rate.denominator}/({rate.numerator}*TB),scale=960:{scaled_height}:flags=neighbor,pad=960:768:0:{top_padding}:black,setsar=1[game];"
        f"[1:v]fps={rate},trim=duration={duration},setpts=PTS-STARTPTS[bg];"
        "[bg][game]overlay=480:120:eof_action=repeat,fade=t=in:st=0:d=0.6,"
        f"fade=t=out:st={duration - 1.6}:d=1.6,format=yuv420p[v]"
    )
    if args.output.suffix.lower() == ".webm":
        # Encode directly from lossless video and PCM so the published WebM
        # does not inherit an intermediate H.264 or AAC generation.
        audio_graph = (
            "[0:a]asetpts=N/SR/TB,volume=0.45,afade=t=in:st=0:d=0.6,"
            f"afade=t=out:st={duration - 1.6}:d=1.6,aresample=48000[a]"
        )
        run([
            args.ffmpeg, "-hide_banner", "-y",
            "-f", "concat", "-safe", "0", "-i", str(montage),
            "-f", "concat", "-safe", "0", "-i", str(graphics),
            "-i", str(metadata), "-filter_complex", graph + ";" + audio_graph,
            "-map", "[v]", "-map", "[a]", "-map_metadata", "2", "-map_chapters", "2",
            "-c:v", "libvpx-vp9", "-b:v", "0", "-crf", "32", "-deadline", "good",
            "-cpu-used", "4", "-row-mt", "1", "-tile-columns", "2", "-frame-parallel", "1",
            "-g", "250", "-pix_fmt", "yuv420p", "-r", str(rate), "-fps_mode", "cfr",
            "-frames:v", str(round(duration * float(rate))),
            "-c:a", "libopus", "-b:a", "128k", "-ar", "48000", "-application", "audio",
            "-t", str(duration), str(args.output),
        ])
        print(f"Finished: {args.output}\nEnglish captions: {subtitles}")
        return
    if args.output.suffix.lower() != ".mp4":
        raise ValueError("The presentation output must use .webm or .mp4.")

    encoded = args.work / "video.mp4"
    # Attach chapters after image compositing so the final chapter track is
    # written consistently. The second pass copies the encoded video exactly.
    run([
        args.ffmpeg, "-hide_banner", "-y",
        "-f", "concat", "-safe", "0", "-i", str(montage),
        "-f", "concat", "-safe", "0", "-i", str(graphics),
        "-filter_complex", graph, "-map", "[v]",
        "-map_metadata", "-1", "-map_chapters", "-1",
        "-c:v", "libx264", "-preset", "medium", "-crf", "18",
        "-profile:v", "high", "-level", "4.2", "-r", str(rate), "-threads", "2",
        "-frames:v", str(round(duration * float(rate))),
        "-t", str(duration), str(encoded),
    ])
    # AAC can overshoot the PCM sample peak. Preserve sufficient headroom and
    # encode directly from lossless PCM, with no intermediate lossy audio pass.
    envelope = f"asetpts=N/SR/TB,volume=0.45,afade=t=in:st=0:d=0.6,afade=t=out:st={duration - 1.6}:d=1.6"
    run([
        args.ffmpeg, "-hide_banner", "-y", "-i", str(encoded),
        "-f", "concat", "-safe", "0", "-i", str(montage),
        "-i", str(metadata), "-map", "0:v:0", "-map", "1:a:0",
        "-map_metadata", "2", "-map_chapters", "2",
        "-c:v", "copy", "-c:a", "aac", "-b:a", "192k",
        "-af", envelope, "-movflags", "+faststart+disable_chpl",
        "-video_track_timescale", str(rate.numerator), "-movie_timescale", "1000",
        "-t", str(duration), str(args.output),
    ])
    print(f"Finished: {args.output}\nEnglish captions: {subtitles}")


if __name__ == "__main__":
    main()
