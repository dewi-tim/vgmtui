package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

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
	browser  components.Browser
	playlist components.Playlist
	progress components.ProgressBar
	help     help.Model

	// Key bindings
	keyMap KeyMap

	// UI state
	showHelp bool
	quitting bool

	// Playback state (mocked for now)
	playback     PlaybackInfo
	currentTrack *Track

	// Styles
	styles Styles
}

// New creates a new Model with default values.
func New() Model {
	// Initialize browser with home directory
	browser := components.NewBrowser("")
	browser.Focus() // Start with browser focused

	// Initialize playlist with some mock tracks
	playlist := components.NewPlaylist()
	playlist.AddTracks([]Track{
		{
			Path:     "/music/sonic1/green_hill.vgm",
			Title:    "Green Hill Zone",
			Game:     "Sonic 1",
			System:   "Genesis",
			Composer: "Masato Nakamura",
			Duration: 2*time.Minute + 34*time.Second,
		},
		{
			Path:     "/music/sonic1/marble.vgm",
			Title:    "Marble Zone",
			Game:     "Sonic 1",
			System:   "Genesis",
			Composer: "Masato Nakamura",
			Duration: 3*time.Minute + 1*time.Second,
		},
		{
			Path:     "/music/sonic1/starlight.vgm",
			Title:    "Star Light Zone",
			Game:     "Sonic 1",
			System:   "Genesis",
			Composer: "Masato Nakamura",
			Duration: 1*time.Minute + 45*time.Second,
		},
	})
	playlist.SetCurrentTrack(1) // Set Marble Zone as currently playing

	return Model{
		focus:    FocusBrowser,
		browser:  browser,
		playlist: playlist,
		progress: components.NewProgressBar(),
		help:     help.New(),
		keyMap:   DefaultKeyMap(),
		styles:   DefaultStyles(),
		playback: PlaybackInfo{
			State:      StateStopped,
			Duration:   3*time.Minute + 1*time.Second, // 3:01 mock duration
			TotalLoops: 2,
		},
		currentTrack: &Track{
			Title:    "Marble Zone",
			Game:     "Sonic the Hedgehog",
			System:   "Sega Genesis",
			Composer: "Masato Nakamura",
			Duration: 3*time.Minute + 1*time.Second,
		},
	}
}

// Init returns the initial command to run.
func (m Model) Init() tea.Cmd {
	// Start a tick for updating the progress bar during playback
	// Also initialize the browser
	return tea.Batch(
		tickCmd(),
		m.browser.Init(),
	)
}

// tickCmd returns a command that ticks every 100ms for smooth progress updates.
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

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
