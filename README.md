# vgmtui

A terminal-based VGM (Video Game Music) player built with Go, inspired by [termusic](https://github.com/tramhao/termusic).

## Overview

vgmtui is a TUI wrapper around [libvgm](https://github.com/ValleyBell/libvgm) for playing video game music files. It uses:

- **Go** with [Bubbletea](https://github.com/charmbracelet/bubbletea) for the TUI
- **libvgm** for audio playback (via CGO)

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
- Audio libraries: ALSA and PulseAudio

On Debian/Ubuntu:
```bash
sudo apt install build-essential cmake zlib1g-dev libasound2-dev libpulse-dev libao-dev
```

### Clone and Build

```bash
# Clone with submodules
git clone --recurse-submodules https://github.com/user/vgmtui.git
cd vgmtui

# Or if already cloned, initialize submodules
git submodule update --init --recursive

# Build
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

If no path is given and `~/VGM` exists, it starts in library mode with a hierarchical view (System > Game > Track). Otherwise, it falls back to file browser mode starting from the home directory.

### Key Bindings

| Key | Action |
|-----|--------|
| `Space` | Play/Pause |
| `n` / `N` | Next/Previous track |
| `Enter` | Add file to playlist / Play selected |
| `d` / `D` | Remove track / Clear playlist |
| `Tab` | Switch focus between panels |
| `j/k` | Navigate up/down |
| `a` | Add all tracks from current game/system |
| `L` | Add all files from current directory |
| `?` | Help |
| `q` | Quit |

### Library Mode

When `~/VGM` exists, vgmtui operates in library mode with a hierarchical browser:
- **System** (e.g., Genesis, SNES, PC Engine)
- **Game** (organized by GD3 metadata)
- **Track** (individual VGM files)

The library is indexed on startup by scanning GD3 tags from VGM files.

### File Browser Mode

When `~/VGM` doesn't exist, vgmtui falls back to a traditional file browser starting from the home directory. Navigate to find your VGM files.

## License

MIT
