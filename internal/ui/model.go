package ui

import (
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dewi-tim/vgmtui/internal/library"
	"github.com/dewi-tim/vgmtui/internal/player"
	"github.com/dewi-tim/vgmtui/internal/ui/components"
)

// Focus represents which panel is currently focused.
type Focus int

const (
	FocusBrowser Focus = iota
	FocusPlaylist
)

// PlayState represents the current playback state.
type PlayState int

const (
	StateStopped PlayState = iota
	StatePlaying
	StatePaused
)

// Track is an alias for the components.Track type.
// This allows other packages to use ui.Track without importing components.
type Track = components.Track

// PlaybackInfo holds current playback state.
type PlaybackInfo struct {
	State       PlayState
	Position    time.Duration
	Duration    time.Duration
	CurrentLoop int
	TotalLoops  int
}

// Model is the main Bubbletea model for vgmtui.
type Model struct {
	// Window dimensions
	width  int
	height int

	// Focus management
	focus Focus

	// UI Components
	browser    components.Browser     // File browser (fallback mode)
	libBrowser components.LibBrowser  // Library browser (main mode)
	lib        *library.Library       // Music library
	useLibrary bool                   // Whether to use library browser
	playlist   components.Playlist
	progress   components.ProgressBar
	helpPopup  components.HelpPopup

	// Key bindings
	keyMap KeyMap

	// UI state
	showHelp bool
	quitting bool

	// Error display
	lastError string
	errorTime time.Time

	// Playback state
	playback     PlaybackInfo
	currentTrack *Track
	volume       float64 // Volume level (0.0 - 1.0+)
	trackLoading bool    // True while a playTrack command is in flight

	// Audio player (nil in TUI-only mode)
	audioPlayer *player.AudioPlayer
	playerSub   <-chan player.PlaybackInfo

	// Track chip info (from real player)
	trackChips []player.ChipInfo

	// Styles
	styles Styles
}

// New creates a new Model with default values (TUI-only mode).
func New() Model {
	return NewWithPlayer(nil)
}

// NewWithPlayer creates a new Model with an optional audio player.
// If player is nil, the TUI runs in display-only mode.
func NewWithPlayer(ap *player.AudioPlayer) Model {
	// Determine library root - prefer ~/VGM if it exists
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	vgmDir := filepath.Join(home, "VGM")
	useLibrary := false

	// Check if ~/VGM exists
	if home != "" {
		if info, err := os.Stat(vgmDir); err == nil && info.IsDir() {
			useLibrary = true
		}
	}

	// Initialize library and library browser if ~/VGM exists
	var lib *library.Library
	var libBrowser components.LibBrowser
	if useLibrary {
		lib = library.New(vgmDir)
		libBrowser = components.NewLibBrowser(lib)
		libBrowser.Focus() // Start with library focused
	}

	// Initialize browser with home directory (fallback - always created for switching)
	browser := components.NewBrowser("")
	if !useLibrary {
		browser.Focus() // Only focus if not using library
	}

	// Initialize empty playlist
	playlist := components.NewPlaylist()

	m := Model{
		focus:       FocusBrowser,
		browser:     browser,
		libBrowser:  libBrowser,
		lib:         lib,
		useLibrary:  useLibrary,
		playlist:    playlist,
		progress:    components.NewProgressBar(),
		helpPopup:   components.NewHelpPopup(),
		keyMap:      DefaultKeyMap(),
		styles:      DefaultStyles(),
		audioPlayer: ap,
		volume:      1.0,
		playback: PlaybackInfo{
			State:      StateStopped,
			TotalLoops: 2,
		},
	}

	// Subscribe to player updates if player is available
	if ap != nil {
		m.playerSub = ap.Subscribe()
	}

	return m
}

// Init returns the initial command to run.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Initialize browser based on mode
	if m.useLibrary {
		cmds = append(cmds, m.libBrowser.Init())
	} else {
		cmds = append(cmds, m.browser.Init())
	}

	// If we have a real player, start listening for playback updates
	if m.playerSub != nil {
		cmds = append(cmds, listenForPlayback(m.playerSub))
	}

	return tea.Batch(cmds...)
}

// listenForPlayback returns a command that listens for playback info updates.
func listenForPlayback(sub <-chan player.PlaybackInfo) tea.Cmd {
	return func() tea.Msg {
		info, ok := <-sub
		if !ok {
			// Channel closed
			return PlaybackChannelClosedMsg{}
		}
		return PlayerTickMsg{Info: info}
	}
}

// PlayerTickMsg is sent when the audio player provides a playback update.
type PlayerTickMsg struct {
	Info player.PlaybackInfo
}

// PlaybackChannelClosedMsg is sent when the playback subscription channel closes.
type PlaybackChannelClosedMsg struct{}

// TrackEndedMsg is sent when the current track finishes playing.
type TrackEndedMsg struct{}

// Width returns the current window width.
func (m Model) Width() int {
	return m.width
}

// Height returns the current window height.
func (m Model) Height() int {
	return m.height
}

// Focus returns the currently focused panel.
func (m Model) Focus() Focus {
	return m.focus
}

// IsPlaying returns true if playback is active.
func (m Model) IsPlaying() bool {
	return m.playback.State == StatePlaying
}

// IsPaused returns true if playback is paused.
func (m Model) IsPaused() bool {
	return m.playback.State == StatePaused
}

// IsStopped returns true if playback is stopped.
func (m Model) IsStopped() bool {
	return m.playback.State == StateStopped
}

// HasPlayer returns true if a real audio player is available.
func (m Model) HasPlayer() bool {
	return m.audioPlayer != nil
}

// ChipInfo returns the chip information for the current track.
func (m Model) ChipInfo() []player.ChipInfo {
	return m.trackChips
}
