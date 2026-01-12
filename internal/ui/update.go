package ui

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dewi-tim/vgmtui/internal/library"
	"github.com/dewi-tim/vgmtui/internal/player"
	"github.com/dewi-tim/vgmtui/internal/ui/components"
)

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

	// TrackMetadataForPlayMsg is sent when track metadata has been loaded and should play immediately.
	TrackMetadataForPlayMsg struct {
		Track Track
		Chips []player.ChipInfo
	}

	// ErrorMsg is sent when an error occurs that should be displayed to the user.
	ErrorMsg struct {
		Err error
	}

	// ClearErrorMsg is sent to clear the error display.
	ClearErrorMsg struct{}

	// TrackChipsLoadedMsg is sent when chip info is loaded for the current track.
	TrackChipsLoadedMsg struct {
		Chips []player.ChipInfo
	}

	// TrackLoadStartedMsg is sent when a playTrack command begins.
	TrackLoadStartedMsg struct{}

	// TrackLoadCompleteMsg is sent when a playTrack command completes (success or failure).
	TrackLoadCompleteMsg struct{}
)

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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
		if m.useLibrary {
			m.libBrowser.SetSize(browserInnerWidth, browserInnerHeight)
		}

		// Right pane layout (from renderRightPane)
		progressHeight := 4  // No title now
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

	case components.LibBrowserScanCompleteMsg:
		// Library scan completed
		if msg.Err != nil {
			// Propagate scan error to UI
			m.lastError = "Library scan failed: " + msg.Err.Error()
			m.errorTime = time.Now()
		}
		if m.useLibrary {
			var cmd tea.Cmd
			m.libBrowser, cmd = m.libBrowser.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case components.LibTrackSelectedMsg:
		// Single track selected from library (just adds to playlist, doesn't play)
		m.playlist.AddTrack(components.Track{
			Path:        msg.Track.Path,
			Title:       msg.Track.Title,
			Game:        msg.Track.Game,
			System:      msg.Track.System,
			Composer:    msg.Track.Composer,
			Duration:    msg.Track.Duration,
			TrackNumber: msg.Track.TrackNumber,
		})
		return m, nil

	case components.LibTracksSelectedMsg:
		// Multiple tracks selected (add all from game/system)
		for _, t := range msg.Tracks {
			m.playlist.AddTrack(components.Track{
				Path:        t.Path,
				Title:       t.Title,
				Game:        t.Game,
				System:      t.System,
				Composer:    t.Composer,
				Duration:    t.Duration,
				TrackNumber: t.TrackNumber,
			})
		}
		return m, nil

	case components.LibTrackPlayMsg:
		// Track selected for immediate playback - add to playlist and play
		if m.trackLoading {
			return m, nil
		}
		track := components.Track{
			Path:        msg.Track.Path,
			Title:       msg.Track.Title,
			Game:        msg.Track.Game,
			System:      msg.Track.System,
			Composer:    msg.Track.Composer,
			Duration:    msg.Track.Duration,
			TrackNumber: msg.Track.TrackNumber,
		}
		m.playlist.AddTrack(track)
		// Use startPlayingTrack for atomic state transition
		newIdx := m.playlist.Len() - 1
		cmd := m.startPlayingTrack(newIdx)
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case components.FileSelectedMsg:
		// A file was selected in the browser (add only, no play)
		if m.audioPlayer != nil {
			// Load metadata from the real player
			return m, loadTrackMetadata(msg.Path)
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

	case components.FilePlayMsg:
		// A file was selected for immediate playback (add and play)
		if m.audioPlayer != nil && !m.trackLoading {
			return m, loadTrackMetadataForPlay(msg.Path)
		}
		return m, nil

	case TrackMetadataLoadedMsg:
		// Track metadata has been loaded from the player
		// Just add to playlist
		m.playlist.AddTrack(msg.Track)
		m.trackChips = msg.Chips
		return m, nil

	case TrackMetadataForPlayMsg:
		// Track metadata loaded and should be played immediately
		if m.trackLoading {
			return m, nil
		}
		m.playlist.AddTrack(msg.Track)
		m.trackChips = msg.Chips
		// Start playing the newly added track
		newIdx := m.playlist.Len() - 1
		cmd := m.startPlayingTrack(newIdx)
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case components.DirChangedMsg:
		// Directory changed - nothing special to do for now
		return m, nil

	case components.BrowserSelectNameMsg:
		// Message to select a specific entry by name after navigating up
		m.browser.HandleSelectName(msg.Name)
		return m, nil

	case PlayerTickMsg:
		// Update from real audio player
		// Consider both Playing and Fading as "was playing" for auto-advance
		wasPlaying := m.playback.State == StatePlaying || m.playback.State == StateFading
		m.playback.Position = msg.Info.Position
		m.playback.Duration = msg.Info.Duration
		m.playback.CurrentLoop = msg.Info.CurrentLoop
		m.playback.TotalLoops = msg.Info.TotalLoops

		// Convert player state to UI state
		// When trackLoading is true, we're switching tracks - ignore StateStopped
		// to avoid briefly showing "Stopped" during the transition
		switch msg.Info.State {
		case player.StatePlaying:
			m.playback.State = StatePlaying
		case player.StatePaused:
			m.playback.State = StatePaused
		case player.StateStopped:
			// Only update to stopped if we're not in the middle of loading a new track
			if !m.trackLoading {
				m.playback.State = StateStopped
				// Check if track ended (was playing, now stopped)
				if wasPlaying {
					// Auto-advance to next track
					// Also continue listening for updates from the new track
					batchCmds := []tea.Cmd{func() tea.Msg { return TrackEndedMsg{} }}
					if m.playerSub != nil {
						batchCmds = append(batchCmds, listenForPlayback(m.playerSub))
					}
					return m, tea.Batch(batchCmds...)
				}
			}
		case player.StateFading:
			m.playback.State = StateFading
		}

		// Continue listening for playback updates
		if m.playerSub != nil {
			cmds = append(cmds, listenForPlayback(m.playerSub))
		}

	case PlaybackChannelClosedMsg:
		// Playback subscription channel was closed (player shutdown)
		// Clear the subscription so we don't try to re-subscribe
		m.playerSub = nil
		return m, nil

	case TrackEndedMsg:
		// Current track finished, try to play next
		if m.audioPlayer != nil && !m.trackLoading {
			// Use PeekNextTrack to query without mutating state
			nextIdx := m.playlist.PeekNextTrack()
			if nextIdx >= 0 {
				// Use startPlayingTrack for atomic state transition
				cmd := m.startPlayingTrack(nextIdx)
				if cmd != nil {
					return m, cmd
				}
			}
			// No next track or failed to start - stop playback and clear state
			m.stopPlayback()
		}
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
		if m.trackLoading {
			return m, nil
		}
		if m.audioPlayer != nil {
			// Use PeekNextTrack to query without mutating state
			nextIdx := m.playlist.PeekNextTrack()
			if nextIdx >= 0 {
				// Stop current playback and start next track
				m.audioPlayer.Stop()
				cmd := m.startPlayingTrack(nextIdx)
				if cmd != nil {
					return m, cmd
				}
			}
			// No next track available - just reset position
			m.playback.Position = 0
			m.playback.CurrentLoop = 0
		}
		return m, nil

	case PrevTrackMsg:
		// Go to previous track in playlist
		if m.trackLoading {
			return m, nil
		}
		if m.audioPlayer != nil {
			// Use PeekPrevTrack to query without mutating state
			prevIdx := m.playlist.PeekPrevTrack()
			if prevIdx >= 0 {
				// Stop current playback and start previous track
				m.audioPlayer.Stop()
				cmd := m.startPlayingTrack(prevIdx)
				if cmd != nil {
					return m, cmd
				}
			}
			// No previous track available - just reset position
			m.playback.Position = 0
			m.playback.CurrentLoop = 0
		}
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
		// Check if we're removing the currently playing track
		selectedIdx := m.playlist.SelectedIndex()
		currentIdx := m.playlist.CurrentIndex()
		wasPlayingRemoved := selectedIdx == currentIdx && currentIdx >= 0

		m.playlist.RemoveSelected()

		// If we removed the currently playing track, stop playback
		if wasPlayingRemoved {
			m.stopPlayback()
		}
		return m, nil

	case ClearQueueMsg:
		// Stop playback before clearing since we're removing all tracks
		m.stopPlayback()
		m.playlist.Clear()
		return m, nil

	case PlaySelectedMsg:
		if m.trackLoading {
			return m, nil
		}
		idx := m.playlist.SelectedIndex()
		if m.audioPlayer != nil {
			// Use startPlayingTrack for atomic state transition
			cmd := m.startPlayingTrack(idx)
			if cmd != nil {
				return m, cmd
			}
		} else {
			// Mock mode - set state directly
			if track := m.playlist.GetTrack(idx); track != nil {
				m.playlist.SetCurrentTrack(idx)
				m.currentTrack = track
				m.playback.State = StatePlaying
				m.playback.Position = 0
				m.playback.Duration = track.Duration
				m.playback.CurrentLoop = 0
			}
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

	case TrackChipsLoadedMsg:
		// Update chip info for current track
		m.trackChips = msg.Chips
		return m, nil

	case TrackLoadStartedMsg:
		// Mark that a track load is in progress
		m.trackLoading = true
		return m, nil

	case TrackLoadCompleteMsg:
		// Track load finished (success or failure)
		m.trackLoading = false
		return m, nil

	case playTrackResult:
		// Handle combined result from playTrack command
		if msg.err != nil {
			// Playback failed - rollback pending state
			m.cancelPendingTrack()
			m.lastError = msg.err.Error()
			m.errorTime = time.Now()
			return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return ClearErrorMsg{}
			})
		}
		// Playback succeeded - commit pending state
		m.confirmTrackStarted()
		if len(msg.chips) > 0 {
			m.trackChips = msg.chips
		}
		// Note: Don't queue listenForPlayback here - it's already queued
		// from the PlayerTickMsg handler (either in the early return for
		// auto-advance, or at the end for normal playback)
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
		if m.trackLoading {
			return m, nil
		}
		if m.audioPlayer != nil {
			// Use PeekNextTrack to query without mutating state
			nextIdx := m.playlist.PeekNextTrack()
			if nextIdx >= 0 {
				// Stop current playback and start next track
				m.audioPlayer.Stop()
				cmd := m.startPlayingTrack(nextIdx)
				if cmd != nil {
					return m, cmd
				}
			}
			// No next track available - just reset position
			m.playback.Position = 0
			m.playback.CurrentLoop = 0
		}
		return m, nil

	case key.Matches(msg, m.keyMap.PrevTrack):
		if m.trackLoading {
			return m, nil
		}
		if m.audioPlayer != nil {
			// Use PeekPrevTrack to query without mutating state
			prevIdx := m.playlist.PeekPrevTrack()
			if prevIdx >= 0 {
				// Stop current playback and start previous track
				m.audioPlayer.Stop()
				cmd := m.startPlayingTrack(prevIdx)
				if cmd != nil {
					return m, cmd
				}
			}
			// No previous track available - just reset position
			m.playback.Position = 0
			m.playback.CurrentLoop = 0
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Stop):
		if m.audioPlayer != nil {
			m.audioPlayer.Stop()
		}
		m.playback.State = StateStopped
		m.playback.Position = 0
		m.playback.CurrentLoop = 0
		return m, nil

	case key.Matches(msg, m.keyMap.SeekForward):
		if m.audioPlayer != nil {
			m.audioPlayer.SeekRelative(5 * time.Second)
		} else {
			m.playback.Position += 5 * time.Second
			if m.playback.Position > m.playback.Duration {
				m.playback.Position = m.playback.Duration
			}
		}
		return m, nil

	case key.Matches(msg, m.keyMap.SeekBackward):
		if m.audioPlayer != nil {
			m.audioPlayer.SeekRelative(-5 * time.Second)
		} else {
			m.playback.Position -= 5 * time.Second
			if m.playback.Position < 0 {
				m.playback.Position = 0
			}
		}
		return m, nil

	case key.Matches(msg, m.keyMap.VolumeUp):
		m.volume += 0.1
		if m.volume > 2.0 {
			m.volume = 2.0
		}
		if m.audioPlayer != nil {
			m.audioPlayer.SetVolume(m.volume)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.VolumeDown):
		m.volume -= 0.1
		if m.volume < 0.0 {
			m.volume = 0.0
		}
		if m.audioPlayer != nil {
			m.audioPlayer.SetVolume(m.volume)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.TabFocus):
		// Cycle focus between panels
		if m.focus == FocusBrowser {
			m.focus = FocusPlaylist
			if m.useLibrary {
				m.libBrowser.Blur()
			} else {
				m.browser.Blur()
			}
			m.playlist.Focus()
		} else {
			m.focus = FocusBrowser
			m.playlist.Blur()
			if m.useLibrary {
				m.libBrowser.Focus()
			} else {
				m.browser.Focus()
			}
		}
		return m, nil
	}

	// Panel-specific key handling
	switch m.focus {
	case FocusBrowser:
		// Forward navigation keys to appropriate browser
		if m.useLibrary {
			var cmd tea.Cmd
			m.libBrowser, cmd = m.libBrowser.Update(msg)
			if cmd != nil {
				return m, cmd
			}
		} else {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			if cmd != nil {
				return m, cmd
			}
		}
	case FocusPlaylist:
		// Check for playlist-specific actions first
		playlistKeyMap := m.playlist.KeyMap()
		switch {
		case key.Matches(msg, playlistKeyMap.Select):
			// Play the selected track
			if m.trackLoading {
				return m, nil
			}
			idx := m.playlist.SelectedIndex()
			if m.audioPlayer != nil {
				// Use startPlayingTrack for atomic state transition
				cmd := m.startPlayingTrack(idx)
				if cmd != nil {
					return m, cmd
				}
			} else {
				// Mock mode - set state directly
				if track := m.playlist.GetTrack(idx); track != nil {
					m.playlist.SetCurrentTrack(idx)
					m.currentTrack = track
					m.playback.State = StatePlaying
					m.playback.Position = 0
					m.playback.Duration = track.Duration
					m.playback.CurrentLoop = 0
				}
			}
			return m, nil
		case key.Matches(msg, playlistKeyMap.Remove):
			// Check if we're removing the currently playing track
			selectedIdx := m.playlist.SelectedIndex()
			currentIdx := m.playlist.CurrentIndex()
			wasPlayingRemoved := selectedIdx == currentIdx && currentIdx >= 0

			m.playlist.RemoveSelected()

			// If we removed the currently playing track, stop playback
			if wasPlayingRemoved {
				m.stopPlayback()
			}
			return m, nil
		case key.Matches(msg, playlistKeyMap.Clear):
			// Stop playback before clearing since we're removing all tracks
			m.stopPlayback()
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
		// Prevent duplicate commands while loading
		if m.trackLoading {
			return m, nil
		}

		// Use player's actual state instead of stale UI tick data
		playerState := m.audioPlayer.State()

		switch playerState {
		case player.StateStopped:
			// Try to play current track or first track in playlist
			currentIdx := m.playlist.CurrentIndex()
			if currentIdx >= 0 {
				// Resume from current position in playlist
				cmd := m.startPlayingTrack(currentIdx)
				if cmd != nil {
					return m, cmd
				}
			} else {
				// Start from the first track
				cmd := m.startPlayingTrack(0)
				if cmd != nil {
					return m, cmd
				}
			}
		case player.StatePlaying:
			m.audioPlayer.Pause()
			// Update UI state immediately to avoid stale state issues
			m.playback.State = StatePaused
		case player.StatePaused:
			if err := m.audioPlayer.Play(); err == nil {
				// Update UI state immediately
				m.playback.State = StatePlaying
			}
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

// startPlayingTrack initiates playback of a track at the given playlist index.
// It sets up pending state and returns a command to load and play the track.
// The pending state will be confirmed or cancelled by playTrackResult handler.
// Returns nil if the track cannot be found or no audio player is available.
func (m *Model) startPlayingTrack(playlistIndex int) tea.Cmd {
	if m.audioPlayer == nil {
		return nil
	}

	track := m.playlist.GetTrack(playlistIndex)
	if track == nil {
		return nil
	}

	// Set pending state (will be confirmed on success)
	m.pendingPlayIndex = playlistIndex
	m.pendingTrack = track
	m.trackLoading = true

	return playTrack(m.audioPlayer, track.Path)
}

// confirmTrackStarted commits the pending playback state after successful load.
// Call this when playTrackResult indicates success.
func (m *Model) confirmTrackStarted() {
	if m.pendingPlayIndex >= 0 && m.pendingTrack != nil {
		m.playlist.SetCurrentTrack(m.pendingPlayIndex)
		m.currentTrack = m.pendingTrack
	}
	m.trackLoading = false
	m.pendingPlayIndex = -1
	m.pendingTrack = nil
}

// cancelPendingTrack discards the pending playback state after a failed load.
// The previous track remains "current" in the UI.
func (m *Model) cancelPendingTrack() {
	m.trackLoading = false
	m.pendingPlayIndex = -1
	m.pendingTrack = nil
}

// stopPlayback stops the audio player and clears the current track state.
func (m *Model) stopPlayback() {
	if m.audioPlayer != nil {
		m.audioPlayer.Stop()
	}
	m.playlist.ClearCurrent()
	m.currentTrack = nil
	m.playback.State = StateStopped
	m.playback.Position = 0
	m.playback.CurrentLoop = 0
}

// loadTrackMetadata returns a command that loads track metadata without
// affecting the current playback state. It uses a separate player instance
// to read metadata, so it can be called while music is playing.
func loadTrackMetadata(path string) tea.Cmd {
	return func() tea.Msg {
		// Read metadata using a temporary player instance
		track, err := player.ReadTrackMetadata(path)
		if err != nil {
			return ErrorMsg{Err: err}
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

// loadTrackMetadataForPlay returns a command that loads track metadata and
// signals that the track should be played immediately after adding.
func loadTrackMetadataForPlay(path string) tea.Cmd {
	return func() tea.Msg {
		// Read metadata using a temporary player instance
		track, err := player.ReadTrackMetadata(path)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Convert player.Track to components.Track
		return TrackMetadataForPlayMsg{
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
// After successful play, it returns chip info for the track.
// Always sends TrackLoadCompleteMsg to clear the loading flag.
func playTrack(ap *player.AudioPlayer, path string) tea.Cmd {
	return func() tea.Msg {
		if err := ap.Load(path); err != nil {
			// Return sequence: load complete first, then error
			return playTrackResult{err: err}
		}
		if err := ap.Play(); err != nil {
			return playTrackResult{err: err}
		}
		// Return chip info after track is loaded and playing
		if track := ap.Track(); track != nil {
			return playTrackResult{chips: track.Chips}
		}
		return playTrackResult{}
	}
}

// playTrackResult bundles the result of a playTrack command.
type playTrackResult struct {
	err   error
	chips []player.ChipInfo
}

// defaultString returns s if non-empty, otherwise returns def.
func defaultString(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// loadLibTrackMetadata returns a command that loads track metadata from a library track.
func loadLibTrackMetadata(t library.Track) tea.Cmd {
	return func() tea.Msg {
		// Read full metadata using a temporary player instance
		track, err := player.ReadTrackMetadata(t.Path)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		return TrackMetadataLoadedMsg{
			Track: Track{
				Path:     track.Path,
				Title:    defaultString(track.Title, t.Title),
				Game:     defaultString(track.Game, t.Game),
				System:   defaultString(track.System, t.System),
				Composer: defaultString(track.Composer, t.Composer),
				Duration: track.Duration,
			},
			Chips: track.Chips,
		}
	}
}
