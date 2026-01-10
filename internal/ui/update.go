package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

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
		return m, nil

	case tea.KeyMsg:
		// Handle key presses
		return m.handleKeyMsg(msg)

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
		} else {
			m.focus = FocusBrowser
		}
		return m, nil
	}

	// Panel-specific key handling (for future)
	switch m.focus {
	case FocusBrowser:
		// Browser navigation will go here
	case FocusPlaylist:
		// Playlist navigation will go here
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
