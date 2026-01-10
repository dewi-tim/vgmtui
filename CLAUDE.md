# CLAUDE.md

Instructions for Claude Code when working on vgmtui.

## Project Overview

vgmtui is a terminal-based VGM (Video Game Music) player built with:
- **Go** with Bubbletea TUI framework
- **libvgm** for audio playback (via CGO, included as git submodule)

## Build Commands

```bash
# Initialize submodule (first time only)
git submodule update --init --recursive

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
    components/        # UI components (browser, playlist, help, progress)
  player/              # Audio engine
    player.go          # AudioPlayer high-level interface
    libvgm.go          # CGO bindings to libvgm
    track.go           # Track metadata types
  library/             # Music library indexing
    library.go         # Library scanner and index
libvgm/
  libvgm/              # Git submodule - ValleyBell/libvgm
  wrapper.h            # C API header
  wrapper.cpp          # C++ to C wrapper
  build/               # Build artifacts
```

## CGO Notes

- The C wrapper (`libvgm/wrapper.cpp`) wraps libvgm's C++ PlayerA class
- CGO flags are in `internal/player/libvgm.go`
- libvgm is built as static libraries in `libvgm/build/`
- Link order matters: `-lvgm_wrapper -lvgm-audio -lvgm-player -lvgm-emu -lvgm-utils -lz -lstdc++`

## Key Patterns

### Bubbletea Model
Follow the Elm architecture: Model, Update, View. All state changes flow through typed messages.

### Message Passing
Audio events come from the player via channels. UI messages are typed structs.

### Focus Management
Track focused panel with enum (FocusBrowser, FocusPlaylist). Tab cycles focus.

### Library Mode vs File Browser
- If `~/VGM` exists: Library mode with hierarchical System > Game > Track browser
- Otherwise: File browser fallback starting from home directory

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
