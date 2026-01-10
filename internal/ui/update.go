package ui

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dewi-tim/vgmtui/internal/ui/components"
)

// browserSelectNameMsg is used internally to select a specific entry by name.
type browserSelectNameMsg struct {
	name string
}

// Message types for the TUI.
type (
	// TickMsg is sent periodically to update the progress bar.
	TickMsg time.Time

	// PlayPauseMsg toggles playback state.
	PlayPauseMsg struct{}

	// NextTrackMsg advances to the next track.
	NextTrackMsg struct{}

	// PrevTrackMsg goes to the previous track.
	PrevTrackMsg struct{}

	// StopMsg stops playback.
	StopMsg struct{}

	// SeekMsg seeks by a delta amount.
	SeekMsg struct {
		Delta time.Duration
	}

	// ToggleHelpMsg toggles the help overlay.
	ToggleHelpMsg struct{}

	// FocusMsg changes the focused panel.
	FocusMsg Focus

	// QuitMsg signals the application should quit.
	QuitMsg struct{}

	// AddToQueueMsg adds tracks to the playlist queue.
	AddToQueueMsg struct {
		Tracks []Track
	}

	// RemoveFromQueueMsg removes the selected track from the queue.
	RemoveFromQueueMsg struct{}

	// ClearQueueMsg clears the entire playlist queue.
	ClearQueueMsg struct{}

	// PlaySelectedMsg starts playing the selected track in the playlist.
	PlaySelectedMsg struct{}
)

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.progress.SetWidth(msg.Width - 20) // Leave room for borders

		// Update browser size
		libraryWidth := m.width * libraryWidthPercent / 100
		browserHeight := m.height - 4 // Account for borders and footer
		m.browser.SetSize(libraryWidth-4, browserHeight-4) // Account for panel borders/padding

		// Update playlist size
		rightWidth := m.width - libraryWidth - 3
		playlistHeight := (m.height - 4) * 50 / 100
		m.playlist.SetSize(rightWidth-4, playlistHeight-2)

		return m, nil

	case tea.KeyMsg:
		// Handle key presses
		return m.handleKeyMsg(msg)

	case components.BrowserReadDirMsg:
		// Forward to browser
		var cmd tea.Cmd
		m.browser, cmd = m.browser.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case components.FileSelectedMsg:
		// A file was selected in the browser
		// For now, just update the current track display (mock)
		m.currentTrack = &Track{
			Path:     msg.Path,
			Title:    filepath.Base(msg.Path),
			Game:     "(loading...)",
			System:   "(unknown)",
			Composer: "(unknown)",
			Duration: 0,
		}
		// In the future, this would add to queue or start playback
		return m, nil

	case components.DirChangedMsg:
		// Directory changed - nothing special to do for now
		return m, nil

	case browserSelectNameMsg:
		// Internal message to select a specific entry by name
		m.browser.HandleSelectName(msg.name)
		return m, nil

	case TickMsg:
		// Update progress during playback
		if m.playback.State == StatePlaying {
			m.playback.Position += 100 * time.Millisecond
			if m.playback.Position > m.playback.Duration {
				// Loop or stop
				if m.playback.CurrentLoop < m.playback.TotalLoops-1 {
					m.playback.CurrentLoop++
					m.playback.Position = 0
				} else {
					m.playback.State = StateStopped
					m.playback.Position = 0
					m.playback.CurrentLoop = 0
				}
			}
		}
		// Continue ticking
		cmds = append(cmds, tickCmd())

	case PlayPauseMsg:
		return m.togglePlayPause()

	case NextTrackMsg:
		// Placeholder - will be implemented with actual playlist
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case PrevTrackMsg:
		// Placeholder - will be implemented with actual playlist
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case StopMsg:
		m.playback.State = StateStopped
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case SeekMsg:
		m.playback.Position += msg.Delta
		if m.playback.Position < 0 {
			m.playback.Position = 0
		}
		if m.playback.Position > m.playback.Duration {
			m.playback.Position = m.playback.Duration
		}
		return m, nil

	case ToggleHelpMsg:
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil

	case FocusMsg:
		m.focus = Focus(msg)
		return m, nil

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit

	case AddToQueueMsg:
		m.playlist.AddTracks(msg.Tracks)
		return m, nil

	case RemoveFromQueueMsg:
		m.playlist.RemoveSelected()
		return m, nil

	case ClearQueueMsg:
		m.playlist.Clear()
		return m, nil

	case PlaySelectedMsg:
		idx := m.playlist.SelectedIndex()
		if track := m.playlist.GetTrack(idx); track != nil {
			m.playlist.SetCurrentTrack(idx)
			m.currentTrack = track
			m.playback.State = StatePlaying
			m.playback.Position = 0
			m.playback.Duration = track.Duration
			m.playback.CurrentLoop = 0
		}
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg processes keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global key bindings (work regardless of focus)
	switch {
	case key.Matches(msg, m.keyMap.Quit):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keyMap.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil

	case key.Matches(msg, m.keyMap.PlayPause):
		return m.togglePlayPause()

	case key.Matches(msg, m.keyMap.NextTrack):
		// Placeholder
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case key.Matches(msg, m.keyMap.PrevTrack):
		// Placeholder
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case key.Matches(msg, m.keyMap.Stop):
		m.playback.State = StateStopped
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case key.Matches(msg, m.keyMap.SeekForward):
		m.playback.Position += 5 * time.Second
		if m.playback.Position > m.playback.Duration {
			m.playback.Position = m.playback.Duration
		}
		return m, nil

	case key.Matches(msg, m.keyMap.SeekBackward):
		m.playback.Position -= 5 * time.Second
		if m.playback.Position < 0 {
			m.playback.Position = 0
		}
		return m, nil

	case key.Matches(msg, m.keyMap.TabFocus):
		// Cycle focus between panels
		if m.focus == FocusBrowser {
			m.focus = FocusPlaylist
			m.browser.Blur()
			m.playlist.Focus()
		} else {
			m.focus = FocusBrowser
			m.playlist.Blur()
			m.browser.Focus()
		}
		return m, nil
	}

	// Panel-specific key handling
	switch m.focus {
	case FocusBrowser:
		// Forward navigation keys to browser
		var cmd tea.Cmd
		m.browser, cmd = m.browser.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	case FocusPlaylist:
		// Check for playlist-specific actions first
		playlistKeyMap := m.playlist.KeyMap()
		switch {
		case key.Matches(msg, playlistKeyMap.Select):
			// Play the selected track
			idx := m.playlist.SelectedIndex()
			if track := m.playlist.GetTrack(idx); track != nil {
				m.playlist.SetCurrentTrack(idx)
				m.currentTrack = track
				m.playback.State = StatePlaying
				m.playback.Position = 0
				m.playback.Duration = track.Duration
				m.playback.CurrentLoop = 0
			}
			return m, nil
		case key.Matches(msg, playlistKeyMap.Remove):
			m.playlist.RemoveSelected()
			return m, nil
		case key.Matches(msg, playlistKeyMap.Clear):
			m.playlist.Clear()
			return m, nil
		default:
			// Forward navigation keys to playlist
			var cmd tea.Cmd
			m.playlist, cmd = m.playlist.Update(msg)
			if cmd != nil {
				return m, cmd
			}
		}
	}

	return m, nil
}

// togglePlayPause toggles between playing and paused states.
func (m Model) togglePlayPause() (tea.Model, tea.Cmd) {
	switch m.playback.State {
	case StateStopped:
		m.playback.State = StatePlaying
		m.playback.Position = 0
	case StatePlaying:
		m.playback.State = StatePaused
	case StatePaused:
		m.playback.State = StatePlaying
	}
	return m, nil
}
