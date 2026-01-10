# CLAUDE.md

Instructions for Claude Code when working on vgmtui.

## Project Overview

vgmtui is a terminal-based VGM (Video Game Music) player built with:
- **Go** with Bubbletea TUI framework
- **libvgm** for audio playback (via CGO)
- **oto** for audio output

## Reference Repositories

These repositories in `../reference-repos/` inform the design:

| Path | Purpose |
|------|---------|
| `music/termusic/` | UI patterns, especially `tui/` subdirectory |
| `vgm/libvgm/` | Core playback library (C/C++) |
| `vgm/vgmplay-libvgm/` | CLI features, VGM-specific functionality |
| `tui/bubbletea/` | TUI framework patterns |
| `tui/bubbles/` | UI components (list, table, progress, etc.) |

## Build Commands

```bash
# Build libvgm wrapper and Go binary
make

# Build only libvgm
make libvgm

# Build only Go binary (after libvgm is built)
make build

# Run tests
make test

# Clean build artifacts
make clean
```

## Architecture

```
cmd/vgmtui/main.go     # Entry point
internal/
  ui/                  # Bubbletea TUI layer
    model.go           # Main model
    update.go          # Message handling
    view.go            # Rendering
    keymap.go          # Key bindings
    styles.go          # Lipgloss styles
    components/        # UI components
  player/              # Audio engine
    player.go          # Player interface
    libvgm.go          # CGO bindings
    track.go           # Track metadata
  config/              # Configuration
libvgm/
  wrapper.h            # C API header
  wrapper.cpp          # C++ to C wrapper
  CMakeLists.txt       # libvgm build
```

## CGO Notes

- The C wrapper (`libvgm/wrapper.cpp`) wraps libvgm's C++ PlayerA class
- CGO flags are in `internal/player/libvgm.go`
- libvgm is built as static libraries in `libvgm/build/`
- Link order matters: `-lvgm_wrapper -lvgm-player -lvgm-emu -lvgm-utils -lz -lstdc++`

## Key Patterns

### Bubbletea Model
Follow the Elm architecture: Model, Update, View. See `reference-repos/tui/bubbletea/tea.go`.

### Message Passing
Use typed messages for all state changes. Audio events come via channels from the player goroutine.

### Focus Management
Track focused panel with enum, use Tab to cycle, route navigation keys to focused component only.

### Audio Thread
Player runs in separate goroutine, communicates via channels for thread safety.

## File Formats

Supported: `.vgm`, `.vgz`, `.s98`, `.dro`, `.gym`

## Testing

```bash
# Run all tests
go test ./...

# Test specific package
go test ./internal/player/

# Test with verbose output
go test -v ./...
```
