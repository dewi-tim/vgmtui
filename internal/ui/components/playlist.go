// Package components provides UI components for vgmtui.
package components

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Track represents a track in the playlist.
// This is a simplified version to avoid circular imports with the player package.
type Track struct {
	Path     string
	Title    string
	Game     string
	System   string
	Composer string
	Duration time.Duration
}

// PlaylistKeyMap defines keybindings for the playlist component.
type PlaylistKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Select   key.Binding
	Remove   key.Binding
	Clear    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	MoveUp   key.Binding
	MoveDown key.Binding
	Shuffle  key.Binding
	LoopMode key.Binding
}

// DefaultPlaylistKeyMap returns the default keybindings for the playlist.
func DefaultPlaylistKeyMap() PlaylistKeyMap {
	return PlaylistKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "bottom"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", "l"),
			key.WithHelp("enter/l", "play"),
		),
		Remove: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "remove"),
		),
		Clear: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "clear"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "move up"),
		),
		MoveDown: key.NewBinding(
			key.WithKeys("J"),
			key.WithHelp("J", "move down"),
		),
		Shuffle: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "shuffle"),
		),
		LoopMode: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "loop mode"),
		),
	}
}

// LoopMode represents the playlist loop behavior.
type LoopMode int

const (
	LoopNone LoopMode = iota // No looping - stop at end
	LoopOne                  // Loop current track
	LoopAll                  // Loop entire playlist
)

// Playlist manages a queue of tracks to play.
type Playlist struct {
	table   table.Model
	tracks  []Track
	current int // Currently playing index (-1 if none)
	focused bool

	keyMap   PlaylistKeyMap
	loopMode LoopMode

	// Dimensions
	width  int
	height int

	// Styles
	styles PlaylistStyles
}

// PlaylistStyles defines the styles for the playlist component.
type PlaylistStyles struct {
	// Table styles
	Header   lipgloss.Style
	Cell     lipgloss.Style
	Selected lipgloss.Style
	Playing  lipgloss.Style

	// Panel styles
	FocusedBorder lipgloss.Style
	NormalBorder  lipgloss.Style
	Title         lipgloss.Style
	TitleMuted    lipgloss.Style
}

// DefaultPlaylistStyles returns the default styles for the playlist.
func DefaultPlaylistStyles() PlaylistStyles {
	return PlaylistStyles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A0A0A0")).
			Padding(0, 1),
		Cell: lipgloss.NewStyle().
			Padding(0, 1),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7571F9")),
		Playing: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575")),
		FocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7571F9")),
		NormalBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#606060")),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true),
		TitleMuted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")),
	}
}

// NewPlaylist creates a new Playlist component.
func NewPlaylist() Playlist {
	columns := []table.Column{
		{Title: "#", Width: 5},         // Track number with "> " indicator
		{Title: "Duration", Width: 8},
		{Title: "Title", Width: 20},
		{Title: "Game", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(false),
		table.WithHeight(5),
	)

	// Apply default table styles
	s := table.DefaultStyles()
	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#A0A0A0")).
		Padding(0, 1)
	s.Cell = lipgloss.NewStyle().
		Padding(0, 1)
	s.Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9"))
	t.SetStyles(s)

	return Playlist{
		table:   t,
		tracks:  []Track{},
		current: -1,
		focused: false,
		keyMap:  DefaultPlaylistKeyMap(),
		styles:  DefaultPlaylistStyles(),
		width:   40,
		height:  10,
	}
}

// Update handles messages for the playlist.
func (p Playlist) Update(msg tea.Msg) (Playlist, tea.Cmd) {
	// Always handle window size messages regardless of focus
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		var cmd tea.Cmd
		p.table, cmd = p.table.Update(msg)
		return p, cmd
	}

	if !p.focused {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle navigation keys directly - don't pass to table
		switch {
		case key.Matches(msg, p.keyMap.Up):
			p.table.MoveUp(1)
			return p, nil
		case key.Matches(msg, p.keyMap.Down):
			p.table.MoveDown(1)
			return p, nil
		case key.Matches(msg, p.keyMap.Top):
			p.table.GotoTop()
			return p, nil
		case key.Matches(msg, p.keyMap.Bottom):
			p.table.GotoBottom()
			return p, nil
		case key.Matches(msg, p.keyMap.PageUp):
			p.table.MoveUp(p.table.Height())
			return p, nil
		case key.Matches(msg, p.keyMap.PageDown):
			p.table.MoveDown(p.table.Height())
			return p, nil
		}
	}

	// Only pass non-navigation messages to the table
	var cmd tea.Cmd
	p.table, cmd = p.table.Update(msg)
	return p, cmd
}

// View renders the playlist.
func (p Playlist) View() string {
	return p.table.View()
}

// SetSize sets the size of the playlist component.
func (p *Playlist) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Calculate column widths based on available space
	// #: 5 (track number with "> " indicator), Duration: 8, Title: flexible, Game: ~25%
	availableWidth := width - 6 // Account for borders and padding
	if availableWidth < 30 {
		availableWidth = 30
	}

	numWidth := 5       // Track number with "> " indicator
	durationWidth := 8  // Duration without indicator
	gameWidth := availableWidth * 25 / 100
	if gameWidth < 8 {
		gameWidth = 8
	}
	titleWidth := availableWidth - numWidth - durationWidth - gameWidth
	if titleWidth < 10 {
		titleWidth = 10
	}

	columns := []table.Column{
		{Title: "#", Width: numWidth},
		{Title: "Duration", Width: durationWidth},
		{Title: "Title", Width: titleWidth},
		{Title: "Game", Width: gameWidth},
	}
	p.table.SetColumns(columns)
	p.table.SetWidth(availableWidth)

	// Height minus header row and borders
	tableHeight := height - 4
	if tableHeight < 1 {
		tableHeight = 1
	}
	p.table.SetHeight(tableHeight)
}

// Focus sets the playlist to focused state.
func (p *Playlist) Focus() {
	p.focused = true
	p.table.Focus()
}

// Blur removes focus from the playlist.
func (p *Playlist) Blur() {
	p.focused = false
	p.table.Blur()
}

// Focused returns whether the playlist is focused.
func (p Playlist) Focused() bool {
	return p.focused
}

// AddTrack adds a single track to the playlist.
func (p *Playlist) AddTrack(track Track) {
	p.tracks = append(p.tracks, track)
	p.updateTableRows()
}

// AddTracks adds multiple tracks to the playlist.
func (p *Playlist) AddTracks(tracks []Track) {
	p.tracks = append(p.tracks, tracks...)
	p.updateTableRows()
}

// RemoveSelected removes the currently selected track from the playlist.
func (p *Playlist) RemoveSelected() {
	if len(p.tracks) == 0 {
		return
	}

	idx := p.table.Cursor()
	if idx < 0 || idx >= len(p.tracks) {
		return
	}

	// Remove the track
	p.tracks = append(p.tracks[:idx], p.tracks[idx+1:]...)

	// Adjust current playing index if needed
	if p.current >= 0 {
		if idx < p.current {
			p.current--
		} else if idx == p.current {
			p.current = -1 // Removed the currently playing track
		}
	}

	p.updateTableRows()

	// Adjust cursor if it's now out of bounds
	if idx >= len(p.tracks) && len(p.tracks) > 0 {
		p.table.SetCursor(len(p.tracks) - 1)
	}
}

// Clear removes all tracks from the playlist.
func (p *Playlist) Clear() {
	p.tracks = []Track{}
	p.current = -1
	p.updateTableRows()
}

// SetCurrentTrack sets the index of the currently playing track.
func (p *Playlist) SetCurrentTrack(index int) {
	if index < -1 {
		index = -1
	}
	if index >= len(p.tracks) {
		index = -1
	}
	p.current = index
	p.updateTableRows()
}

// SelectedIndex returns the index of the currently selected (highlighted) track.
func (p Playlist) SelectedIndex() int {
	return p.table.Cursor()
}

// GetTrack returns a copy of the track at the given index, or nil if out of bounds.
// Note: Returns a new pointer to a copy, not a pointer into the slice.
func (p Playlist) GetTrack(index int) *Track {
	if index < 0 || index >= len(p.tracks) {
		return nil
	}
	track := p.tracks[index] // Copy the track
	return &track
}

// SelectedTrack returns a copy of the currently selected track, or nil if none.
func (p Playlist) SelectedTrack() *Track {
	return p.GetTrack(p.SelectedIndex())
}

// CurrentTrack returns a copy of the currently playing track, or nil if none.
func (p Playlist) CurrentTrack() *Track {
	return p.GetTrack(p.current)
}

// CurrentIndex returns the index of the currently playing track (-1 if none).
func (p Playlist) CurrentIndex() int {
	return p.current
}

// Len returns the number of tracks in the playlist.
func (p Playlist) Len() int {
	return len(p.tracks)
}

// Tracks returns a copy of all tracks in the playlist.
func (p Playlist) Tracks() []Track {
	result := make([]Track, len(p.tracks))
	copy(result, p.tracks)
	return result
}

// updateTableRows syncs the table rows with the tracks slice.
func (p *Playlist) updateTableRows() {
	// Save cursor position before updating rows
	savedCursor := p.table.Cursor()

	rows := make([]table.Row, len(p.tracks))
	for i, track := range p.tracks {
		// Format track number with playing indicator
		var trackNum string
		if i == p.current {
			// Use play symbol as indicator (visible in all terminals)
			trackNum = fmt.Sprintf(">%d", i+1)
		} else {
			trackNum = fmt.Sprintf(" %d", i+1)
		}

		// Format duration
		duration := formatDuration(track.Duration)

		rows[i] = table.Row{trackNum, duration, track.Title, track.Game}
	}
	p.table.SetRows(rows)

	// Restore cursor position if still valid
	if savedCursor >= 0 && savedCursor < len(rows) {
		p.table.SetCursor(savedCursor)
	} else if len(rows) > 0 {
		p.table.SetCursor(0)
	}
}

// Title returns the title for the playlist panel.
func (p Playlist) Title() string {
	if len(p.tracks) == 0 {
		return "Playlist"
	}
	if p.current >= 0 {
		return fmt.Sprintf("Playlist [%d/%d]", p.current+1, len(p.tracks))
	}
	return fmt.Sprintf("Playlist [%d]", len(p.tracks))
}

// KeyMap returns the playlist's keymap for help display.
func (p Playlist) KeyMap() PlaylistKeyMap {
	return p.keyMap
}

// IsEmpty returns true if the playlist has no tracks.
func (p Playlist) IsEmpty() bool {
	return len(p.tracks) == 0
}

// NextTrack advances to the next track, returning its index or -1 if at end.
// Honors LoopAll mode by wrapping around to the beginning.
func (p *Playlist) NextTrack() int {
	if len(p.tracks) == 0 {
		return -1
	}
	if p.current < 0 {
		p.current = 0
	} else if p.current < len(p.tracks)-1 {
		p.current++
	} else {
		// At end of playlist - check loop mode
		if p.loopMode == LoopAll {
			p.current = 0 // Wrap to beginning
		} else {
			return -1 // At end, no looping
		}
	}
	p.updateTableRows()
	return p.current
}

// PrevTrack goes to the previous track, returning its index or -1 if at start.
// Honors LoopAll mode by wrapping around to the end.
func (p *Playlist) PrevTrack() int {
	if len(p.tracks) == 0 {
		return -1
	}
	if p.current <= 0 {
		// At start of playlist - check loop mode
		if p.loopMode == LoopAll && len(p.tracks) > 0 {
			p.current = len(p.tracks) - 1 // Wrap to end
		} else {
			return -1 // At start, no looping
		}
	} else {
		p.current--
	}
	p.updateTableRows()
	return p.current
}

// MoveUp moves the selected track up in the playlist.
func (p *Playlist) MoveUp() {
	idx := p.table.Cursor()
	if idx <= 0 || idx >= len(p.tracks) {
		return
	}

	// Swap tracks
	p.tracks[idx], p.tracks[idx-1] = p.tracks[idx-1], p.tracks[idx]

	// Adjust current playing index if affected
	if p.current == idx {
		p.current--
	} else if p.current == idx-1 {
		p.current++
	}

	p.updateTableRows()
	p.table.SetCursor(idx - 1)
}

// MoveDown moves the selected track down in the playlist.
func (p *Playlist) MoveDown() {
	idx := p.table.Cursor()
	if idx < 0 || idx >= len(p.tracks)-1 {
		return
	}

	// Swap tracks
	p.tracks[idx], p.tracks[idx+1] = p.tracks[idx+1], p.tracks[idx]

	// Adjust current playing index if affected
	if p.current == idx {
		p.current++
	} else if p.current == idx+1 {
		p.current--
	}

	p.updateTableRows()
	p.table.SetCursor(idx + 1)
}

// Shuffle randomizes the order of tracks in the playlist.
func (p *Playlist) Shuffle() {
	if len(p.tracks) <= 1 {
		return
	}

	// Remember the currently playing track
	var currentTrack *Track
	if p.current >= 0 && p.current < len(p.tracks) {
		currentTrack = &p.tracks[p.current]
	}

	// Fisher-Yates shuffle
	for i := len(p.tracks) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		p.tracks[i], p.tracks[j] = p.tracks[j], p.tracks[i]
	}

	// Find and update the current track index
	if currentTrack != nil {
		for i, t := range p.tracks {
			if t.Path == currentTrack.Path {
				p.current = i
				break
			}
		}
	}

	p.updateTableRows()
}

// CycleLoopMode cycles through the loop modes: None -> One -> All -> None.
func (p *Playlist) CycleLoopMode() {
	p.loopMode = (p.loopMode + 1) % 3
}

// LoopMode returns the current loop mode.
func (p Playlist) LoopMode() LoopMode {
	return p.loopMode
}

// LoopModeString returns a string representation of the current loop mode.
func (p Playlist) LoopModeString() string {
	switch p.loopMode {
	case LoopOne:
		return "1"
	case LoopAll:
		return "A"
	default:
		return "-"
	}
}
