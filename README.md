# ClipCompress

**Tray app for Windows that watches your NVIDIA Instant Replay clips and re-encodes them to AV1 (NVENC) so they're small enough to drop into Discord.**

## Contents

- [How it works](#how-it-works)
- [Installation](#installation)
- [Configuration](#configuration)
- [Encoding](#encoding)
- [Building from source](#building-from-source)

## How it works

ClipCompress sits in the system tray and watches your clips folder (and every game subfolder under it). When NVIDIA's Instant Replay / ShadowPlay drops a new clip, ClipCompress waits for the file to finish writing, then re-encodes it with your GPU's NVENC encoder (AV1, HEVC, or H.264 depending on the card) and writes a much smaller file into the output folder. Encoding is constant-bitrate (1900 kbps by default), which keeps clips small enough to drop into Discord.

ffmpeg ships with the app. It is a purpose-built LGPL build cross-compiled from source in CI, carrying only the NVENC encoders, the AAC and Opus encoders, and the decoders needed to read your clips, which keeps it around a tenth the size of a general-purpose build. The build script and its exact configure flags live in `build/ffmpeg/build.sh`, and the licence and source links are installed next to the binary.

## Installation

Download `ClipCompressSetup.exe` from the [latest release](../../releases/latest) and run it. The installer is per-user (no admin prompt) and adds a Start Menu shortcut. On first launch ClipCompress registers itself to start at login and starts watching.

> ⚠️ Encoding uses NVIDIA NVENC, so an NVIDIA GPU is required. AV1 needs an RTX 40-series (or newer); older cards fall back to HEVC or H.264 automatically. Without an NVIDIA GPU the tray shows an error and clips won't be encoded.

## Configuration

Everything is configured from the tray: right-click the icon and choose **Settings…**. Pick the source and output folders, the target video bitrate (kbps), and toggles for deleting originals, finish notifications, and start-at-login. Settings are saved on **Save** and applied live. Use **Pause** in the tray menu to stop processing temporarily.

## Encoding

All encoding is hardware-accelerated through NVIDIA NVENC. The codec is chosen automatically — it probes your GPU and picks the best encoder that works, trying AV1 → HEVC → H.264 in order:

| Codec | GPU                         | Container | Audio         |
| ----- | --------------------------- | --------- | ------------- |
| AV1   | RTX 40/50-series            | `.webm`   | Opus (stereo) |
| HEVC  | RTX 20/30-series + most GTX | `.mp4`    | AAC (stereo)  |
| H.264 | older / anything            | `.mp4`    | AAC (stereo)  |

Every profile uses the slowest/highest-quality NVENC preset (`p7`, `-tune hq`) and encodes at a constant bitrate (`-rc cbr`, two-pass full-res). The only encoding knob is the target video bitrate in kbps, set in Settings (default 1900, floored at 200). Audio is re-encoded to stereo at 128 kbps.

Clips are encoded one at a time to avoid GPU contention.

## Building from source

This is a Windows-only application and only builds for Windows.

Requires Go (see `go.mod`) and mingw, since the interface is built on [IUP](https://www.tecgraf.puc-rio.br/iup) through [iup-go](https://github.com/gen2brain/iup-go), which draws real Win32 controls and compiles its own C sources as part of the build.
`CGO_ENABLED=1` is therefore required.
The first build takes a few minutes while those sources compile; everything after that comes from the build cache.

Generate the Windows resources before building.
They carry the icon, the version info and the manifest that asks for Common Controls v6, and without them the controls fall back to the Windows Classic look:

```bash
go tool goversioninfo -64 -arm=false -o resource_windows_amd64.syso build/versioninfo.json
go build -tags nomanifest -ldflags "-H windowsgui -s -w" -o clip-compress.exe .
```

The `nomanifest` tag matters: iup-go embeds a manifest of its own by default, and the linker refuses to merge two.

The app looks for `ffmpeg.exe` beside its own executable and does nothing without it.
That binary is cross-compiled from source with `build/ffmpeg/build.sh`, which needs a Linux host with mingw-w64, meson, ninja, cmake and nasm; run it in a container if you are not on Linux:

```bash
docker run --rm -v "$PWD/build/ffmpeg:/build" ubuntu:24.04 bash -c \
  'apt-get update && apt-get install -y build-essential mingw-w64 nasm yasm meson ninja-build cmake pkg-config curl xz-utils libz-mingw-w64-dev && bash /build/build.sh'
cp build/ffmpeg/out/ffmpeg.exe .
```

Releases (the `.exe` plus the Inno Setup installer) are built on a `windows-latest` GitHub runner — see `.github/workflows/release.yml`.
