package ui

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dewi-tim/vgmtui/internal/player"
	"github.com/dewi-tim/vgmtui/internal/ui/components"
)

// browserSelectNameMsg is used internally to select a specific entry by name.
type browserSelectNameMsg struct {
	name string
}

// Message types for the TUI.
type (
	// TickMsg is sent periodically to update the progress bar (mock mode only).
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

	// LoadTrackMetadataMsg is sent when a file is selected in the browser.
	// It triggers loading metadata from the audio player.
	LoadTrackMetadataMsg struct {
		Path string
	}

	// TrackMetadataLoadedMsg is sent when track metadata has been loaded.
	TrackMetadataLoadedMsg struct {
		Track Track
		Chips []player.ChipInfo
	}

	// ErrorMsg is sent when an error occurs that should be displayed to the user.
	ErrorMsg struct {
		Err error
	}

	// ClearErrorMsg is sent to clear the error display.
	ClearErrorMsg struct{}
)

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

		// Match the layout calculations from View()
		footerHeight := 1
		mainHeight := m.height - footerHeight

		// Panel widths
		libraryWidth := m.width * libraryWidthPercent / 100
		rightWidth := m.width - libraryWidth

		// Browser size: outer=libraryWidth x mainHeight, inner subtracts border(2) and title(1)
		browserInnerWidth := libraryWidth - 2
		browserInnerHeight := mainHeight - 3 // border(2) + title(1)
		m.browser.SetSize(browserInnerWidth, browserInnerHeight)

		// Right pane layout (from renderRightPane)
		progressHeight := 5
		trackInfoHeight := 6
		playlistHeight := mainHeight - progressHeight - trackInfoHeight

		// Playlist size: inner dimensions
		playlistInnerWidth := rightWidth - 2
		playlistInnerHeight := playlistHeight - 3 // border(2) + title(1)
		m.playlist.SetSize(playlistInnerWidth, playlistInnerHeight)

		// Progress bar width (inside progress panel)
		progressInnerWidth := rightWidth - 4 // border + some padding
		m.progress.SetWidth(progressInnerWidth)

		// Help popup
		m.helpPopup.SetSize(msg.Width, msg.Height)

		return m, nil

	case tea.KeyMsg:
		// If help popup is visible, only handle help popup keys
		if m.helpPopup.Visible() {
			var cmd tea.Cmd
			m.helpPopup, cmd = m.helpPopup.Update(msg)
			return m, cmd
		}
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
		if m.audioPlayer != nil {
			// Load metadata from the real player
			return m, loadTrackMetadata(m.audioPlayer, msg.Path)
		}
		// Mock mode: just show filename
		m.currentTrack = &Track{
			Path:     msg.Path,
			Title:    filepath.Base(msg.Path),
			Game:     "(no player)",
			System:   "(unknown)",
			Composer: "(unknown)",
			Duration: 0,
		}
		return m, nil

	case TrackMetadataLoadedMsg:
		// Track metadata has been loaded from the player
		// Add to playlist and set as current display
		m.playlist.AddTrack(msg.Track)
		m.currentTrack = &msg.Track
		m.trackChips = msg.Chips
		return m, nil

	case components.DirChangedMsg:
		// Directory changed - nothing special to do for now
		return m, nil

	case browserSelectNameMsg:
		// Internal message to select a specific entry by name
		m.browser.HandleSelectName(msg.name)
		return m, nil

	case PlayerTickMsg:
		// Update from real audio player
		wasPlaying := m.playback.State == StatePlaying
		m.playback.Position = msg.Info.Position
		m.playback.Duration = msg.Info.Duration
		m.playback.CurrentLoop = msg.Info.CurrentLoop
		m.playback.TotalLoops = msg.Info.TotalLoops

		// Convert player state to UI state
		switch msg.Info.State {
		case player.StatePlaying:
			m.playback.State = StatePlaying
		case player.StatePaused:
			m.playback.State = StatePaused
		case player.StateStopped:
			m.playback.State = StateStopped
			// Check if track ended (was playing, now stopped)
			if wasPlaying {
				// Auto-advance to next track
				return m, func() tea.Msg { return TrackEndedMsg{} }
			}
		case player.StateFading:
			m.playback.State = StatePlaying // Show as playing during fade
		}

		// Continue listening for playback updates
		if m.playerSub != nil {
			cmds = append(cmds, listenForPlayback(m.playerSub))
		}

	case TrackEndedMsg:
		// Current track finished, try to play next
		if m.audioPlayer != nil {
			nextIdx := m.playlist.NextTrack()
			if nextIdx >= 0 {
				if track := m.playlist.GetTrack(nextIdx); track != nil {
					m.currentTrack = track
					return m, playTrack(m.audioPlayer, track.Path)
				}
			}
		}
		// No next track or no player
		m.playback.State = StateStopped
		m.playback.Position = 0
		return m, nil

	case TickMsg:
		// Mock tick - only used when no real player
		// This is kept for backwards compatibility but won't be started
		// in normal operation with a real player
		return m, nil

	case PlayPauseMsg:
		return m.togglePlayPause()

	case NextTrackMsg:
		// Advance to next track in playlist
		if m.audioPlayer != nil {
			m.audioPlayer.Stop()
			nextIdx := m.playlist.NextTrack()
			if nextIdx >= 0 {
				if track := m.playlist.GetTrack(nextIdx); track != nil {
					m.currentTrack = track
					return m, playTrack(m.audioPlayer, track.Path)
				}
			}
		}
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case PrevTrackMsg:
		// Go to previous track in playlist
		if m.audioPlayer != nil {
			m.audioPlayer.Stop()
			prevIdx := m.playlist.PrevTrack()
			if prevIdx >= 0 {
				if track := m.playlist.GetTrack(prevIdx); track != nil {
					m.currentTrack = track
					return m, playTrack(m.audioPlayer, track.Path)
				}
			}
		}
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case StopMsg:
		if m.audioPlayer != nil {
			m.audioPlayer.Stop()
		}
		m.playback.State = StateStopped
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case SeekMsg:
		if m.audioPlayer != nil {
			m.audioPlayer.SeekRelative(msg.Delta)
		} else {
			// Mock mode
			m.playback.Position += msg.Delta
			if m.playback.Position < 0 {
				m.playback.Position = 0
			}
			if m.playback.Position > m.playback.Duration {
				m.playback.Position = m.playback.Duration
			}
		}
		return m, nil

	case ToggleHelpMsg:
		m.helpPopup.Toggle()
		m.showHelp = m.helpPopup.Visible()
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
			if m.audioPlayer != nil {
				// Load and play the track
				return m, playTrack(m.audioPlayer, track.Path)
			}
			// Mock mode
			m.playback.State = StatePlaying
			m.playback.Position = 0
			m.playback.Duration = track.Duration
			m.playback.CurrentLoop = 0
		}
		return m, nil

	case ErrorMsg:
		m.lastError = msg.Err.Error()
		m.errorTime = time.Now()
		// Schedule auto-clear after 5 seconds
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return ClearErrorMsg{}
		})

	case ClearErrorMsg:
		// Only clear if the error is old (5+ seconds)
		if time.Since(m.errorTime) >= 5*time.Second {
			m.lastError = ""
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
		m.helpPopup.Toggle()
		m.showHelp = m.helpPopup.Visible()
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
				if m.audioPlayer != nil {
					return m, playTrack(m.audioPlayer, track.Path)
				}
				// Mock mode
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
	if m.audioPlayer != nil {
		switch m.playback.State {
		case StateStopped:
			// If we have a current track, start playing it
			if m.currentTrack != nil {
				return m, playTrack(m.audioPlayer, m.currentTrack.Path)
			}
			// Otherwise try to play the first track in playlist
			if track := m.playlist.GetTrack(0); track != nil {
				m.playlist.SetCurrentTrack(0)
				m.currentTrack = track
				return m, playTrack(m.audioPlayer, track.Path)
			}
		case StatePlaying:
			m.audioPlayer.Pause()
		case StatePaused:
			m.audioPlayer.Play()
		}
		return m, nil
	}

	// Mock mode
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

// loadTrackMetadata returns a command that loads track metadata without
// affecting the current playback state. It uses a separate player instance
// to read metadata, so it can be called while music is playing.
func loadTrackMetadata(ap *player.AudioPlayer, path string) tea.Cmd {
	// Note: ap is not used directly here anymore, but kept for API consistency
	_ = ap
	return func() tea.Msg {
		// Read metadata using a temporary player instance
		track, err := player.ReadTrackMetadata(path)
		if err != nil {
			// Return error along with basic track info
			return tea.Batch(
				func() tea.Msg {
					return ErrorMsg{Err: err}
				},
				func() tea.Msg {
					return TrackMetadataLoadedMsg{
						Track: Track{
							Path:  path,
							Title: filepath.Base(path),
							Game:  "(load error)",
						},
					}
				},
			)()
		}

		// Convert player.Track to components.Track
		return TrackMetadataLoadedMsg{
			Track: Track{
				Path:     track.Path,
				Title:    defaultString(track.Title, filepath.Base(path)),
				Game:     track.Game,
				System:   track.System,
				Composer: track.Composer,
				Duration: track.Duration,
			},
			Chips: track.Chips,
		}
	}
}

// playTrack returns a command that loads and plays a track.
func playTrack(ap *player.AudioPlayer, path string) tea.Cmd {
	return func() tea.Msg {
		if err := ap.Load(path); err != nil {
			return ErrorMsg{Err: err}
		}
		if err := ap.Play(); err != nil {
			return ErrorMsg{Err: err}
		}
		return nil
	}
}

// defaultString returns s if non-empty, otherwise returns def.
func defaultString(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
