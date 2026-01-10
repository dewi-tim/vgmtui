// Package components provides UI components for vgmtui.
package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpPopup is a full-screen help overlay that displays all keybindings.
type HelpPopup struct {
	viewport viewport.Model
	visible  bool
	width    int
	height   int

	// Styles
	borderStyle   lipgloss.Style
	titleStyle    lipgloss.Style
	categoryStyle lipgloss.Style
	keyStyle      lipgloss.Style
	descStyle     lipgloss.Style
	footerStyle   lipgloss.Style
}

// HelpKeyMap defines key bindings for the help popup.
type HelpKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Close    key.Binding
}

// DefaultHelpKeyMap returns the default help popup key bindings.
func DefaultHelpKeyMap() HelpKeyMap {
	return HelpKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		Close: key.NewBinding(
			key.WithKeys("?", "esc", "enter", "q"),
			key.WithHelp("?/esc/enter", "close"),
		),
	}
}

// NewHelpPopup creates a new help popup.
func NewHelpPopup() HelpPopup {
	vp := viewport.New(50, 20)
	vp.MouseWheelEnabled = true

	return HelpPopup{
		viewport: vp,
		visible:  false,
		width:    60,
		height:   24,
		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7571F9")),
		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true),
		categoryStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true),
		keyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true),
		descStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")),
		footerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")).
			Italic(true),
	}
}

// Update handles messages for the help popup.
func (h HelpPopup) Update(msg tea.Msg) (HelpPopup, tea.Cmd) {
	if !h.visible {
		return h, nil
	}

	keyMap := DefaultHelpKeyMap()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keyMap.Close):
			h.visible = false
			return h, nil
		case key.Matches(msg, keyMap.Up):
			h.viewport.ScrollUp(1)
		case key.Matches(msg, keyMap.Down):
			h.viewport.ScrollDown(1)
		case key.Matches(msg, keyMap.PageUp):
			h.viewport.PageUp()
		case key.Matches(msg, keyMap.PageDown):
			h.viewport.PageDown()
		}
	}

	var cmd tea.Cmd
	h.viewport, cmd = h.viewport.Update(msg)
	return h, cmd
}

// View renders the help popup as an overlay.
func (h HelpPopup) View() string {
	if !h.visible {
		return ""
	}

	content := h.buildHelpContent()
	h.viewport.SetContent(content)

	// Calculate popup dimensions
	popupWidth := h.width
	if popupWidth > h.width-4 {
		popupWidth = h.width - 4
	}
	if popupWidth < 40 {
		popupWidth = 40
	}

	popupHeight := h.height
	if popupHeight > h.height-4 {
		popupHeight = h.height - 4
	}
	if popupHeight < 15 {
		popupHeight = 15
	}

	// Update viewport size
	h.viewport.Width = popupWidth - 4
	h.viewport.Height = popupHeight - 4

	// Build the popup
	title := h.titleStyle.Render(" Help ")
	footer := h.footerStyle.Render("Press ? or Esc to close")

	viewportContent := h.viewport.View()

	// Create the box content
	boxContent := lipgloss.JoinVertical(lipgloss.Left,
		viewportContent,
		"",
		lipgloss.NewStyle().Width(popupWidth-4).Align(lipgloss.Center).Render(footer),
	)

	// Apply border with title
	box := h.borderStyle.
		Width(popupWidth).
		Height(popupHeight).
		Render(boxContent)

	// Add title to top border
	lines := strings.Split(box, "\n")
	if len(lines) > 0 {
		// Insert title into top border line
		borderLine := lines[0]
		titlePos := (lipgloss.Width(borderLine) - lipgloss.Width(title)) / 2
		if titlePos > 2 {
			runes := []rune(borderLine)
			titleRunes := []rune(title)
			for i, r := range titleRunes {
				if titlePos+i < len(runes) {
					runes[titlePos+i] = r
				}
			}
			lines[0] = string(runes)
		}
		box = strings.Join(lines, "\n")
	}

	return box
}

// buildHelpContent creates the help text content.
func (h HelpPopup) buildHelpContent() string {
	var b strings.Builder

	// Helper to add a keybinding line
	addKey := func(key, desc string) {
		keyPadded := lipgloss.NewStyle().Width(12).Render(h.keyStyle.Render(key))
		b.WriteString(keyPadded)
		b.WriteString(h.descStyle.Render(desc))
		b.WriteString("\n")
	}

	// Helper to add a category header
	addCategory := func(name string) {
		b.WriteString("\n")
		b.WriteString(h.categoryStyle.Render(name))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("-", 35))
		b.WriteString("\n")
	}

	// Global
	addCategory("Global")
	addKey("?", "Toggle this help")
	addKey("q", "Quit application")
	addKey("Tab", "Switch panel focus")

	// Playback
	addCategory("Playback")
	addKey("Space", "Play/Pause")
	addKey("n", "Next track")
	addKey("N", "Previous track")
	addKey("s", "Stop playback")
	addKey("f", "Seek forward 5s")
	addKey("b", "Seek backward 5s")
	addKey("+/=", "Volume up")
	addKey("-", "Volume down")

	// Browser
	addCategory("Browser")
	addKey("j/k", "Navigate up/down")
	addKey("g/G", "Go to top/bottom")
	addKey("PgUp/Dn", "Page up/down")
	addKey("Enter/l", "Open directory/select file")
	addKey("Backspace/h", "Go to parent directory")
	addKey(".", "Toggle hidden files")

	// Playlist
	addCategory("Playlist")
	addKey("j/k", "Navigate up/down")
	addKey("g/G", "Go to top/bottom")
	addKey("PgUp/Dn", "Page up/down")
	addKey("Enter/l", "Play selected track")
	addKey("d", "Remove selected track")
	addKey("D", "Clear playlist")

	return b.String()
}

// SetSize sets the available size for the help popup.
func (h *HelpPopup) SetSize(width, height int) {
	h.width = width
	h.height = height

	// Calculate content area
	contentWidth := width * 70 / 100
	if contentWidth < 40 {
		contentWidth = 40
	}
	if contentWidth > 60 {
		contentWidth = 60
	}

	contentHeight := height * 80 / 100
	if contentHeight < 15 {
		contentHeight = 15
	}
	if contentHeight > 30 {
		contentHeight = 30
	}

	h.viewport.Width = contentWidth - 4
	h.viewport.Height = contentHeight - 4
}

// Show makes the help popup visible.
func (h *HelpPopup) Show() {
	h.visible = true
	h.viewport.GotoTop()
}

// Hide makes the help popup invisible.
func (h *HelpPopup) Hide() {
	h.visible = false
}

// Visible returns whether the help popup is visible.
func (h HelpPopup) Visible() bool {
	return h.visible
}

// Toggle toggles the visibility of the help popup.
func (h *HelpPopup) Toggle() {
	if h.visible {
		h.Hide()
	} else {
		h.Show()
	}
}
