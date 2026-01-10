// Package ui provides the Bubbletea TUI for vgmtui.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colors used throughout the UI.
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7571F9")
	ColorSecondary = lipgloss.Color("#EE6FF8")
	ColorMuted     = lipgloss.Color("#606060")
	ColorSubtle    = lipgloss.Color("#383838")

	// State colors
	ColorPlaying = lipgloss.Color("#04B575")
	ColorPaused  = lipgloss.Color("#FFA500")
	ColorStopped = lipgloss.Color("#FF5555")

	// Text colors
	ColorText      = lipgloss.Color("#FAFAFA")
	ColorTextMuted = lipgloss.Color("#A0A0A0")
)

// Styles contains all the styles used in the UI.
type Styles struct {
	// Panel styles
	FocusedBorder lipgloss.Style
	NormalBorder  lipgloss.Style

	// Title styles
	Title      lipgloss.Style
	TitleMuted lipgloss.Style

	// Text styles
	Text         lipgloss.Style
	TextMuted    lipgloss.Style
	TextBold     lipgloss.Style
	TextHighlight lipgloss.Style

	// Status styles
	StatusPlaying lipgloss.Style
	StatusPaused  lipgloss.Style
	StatusStopped lipgloss.Style

	// Progress bar styles
	ProgressFilled lipgloss.Style
	ProgressEmpty  lipgloss.Style
	ProgressTime   lipgloss.Style

	// Footer/help styles
	FooterKey  lipgloss.Style
	FooterDesc lipgloss.Style
	FooterSep  lipgloss.Style
}

// DefaultStyles returns the default styles for the UI.
func DefaultStyles() Styles {
	return Styles{
		// Panel borders
		FocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary),

		NormalBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted),

		// Titles
		Title: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		TitleMuted: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		// Text
		Text: lipgloss.NewStyle().
			Foreground(ColorText),

		TextMuted: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		TextBold: lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true),

		TextHighlight: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		// Status indicators
		StatusPlaying: lipgloss.NewStyle().
			Foreground(ColorPlaying).
			Bold(true),

		StatusPaused: lipgloss.NewStyle().
			Foreground(ColorPaused).
			Bold(true),

		StatusStopped: lipgloss.NewStyle().
			Foreground(ColorStopped).
			Bold(true),

		// Progress bar
		ProgressFilled: lipgloss.NewStyle().
			Foreground(ColorPrimary),

		ProgressEmpty: lipgloss.NewStyle().
			Foreground(ColorMuted),

		ProgressTime: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		// Footer
		FooterKey: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		FooterDesc: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		FooterSep: lipgloss.NewStyle().
			Foreground(ColorSubtle),
	}
}

// PanelStyle returns a bordered panel style with the given dimensions.
// width and height are the TOTAL outer dimensions including border.
func (s Styles) PanelStyle(focused bool, width, height int) lipgloss.Style {
	borderColor := ColorMuted
	if focused {
		borderColor = ColorPrimary
	}
	// Inner dimensions after accounting for border (1 char each side)
	innerWidth := width - 2
	innerHeight := height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true, true, true, true). // top, right, bottom, left
		BorderForeground(borderColor).
		Width(innerWidth).
		Height(innerHeight)
}

// RenderPanel renders content in a panel with a title.
// width and height are the TOTAL outer dimensions including border.
func (s Styles) RenderPanel(title, content string, focused bool, width, height int) string {
	titleStyle := s.TitleMuted
	if focused {
		titleStyle = s.Title
	}

	// Render title
	renderedTitle := titleStyle.Render(title)

	// Inner height after border (2 lines) and title (1 line)
	contentMaxHeight := height - 2 - 1
	if contentMaxHeight < 1 {
		contentMaxHeight = 1
	}

	// Truncate content if it exceeds the available height
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentMaxHeight {
		contentLines = contentLines[:contentMaxHeight]
	}
	// Pad content to fill available space
	for len(contentLines) < contentMaxHeight {
		contentLines = append(contentLines, "")
	}
	content = strings.Join(contentLines, "\n")

	// Build panel content with title at top
	panelContent := lipgloss.JoinVertical(lipgloss.Left, renderedTitle, content)

	// Apply border - PanelStyle now expects outer dimensions
	return s.PanelStyle(focused, width, height).Render(panelContent)
}
