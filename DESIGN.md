# VGMTUI - Design Document

A terminal-based VGM music player inspired by termusic, built with Go using the Bubbletea framework and the libvgm playback library via CGO.

## Overview

**vgmtui** provides a user-friendly terminal interface for playing Video Game Music (VGM) files. It combines the proven UI patterns of termusic with the powerful VGM playback capabilities of libvgm.

### Goals

- Provide a termusic-like experience for VGM files
- Support all formats handled by libvgm (VGM, VGZ, S98, DRO, GYM)
- Expose VGM-specific features (loop handling, fade, chip info)
- Clean, responsive TUI using Bubbletea/Bubbles
- Cross-platform (primary target: Linux arm64)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Go Application                         │
├─────────────────────────────────────────────────────────────┤
│  Bubbletea TUI                                              │
│  ┌─────────────┬────────────────────────────────────────┐  │
│  │   Browser   │              Playlist                  │  │
│  │  (filepicker│  ┌─────────────────────────────────┐   │  │
│  │   or list)  │  │  table.Model (queue tracks)     │   │  │
│  │             │  └─────────────────────────────────┘   │  │
│  │             ├────────────────────────────────────────┤  │
│  │             │           Track Info (viewport)        │  │
│  │             ├────────────────────────────────────────┤  │
│  │             │  [▶ Playing] ████████░░░░ 01:23/03:00  │  │
│  └─────────────┴────────────────────────────────────────┘  │
│  Footer: Space: play/pause | n: next | ?: help | q: quit   │
├─────────────────────────────────────────────────────────────┤
│  Audio Engine (goroutine)                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  libvgm CGO bindings → render samples → audio output │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Technology Stack

| Layer | Technology |
|-------|------------|
| TUI Framework | Bubbletea + Bubbles + Lipgloss |
| Audio Backend | libvgm (via CGO) + oto (Go audio) |
| Build | Go 1.21+, CMake (libvgm), Make |
| Platform | Linux arm64 (primary), portable |

## Reference Repositories

The following reference repositories inform this design:

| Repository | Purpose |
|------------|---------|
| `reference-repos/music/termusic/` | UI patterns, layout, key bindings |
| `reference-repos/vgm/libvgm/` | Core playback library |
| `reference-repos/vgm/vgmplay-libvgm/` | CLI features, VGM-specific functionality |
| `reference-repos/tui/bubbletea/` | TUI framework |
| `reference-repos/tui/bubbles/` | UI components |

## File Structure

```
vgmtui/
├── cmd/vgmtui/
│   └── main.go              # Entry point
├── internal/
│   ├── ui/
│   │   ├── model.go         # Main Bubbletea model
│   │   ├── keymap.go        # Key bindings
│   │   ├── view.go          # Layout rendering
│   │   ├── update.go        # Message handling
│   │   ├── styles.go        # Lipgloss styles
│   │   └── components/
│   │       ├── browser.go   # File browser wrapper
│   │       ├── playlist.go  # Playlist table wrapper
│   │       ├── trackinfo.go # Track metadata display
│   │       ├── progress.go  # Progress bar
│   │       └── help.go      # Help popup
│   ├── player/
│   │   ├── player.go        # Go player interface
│   │   ├── libvgm.go        # CGO bindings
│   │   └── track.go         # Track metadata struct
│   └── config/
│       └── config.go        # Configuration management
├── libvgm/
│   ├── wrapper.c            # C wrapper for PlayerA
│   ├── wrapper.h            # C API header
│   └── CMakeLists.txt       # Build libvgm + wrapper
├── go.mod
├── go.sum
├── Makefile
├── DESIGN.md
├── CLAUDE.md
└── README.md
```

## UI Layout

```
┌──────────────────────────────────────────────────────────────────┐
│ ┌──────────────────┐ ┌─────────────────────────────────────────┐ │
│ │  Library         │ │  Playlist                    [2/15]     │ │
│ │ ──────────────── │ │ ─────────────────────────────────────── │ │
│ │  📁 ..           │ │  Duration │ Title       │ Game          │ │
│ │  📁 Genesis      │ │  ─────────┼─────────────┼────────────── │ │
│ │  📁 SNES         │ │    02:34  │ Green Hill  │ Sonic 1       │ │
│ │ >📁 PC-Engine    │ │  ▶ 03:01  │ Marble Zone │ Sonic 1       │ │
│ │    📄 song.vgm   │ │    01:45  │ Star Light  │ Sonic 1       │ │
│ │    📄 track.vgz  │ │                                         │ │
│ │                  │ ├─────────────────────────────────────────┤ │
│ │                  │ │ Track: Marble Zone                      │ │
│ │                  │ │ Game:  Sonic the Hedgehog               │ │
│ │                  │ │ System: Sega Genesis   Chips: YM2612    │ │
│ │                  │ │ Composer: Masato Nakamura               │ │
│ └──────────────────┘ ├─────────────────────────────────────────┤ │
│                      │ ▶ Playing │ ████████░░░░ 01:23 / 03:01  │ │
│                      │ Vol: 100% │ Loop 1/2   │ Speed: 1.0x    │ │
│                      └─────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────────┤
│ Space:play │ n/N:next/prev │ f/b:seek │ Tab:focus │ ?:help │ q  │
└──────────────────────────────────────────────────────────────────┘
```

## Core Data Structures

### Track Metadata

```go
type Track struct {
    Path      string
    Title     string        // GD3: TITLE
    Game      string        // GD3: GAME (album)
    System    string        // GD3: SYSTEM
    Composer  string        // GD3: ARTIST
    Date      string        // GD3: DATE
    Creator   string        // GD3: ENCODED_BY
    Notes     string        // GD3: COMMENT
    Format    string        // "VGM 1.71", "S98", etc.
    Duration  time.Duration // Total including loops
    LoopPoint time.Duration // Loop start position
    HasLoop   bool
    Chips     []ChipInfo
}

type ChipInfo struct {
    Name string // "YM2612"
    Core string // "GPGX"
}
```

### Playback State

```go
type PlayState int

const (
    StateStopped PlayState = iota
    StatePlaying
    StatePaused
    StateFading
)

type PlaybackInfo struct {
    State       PlayState
    Position    time.Duration
    Duration    time.Duration
    CurrentLoop int
    TotalLoops  int
    Volume      float64  // 0.0-1.0
    Speed       float64  // 0.5-2.0
}
```

### UI Model

```go
type Focus int

const (
    FocusBrowser Focus = iota
    FocusPlaylist
)

type Model struct {
    // Window
    width, height int

    // Focus
    focus Focus

    // UI Components (from bubbles)
    browser   filepicker.Model  // File navigation
    playlist  table.Model       // Queue
    trackInfo viewport.Model    // Metadata display
    progress  progress.Model    // Progress bar
    help      help.Model        // Help rendering

    // State
    keyMap     KeyMap
    showHelp   bool

    // Playback
    player       *player.Player
    currentTrack *Track
    queue        []Track
    queueIndex   int
    playback     PlaybackInfo

    // Config
    config *config.Config
}
```

## Key Bindings

| Key | Action | Context |
|-----|--------|---------|
| `Space` | Play/Pause | Global |
| `n` / `N` | Next/Previous track | Global |
| `f` / `b` | Seek +5s / -5s | Global |
| `F` | Fade out | Global |
| `+` / `-` | Volume up/down | Global |
| `[` / `]` | Speed down/up | Global |
| `Tab` / `Shift+Tab` | Cycle focus | Global |
| `?` | Toggle help | Global |
| `q` | Quit | Global |
| `j` / `k` / `↑` / `↓` | Navigate | Focused panel |
| `Enter` | Add to queue / Play | Browser/Playlist |
| `a` | Add to queue | Browser |
| `d` | Remove from queue | Playlist |
| `D` | Clear queue | Playlist |
| `g` / `G` | Top/Bottom | Focused panel |
| `r` | Restart track | Global |

## Message Types

```go
// User actions
type (
    PlayPauseMsg   struct{}
    NextTrackMsg   struct{}
    PrevTrackMsg   struct{}
    SeekMsg        struct{ Delta time.Duration }
    FadeOutMsg     struct{}
    VolumeMsg      struct{ Delta float64 }
    SpeedMsg       struct{ Delta float64 }
    FocusMsg       Focus
    ToggleHelpMsg  struct{}
    QuitMsg        struct{}
)

// Queue management
type (
    AddToQueueMsg    struct{ Tracks []Track }
    RemoveFromQueueMsg struct{ Index int }
    ClearQueueMsg    struct{}
    PlaySelectedMsg  struct{ Index int }
)

// Playback events (from audio goroutine)
type (
    TickMsg        struct{ Info PlaybackInfo }
    TrackEndedMsg  struct{}
    TrackLoadedMsg struct{ Track Track }
    ErrorMsg       struct{ Err error }
)

// Window
type WindowSizeMsg tea.WindowSizeMsg
```

## CGO Integration (libvgm wrapper)

### C Wrapper Header (`libvgm/wrapper.h`)

```c
#ifndef VGM_WRAPPER_H
#define VGM_WRAPPER_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct VgmPlayer VgmPlayer;

// Lifecycle
VgmPlayer* vgm_player_create(void);
void vgm_player_destroy(VgmPlayer* p);

// Configuration
void vgm_player_set_sample_rate(VgmPlayer* p, uint32_t rate);
void vgm_player_set_loop_count(VgmPlayer* p, uint32_t count);
void vgm_player_set_fade_time(VgmPlayer* p, uint32_t ms);
void vgm_player_set_volume(VgmPlayer* p, double vol);
void vgm_player_set_speed(VgmPlayer* p, double speed);

// File operations
int vgm_player_load(VgmPlayer* p, const char* path);
void vgm_player_unload(VgmPlayer* p);

// Playback control
int vgm_player_start(VgmPlayer* p);
void vgm_player_stop(VgmPlayer* p);
void vgm_player_reset(VgmPlayer* p);
void vgm_player_fade_out(VgmPlayer* p);

// Seeking
void vgm_player_seek(VgmPlayer* p, double seconds);

// Rendering
uint32_t vgm_player_render(VgmPlayer* p, uint32_t frames, int16_t* buffer);
int vgm_player_is_playing(VgmPlayer* p);
int vgm_player_is_fading(VgmPlayer* p);

// Position
double vgm_player_get_position(VgmPlayer* p);
double vgm_player_get_duration(VgmPlayer* p);
uint32_t vgm_player_get_current_loop(VgmPlayer* p);
int vgm_player_has_loop(VgmPlayer* p);
double vgm_player_get_loop_point(VgmPlayer* p);

// Metadata
const char* vgm_player_get_title(VgmPlayer* p);
const char* vgm_player_get_game(VgmPlayer* p);
const char* vgm_player_get_system(VgmPlayer* p);
const char* vgm_player_get_composer(VgmPlayer* p);
const char* vgm_player_get_date(VgmPlayer* p);
const char* vgm_player_get_format(VgmPlayer* p);
uint32_t vgm_player_get_chip_count(VgmPlayer* p);
const char* vgm_player_get_chip_name(VgmPlayer* p, uint32_t index);

#ifdef __cplusplus
}
#endif

#endif // VGM_WRAPPER_H
```

### Go Bindings Overview

The Go bindings in `internal/player/libvgm.go` will use CGO to call the C wrapper functions. Key considerations:

- Use `C.CString` for string parameters (remember to `C.free`)
- Return values from C are copied to Go types
- Audio rendering happens in a separate goroutine
- Thread safety via mutexes when accessing player state

## Audio Output

Audio output uses the `oto` library (github.com/ebitengine/oto/v3) which provides cross-platform audio playback. The audio engine runs in a separate goroutine:

1. libvgm renders samples to a buffer via `vgm_player_render()`
2. Samples are written to oto's player
3. Position updates are sent to the TUI via channels
4. Control messages (play/pause/seek) are received via channels

## Supported File Formats

| Extension | Format | Description |
|-----------|--------|-------------|
| `.vgm` | VGM | Video Game Music |
| `.vgz` | VGZ | Compressed VGM (gzip) |
| `.s98` | S98 | PC-98 sound logs |
| `.dro` | DRO | DOSBox OPL recordings |
| `.gym` | GYM | Genesis YM2612 logs |

## Configuration File

Location: `~/.config/vgmtui/config.toml`

```toml
[playback]
sample_rate = 44100
loop_count = 2
fade_time = 4000   # ms
jingle_pause = 500 # ms after non-looping

[display]
prefer_japanese = false
time_style = "elapsed"  # elapsed | remaining | both

[keys]
play_pause = "space"
next = "n"
prev = "N"
seek_forward = "f"
seek_backward = "b"

[paths]
music_roots = ["~/vgm", "~/Music/VGM"]
```

## Build System

### Prerequisites

- Go 1.21+
- CMake 3.10+
- C/C++ compiler (gcc/clang)
- zlib development headers
- ALSA development headers (Linux)

### Makefile

```makefile
.PHONY: all libvgm build clean test

LIBVGM_SRC = ../reference-repos/vgm/libvgm

all: libvgm build

libvgm:
	mkdir -p libvgm/build
	cd libvgm/build && cmake $(LIBVGM_SRC) \
		-DCMAKE_BUILD_TYPE=Release \
		-DBUILD_LIBAUDIO=OFF \
		-DBUILD_TESTS=OFF \
		-DLIBRARY_TYPE=STATIC
	cd libvgm/build && cmake --build . --parallel
	$(CXX) -c libvgm/wrapper.cpp -o libvgm/build/wrapper.o \
		-I libvgm/build -I $(LIBVGM_SRC)
	ar rcs libvgm/build/libvgm_wrapper.a libvgm/build/wrapper.o

build: libvgm
	CGO_ENABLED=1 go build -o vgmtui ./cmd/vgmtui

test:
	go test ./...

clean:
	rm -rf libvgm/build vgmtui

install: build
	install -Dm755 vgmtui $(DESTDIR)/usr/local/bin/vgmtui
```

## Implementation Phases

### Phase 1: Core Foundation
- [ ] Set up Go module structure
- [ ] Implement libvgm C wrapper
- [ ] Create CGO bindings
- [ ] Basic audio playback with oto
- [ ] Test playback of single VGM file

### Phase 2: Basic TUI
- [ ] Bubbletea Model/Update/View skeleton
- [ ] Progress bar component
- [ ] Basic key handling (play/pause/quit)
- [ ] Single-track playback with UI

### Phase 3: File Browser
- [ ] Filepicker component integration
- [ ] VGM/VGZ file filtering
- [ ] Directory navigation
- [ ] Track metadata loading

### Phase 4: Playlist
- [ ] Table component for queue
- [ ] Add/remove tracks
- [ ] Queue navigation
- [ ] Auto-advance to next track

### Phase 5: Full Features
- [ ] Help popup overlay
- [ ] Volume/speed controls
- [ ] Fade out support
- [ ] Loop count configuration
- [ ] Seek functionality

### Phase 6: Polish
- [ ] Configuration file
- [ ] Error handling/display
- [ ] M3U playlist support
- [ ] Japanese tag toggle

## VGM-Specific UI Considerations

### Progress Bar
- Show percentage through file data
- Current time / total time (with loop handling)
- Loop status indicator (e.g., "Loop 1/2")
- Fading indicator when fade-out is active

### Track Info Panel
Display GD3 metadata:
- Track title
- Game name
- System/Platform
- Composer
- Release date
- Sound chips used (with emulation core)

### Loop Handling
VGM files often contain loop points. The player should:
- Display whether track loops
- Show current loop number
- Allow configurable loop count
- Support manual fade-out trigger

### Chip Display
Show sound chips used in the current track:
```
Chips: YM2612 (Nuked), SN76496 (MAME)
```

## Error Handling

- File load errors: Display in status bar, don't crash
- Audio device errors: Graceful fallback or clear error message
- Invalid files: Skip and continue to next in queue
- CGO panics: Recover and display error

## Future Enhancements

- Album art display (if terminal supports)
- Waveform visualization
- Per-chip mute controls
- Equalizer
- Network streaming
- Remote control via socket
- Lyrics/notes display from VGM comments
