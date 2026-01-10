package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

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
	browser   components.Browser
	playlist  components.Playlist
	progress  components.ProgressBar
	help      help.Model
	helpPopup components.HelpPopup

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
	// Initialize browser with home directory
	browser := components.NewBrowser("")
	browser.Focus() // Start with browser focused

	// Initialize empty playlist
	playlist := components.NewPlaylist()

	m := Model{
		focus:       FocusBrowser,
		browser:     browser,
		playlist:    playlist,
		progress:    components.NewProgressBar(),
		help:        help.New(),
		helpPopup:   components.NewHelpPopup(),
		keyMap:      DefaultKeyMap(),
		styles:      DefaultStyles(),
		audioPlayer: ap,
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
	cmds := []tea.Cmd{
		m.browser.Init(),
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
			return nil
		}
		return PlayerTickMsg{Info: info}
	}
}

// PlayerTickMsg is sent when the audio player provides a playback update.
type PlayerTickMsg struct {
	Info player.PlaybackInfo
}

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
