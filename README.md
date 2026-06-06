# ClipCompress

**Tray app for Windows that watches your NVIDIA Instant Replay clips and re-encodes them to AV1 (NVENC) so they're small enough to drop into Discord.**

## Contents

- [How it works](#how-it-works)
- [Installation](#installation)
- [Configuration](#configuration)
- [Encoding](#encoding)
- [Building from source](#building-from-source)

## How it works

ClipCompress sits in the system tray and watches your clips folder (and every game subfolder under it). When NVIDIA's Instant Replay / ShadowPlay drops a new clip, ClipCompress waits for the file to finish writing, then re-encodes it with your GPU's NVENC encoder (AV1, HEVC, or H.264 depending on the card) and writes a much smaller file into the output folder. By default it targets a file size that fits Discord's free upload limit, so the result is ready to share.

ffmpeg is not shipped with the app — on first run it downloads a pinned build into your app-data folder.

## Installation

Download `ClipCompressSetup.exe` from the [latest release](../../releases/latest) and run it. The installer is per-user (no admin prompt) and adds a Start Menu shortcut. On first launch ClipCompress registers itself to start at login, downloads ffmpeg, and starts watching.

> ⚠️ Encoding uses NVIDIA NVENC, so an NVIDIA GPU is required. AV1 needs an RTX 40-series (or newer); older cards fall back to HEVC or H.264 automatically. Without an NVIDIA GPU the tray shows an error and clips won't be encoded.

## Configuration

Everything is configured from the tray: right-click the icon and choose **Settings…**. Pick the source and output folders, the encoding mode, and toggles for deleting originals, finish notifications, and start-at-login. Settings are saved automatically and applied live. Use **Pause** in the tray menu to stop processing temporarily.

## Encoding

All encoding is hardware-accelerated through NVIDIA NVENC. The codec is chosen by **Auto** (default) — it probes your GPU and picks the best encoder that works — or you can pin one in Settings:

| Codec | GPU | Container | Audio |
| ----- | --- | --------- | ----- |
| AV1   | RTX 40/50-series | `.webm` | Opus (stereo) |
| HEVC  | RTX 20/30-series + most GTX | `.mp4` | AAC (stereo) |
| H.264 | older / anything | `.mp4` | AAC (stereo) |

AV1 uses the slowest/highest-quality NVENC preset (`p7`, `-tune hq`). Three rate modes are available:

- **size** — fit each clip under a target size in MB (default 9 MB, safe for Discord's free tier). Raise it if you have Nitro.
- **quality** — constant quality (`cq`, 0–51); file size varies with content.
- **bitrate** — a fixed video bitrate in kbps.

Clips are encoded one at a time to avoid GPU contention.

## Building from source

Requires Go (see `go.mod`). On Windows, Fyne needs a C compiler (mingw) and `CGO_ENABLED=1`:

```bash
go build -ldflags "-H windowsgui -s -w" -o clip-compress.exe .
```

For UI iteration on macOS or Linux the app builds and runs without the Windows-only integrations:

```bash
go run .
```

Releases (the `.exe` plus the Inno Setup installer) are built on a `windows-latest` GitHub runner — see `.github/workflows/release.yml`.
