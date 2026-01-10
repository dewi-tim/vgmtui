package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	// Minimum dimensions
	minWidth  = 60
	minHeight = 15

	// Panel proportions
	libraryWidthPercent = 30
)

// View renders the entire UI.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Handle small terminal
	if m.width < minWidth || m.height < minHeight {
		return m.renderTooSmall()
	}

	// Calculate layout dimensions
	libraryWidth := m.width * libraryWidthPercent / 100
	rightWidth := m.width - libraryWidth - 3 // 3 for spacing/borders

	// Build the main layout
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderLibrary(libraryWidth, m.height-4),
		" ",
		m.renderRightPane(rightWidth, m.height-4),
	)

	// Add footer
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, footer)
}

// renderTooSmall renders a message when the terminal is too small.
func (m Model) renderTooSmall() string {
	msg := fmt.Sprintf("Terminal too small\nNeed at least %dx%d\nCurrent: %dx%d",
		minWidth, minHeight, m.width, m.height)
	return lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Render(msg)
}

// renderLibrary renders the left library panel.
func (m Model) renderLibrary(width, height int) string {
	focused := m.focus == FocusBrowser

	// Build placeholder content
	content := strings.Builder{}
	content.WriteString(m.styles.TextMuted.Render("(file browser placeholder)"))
	content.WriteString("\n\n")
	content.WriteString(m.styles.Text.Render("  .. (parent)"))
	content.WriteString("\n")
	content.WriteString(m.styles.Text.Render("  [Genesis]"))
	content.WriteString("\n")
	content.WriteString(m.styles.Text.Render("  [SNES]"))
	content.WriteString("\n")
	content.WriteString(m.styles.TextHighlight.Render("> [PC-Engine]"))
	content.WriteString("\n")
	content.WriteString(m.styles.Text.Render("    song.vgm"))
	content.WriteString("\n")
	content.WriteString(m.styles.Text.Render("    track.vgz"))

	return m.styles.RenderPanel("Library", content.String(), focused, width-2, height-2)
}

// renderRightPane renders the right side containing playlist, track info, and progress.
func (m Model) renderRightPane(width, height int) string {
	// Calculate heights for sub-panels
	playlistHeight := height * 50 / 100
	trackInfoHeight := height * 25 / 100
	progressHeight := height - playlistHeight - trackInfoHeight - 2

	playlist := m.renderPlaylist(width, playlistHeight)
	trackInfo := m.renderTrackInfo(width, trackInfoHeight)
	progress := m.renderProgress(width, progressHeight)

	return lipgloss.JoinVertical(lipgloss.Left, playlist, trackInfo, progress)
}

// renderPlaylist renders the playlist panel.
func (m Model) renderPlaylist(width, height int) string {
	focused := m.focus == FocusPlaylist

	// Build placeholder content with mock playlist
	content := strings.Builder{}
	content.WriteString(m.styles.TextMuted.Render("(playlist placeholder)"))
	content.WriteString("\n\n")
	content.WriteString(m.styles.TextMuted.Render(" Duration | Title         | Game"))
	content.WriteString("\n")
	content.WriteString(m.styles.TextMuted.Render(" ---------+---------------+--------"))
	content.WriteString("\n")
	content.WriteString(m.styles.Text.Render("   02:34  | Green Hill    | Sonic 1"))
	content.WriteString("\n")
	content.WriteString(m.styles.TextHighlight.Render(" > 03:01  | Marble Zone   | Sonic 1"))
	content.WriteString("\n")
	content.WriteString(m.styles.Text.Render("   01:45  | Star Light    | Sonic 1"))

	title := fmt.Sprintf("Playlist [2/3]")
	return m.styles.RenderPanel(title, content.String(), focused, width-2, height-2)
}

// renderTrackInfo renders the track information panel.
func (m Model) renderTrackInfo(width, height int) string {
	content := strings.Builder{}

	if m.currentTrack != nil {
		content.WriteString(fmt.Sprintf("%s %s\n",
			m.styles.TextMuted.Render("Track:"),
			m.styles.TextBold.Render(m.currentTrack.Title)))
		content.WriteString(fmt.Sprintf("%s %s\n",
			m.styles.TextMuted.Render("Game:"),
			m.styles.Text.Render(m.currentTrack.Game)))
		content.WriteString(fmt.Sprintf("%s %s  %s %s\n",
			m.styles.TextMuted.Render("System:"),
			m.styles.Text.Render(m.currentTrack.System),
			m.styles.TextMuted.Render("Chips:"),
			m.styles.Text.Render("YM2612, SN76496")))
		content.WriteString(fmt.Sprintf("%s %s",
			m.styles.TextMuted.Render("Composer:"),
			m.styles.Text.Render(m.currentTrack.Composer)))
	} else {
		content.WriteString(m.styles.TextMuted.Render("No track loaded"))
	}

	// No border for track info, just content
	style := lipgloss.NewStyle().
		Width(width - 4).
		Height(height - 1).
		Padding(0, 1)

	return style.Render(content.String())
}

// renderProgress renders the progress bar and playback status.
func (m Model) renderProgress(width, height int) string {
	content := strings.Builder{}

	// Status indicator
	var statusStyle lipgloss.Style
	var statusText string
	var statusIcon string

	switch m.playback.State {
	case StatePlaying:
		statusStyle = m.styles.StatusPlaying
		statusText = "Playing"
		statusIcon = ">"
	case StatePaused:
		statusStyle = m.styles.StatusPaused
		statusText = "Paused"
		statusIcon = "||"
	case StateStopped:
		statusStyle = m.styles.StatusStopped
		statusText = "Stopped"
		statusIcon = "[]"
	}

	// First line: status and loop info
	loopInfo := ""
	if m.playback.TotalLoops > 0 {
		loopInfo = fmt.Sprintf(" | Loop %d/%d", m.playback.CurrentLoop+1, m.playback.TotalLoops)
	}

	content.WriteString(fmt.Sprintf("%s %s%s\n",
		statusStyle.Render(statusIcon),
		statusStyle.Render(statusText),
		m.styles.TextMuted.Render(loopInfo)))

	// Second line: progress bar
	m.progress.SetWidth(width - 6)
	m.progress.SetElapsed(m.playback.Position)
	m.progress.SetDuration(m.playback.Duration)
	content.WriteString(m.progress.View())

	// Style the container
	style := lipgloss.NewStyle().
		Width(width - 4).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted)

	return style.Render(content.String())
}

// renderFooter renders the help/key hints footer.
func (m Model) renderFooter() string {
	if m.showHelp {
		return m.help.View(m.keyMap)
	}

	// Short help
	return m.help.View(m.keyMap)
}
