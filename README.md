# vgmtui

A terminal-based VGM (Video Game Music) player built with Go, inspired by [termusic](https://github.com/tramhao/termusic).

## Overview

vgmtui is a TUI wrapper around [libvgm](https://github.com/ValleyBell/libvgm) for playing video game music files. It uses:

- **Go** with [Bubbletea](https://github.com/charmbracelet/bubbletea) for the TUI
- **libvgm** for audio playback (via CGO)
- Inspired by **vgmplay** for VGM-specific functionality

## Supported Formats

- `.vgm` / `.vgz` - Video Game Music
- `.s98` - PC-98 sound logs
- `.dro` - DOSBox OPL recordings
- `.gym` - Genesis YM2612 logs

## Building

### Prerequisites

- Go 1.21+
- CMake 3.10+
- C/C++ compiler (gcc/clang)
- zlib development headers
- Audio libraries: ALSA, PulseAudio, or libao

On Debian/Ubuntu:
```bash
sudo apt install build-essential cmake zlib1g-dev libasound2-dev libpulse-dev libao-dev
```

### Build

```bash
make
```

This builds libvgm and the Go binary. The resulting `vgmtui` binary will be in the project root.

### Install

```bash
make install
```

Installs to `/usr/local/bin/vgmtui`.

## Usage

```bash
vgmtui [path]
```

If no path is given, starts in `~/VGM` or home directory.

### Key Bindings

| Key | Action |
|-----|--------|
| `Space` | Play/Pause |
| `n` / `N` | Next/Previous track |
| `Enter` | Add file to playlist / Play selected |
| `d` / `D` | Remove track / Clear playlist |
| `Tab` | Switch focus between panels |
| `j/k` | Navigate up/down |
| `?` | Help |
| `q` | Quit |

## License

MIT
