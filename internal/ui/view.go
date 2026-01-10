package ui

import (
	"fmt"
	"strings"
	"time"

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

	// Layout: footer takes 1 line at absolute bottom, main content fills the rest
	footerHeight := 1
	mainHeight := m.height - footerHeight

	// Calculate panel widths
	libraryWidth := m.width * libraryWidthPercent / 100
	rightWidth := m.width - libraryWidth

	// Build the main layout - both panels take full mainHeight
	leftPanel := m.renderLibrary(libraryWidth, mainHeight)
	rightPanel := m.renderRightPane(rightWidth, mainHeight)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Ensure main content takes exactly mainHeight lines
	mainContent = lipgloss.NewStyle().
		Width(m.width).
		Height(mainHeight).
		MaxHeight(mainHeight).
		Render(mainContent)

	// Footer pinned to bottom
	footer := m.renderFooter()
	footer = lipgloss.NewStyle().
		Width(m.width).
		Height(footerHeight).
		Render(footer)

	mainView := lipgloss.JoinVertical(lipgloss.Left, mainContent, footer)

	// Render help overlay if visible
	if m.helpPopup.Visible() {
		return m.renderHelpOverlay(mainView)
	}

	return mainView
}

// renderHelpOverlay renders the help popup on top of the main view.
func (m Model) renderHelpOverlay(mainView string) string {
	// Get the popup content
	popup := m.helpPopup.View()

	// Calculate popup dimensions
	popupLines := strings.Split(popup, "\n")
	popupHeight := len(popupLines)
	popupWidth := 0
	for _, line := range popupLines {
		if w := lipgloss.Width(line); w > popupWidth {
			popupWidth = w
		}
	}

	// Calculate position to center the popup
	mainLines := strings.Split(mainView, "\n")
	mainHeight := len(mainLines)

	startY := (mainHeight - popupHeight) / 2
	if startY < 0 {
		startY = 0
	}
	startX := (m.width - popupWidth) / 2
	if startX < 0 {
		startX = 0
	}

	// Create a new view with the popup overlaid
	result := make([]string, mainHeight)
	for i, line := range mainLines {
		// Ensure line is wide enough
		lineWidth := lipgloss.Width(line)
		if lineWidth < m.width {
			line = line + strings.Repeat(" ", m.width-lineWidth)
		}

		// Check if this line overlaps with the popup
		popupLineIdx := i - startY
		if popupLineIdx >= 0 && popupLineIdx < len(popupLines) {
			popupLine := popupLines[popupLineIdx]
			popupLineWidth := lipgloss.Width(popupLine)

			// Build the overlaid line
			// Left part (before popup)
			var newLine strings.Builder
			if startX > 0 {
				// Get characters before popup
				newLine.WriteString(truncateToWidth(line, startX))
			}
			// Popup content
			newLine.WriteString(popupLine)
			// Right part (after popup)
			rightStart := startX + popupLineWidth
			if rightStart < m.width {
				remaining := substringFromWidth(line, rightStart)
				newLine.WriteString(remaining)
			}
			result[i] = newLine.String()
		} else {
			result[i] = line
		}
	}

	return strings.Join(result, "\n")
}

// truncateToWidth truncates a string to fit within a given visual width.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	currentWidth := 0
	var result strings.Builder
	for _, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth > width {
			// Pad with spaces if needed
			for currentWidth < width {
				result.WriteRune(' ')
				currentWidth++
			}
			break
		}
		result.WriteRune(r)
		currentWidth += runeWidth
	}
	// Pad if string was too short
	for currentWidth < width {
		result.WriteRune(' ')
		currentWidth++
	}
	return result.String()
}

// substringFromWidth returns the portion of a string starting from a given visual width.
func substringFromWidth(s string, startWidth int) string {
	currentWidth := 0
	for i, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth >= startWidth {
			return s[i:]
		}
		currentWidth += runeWidth
	}
	return ""
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

	// Render the browser component
	content := m.browser.View()

	// Calculate content height (panel height minus borders and title)
	// Border takes 2 lines (top + bottom), title takes 1 line
	contentHeight := height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Constrain content to fit within the panel
	content = constrainContentHeight(content, contentHeight)

	// Pass full outer dimensions - RenderPanel handles inner calculation
	return m.styles.RenderPanel("Library", content, focused, width, height)
}

// constrainContentHeight truncates content to fit within the specified height.
func constrainContentHeight(content string, maxHeight int) string {
	if maxHeight <= 0 {
		return content
	}

	// Remove trailing newline to avoid off-by-one in line counting
	content = strings.TrimSuffix(content, "\n")

	lines := strings.Split(content, "\n")
	if len(lines) > maxHeight {
		lines = lines[:maxHeight]
	}

	return strings.Join(lines, "\n")
}

// renderRightPane renders the right side containing playlist, track info, and progress.
func (m Model) renderRightPane(width, height int) string {
	// Fixed heights for bottom panels (like termusic's Constraint::Length)
	progressHeight := 5  // Status line + progress bar + border(2) + title(1)
	trackInfoHeight := 6 // Track info with border

	// Playlist takes remaining space (like termusic's Constraint::Min)
	playlistHeight := height - progressHeight - trackInfoHeight
	if playlistHeight < 3 {
		playlistHeight = 3
	}

	playlist := m.renderPlaylist(width, playlistHeight)
	trackInfo := m.renderTrackInfo(width, trackInfoHeight)
	progress := m.renderProgress(width, progressHeight)

	return lipgloss.JoinVertical(lipgloss.Left, playlist, trackInfo, progress)
}

// renderPlaylist renders the playlist panel.
func (m Model) renderPlaylist(width, height int) string {
	focused := m.focus == FocusPlaylist

	// Use the playlist component's view
	content := m.playlist.View()

	// Use the playlist's title which includes track count info
	title := m.playlist.Title()
	// Pass full outer dimensions - RenderPanel handles inner calculation
	return m.styles.RenderPanel(title, content, focused, width, height)
}

// renderTrackInfo renders the track information panel.
func (m Model) renderTrackInfo(width, height int) string {
	content := strings.Builder{}

	if m.currentTrack != nil {
		// Title
		title := m.currentTrack.Title
		if title == "" {
			title = "(Unknown)"
		}
		content.WriteString(fmt.Sprintf("%s %s\n",
			m.styles.TextMuted.Render("Track:"),
			m.styles.TextBold.Render(title)))

		// Game
		game := m.currentTrack.Game
		if game == "" {
			game = "(Unknown)"
		}
		content.WriteString(fmt.Sprintf("%s %s\n",
			m.styles.TextMuted.Render("Game:"),
			m.styles.Text.Render(game)))

		// System and Chips on same line
		system := m.currentTrack.System
		if system == "" {
			system = "(Unknown)"
		}

		// Build chip list from trackChips if available
		chipList := m.formatChipList()
		content.WriteString(fmt.Sprintf("%s %s  %s %s\n",
			m.styles.TextMuted.Render("System:"),
			m.styles.Text.Render(system),
			m.styles.TextMuted.Render("Chips:"),
			m.styles.Text.Render(chipList)))

		// Composer
		composer := m.currentTrack.Composer
		if composer == "" {
			composer = "(Unknown)"
		}
		content.WriteString(fmt.Sprintf("%s %s",
			m.styles.TextMuted.Render("Composer:"),
			m.styles.Text.Render(composer)))
	} else {
		content.WriteString(m.styles.TextMuted.Render("No track loaded"))
		content.WriteString("\n")
		content.WriteString(m.styles.TextMuted.Render("Select a VGM file from the library"))
	}

	// Pass full outer dimensions - RenderPanel handles inner calculation
	return m.styles.RenderPanel("Track Info", content.String(), false, width, height)
}

// formatChipList formats the chip info into a readable string.
func (m Model) formatChipList() string {
	if len(m.trackChips) == 0 {
		return "(none)"
	}

	chips := make([]string, 0, len(m.trackChips))
	for _, chip := range m.trackChips {
		chips = append(chips, chip.Name)
	}
	return strings.Join(chips, ", ")
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

	// Second line: progress bar (adjust width for border and padding)
	m.progress.SetWidth(width - 6)
	m.progress.SetElapsed(m.playback.Position)
	m.progress.SetDuration(m.playback.Duration)
	content.WriteString(m.progress.View())

	// Use RenderPanel to get complete borders like other panels
	// Pass full outer dimensions - RenderPanel handles inner calculation
	return m.styles.RenderPanel("Progress", content.String(), false, width, height)
}

// renderFooter renders the help/key hints footer.
func (m Model) renderFooter() string {
	var content strings.Builder

	// Show error if recent (within 5 seconds)
	if m.lastError != "" && time.Since(m.errorTime) < 5*time.Second {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)
		content.WriteString(errorStyle.Render("Error: " + m.lastError))
		content.WriteString("  ")
	}

	// Show contextual help based on focus
	helpStyle := lipgloss.NewStyle().Foreground(ColorTextMuted)
	keyStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	switch m.focus {
	case FocusBrowser:
		content.WriteString(keyStyle.Render("Enter"))
		content.WriteString(helpStyle.Render(":add "))
		content.WriteString(keyStyle.Render("."))
		content.WriteString(helpStyle.Render(":hidden "))
		content.WriteString(keyStyle.Render("Tab"))
		content.WriteString(helpStyle.Render(":playlist "))
	case FocusPlaylist:
		content.WriteString(keyStyle.Render("Enter"))
		content.WriteString(helpStyle.Render(":play "))
		content.WriteString(keyStyle.Render("d"))
		content.WriteString(helpStyle.Render(":remove "))
		content.WriteString(keyStyle.Render("Tab"))
		content.WriteString(helpStyle.Render(":browser "))
	}

	// Common hints
	content.WriteString(keyStyle.Render("Space"))
	content.WriteString(helpStyle.Render(":play/pause "))
	content.WriteString(keyStyle.Render("?"))
	content.WriteString(helpStyle.Render(":help "))
	content.WriteString(keyStyle.Render("q"))
	content.WriteString(helpStyle.Render(":quit"))

	return content.String()
}
