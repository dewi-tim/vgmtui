// Package components provides UI components for vgmtui.
package components

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// VGM-compatible file extensions.
var vgmExtensions = []string{".vgm", ".vgz", ".s98", ".dro", ".gym"}

// BrowserKeyMap defines key bindings for the browser.
type BrowserKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	GoToTop      key.Binding
	GoToBottom   key.Binding
	Open         key.Binding
	Back         key.Binding
	ToggleHidden key.Binding
}

// DefaultBrowserKeyMap returns the default browser key bindings.
func DefaultBrowserKeyMap() BrowserKeyMap {
	return BrowserKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdown", "page down"),
		),
		GoToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		GoToBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter", "l", "right"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "h", "left"),
			key.WithHelp("backspace", "parent"),
		),
		ToggleHidden: key.NewBinding(
			key.WithKeys("."),
			key.WithHelp(".", "hidden"),
		),
	}
}

// FileEntry represents a file or directory in the browser.
type FileEntry struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

// Browser is a file browser component for navigating and selecting VGM files.
type Browser struct {
	// Current directory
	currentDir string

	// File entries in the current directory
	entries []FileEntry

	// Selection state
	selected int
	min      int // First visible index
	max      int // Last visible index

	// Dimensions
	width  int
	height int

	// State
	focused    bool
	showHidden bool
	err        error

	// Key bindings
	KeyMap BrowserKeyMap

	// Styles
	Styles BrowserStyles
}

// BrowserStyles contains styles for the browser component.
type BrowserStyles struct {
	Cursor       lipgloss.Style
	Directory    lipgloss.Style
	File         lipgloss.Style
	VGMFile      lipgloss.Style
	Selected     lipgloss.Style
	SelectedDir  lipgloss.Style
	Muted        lipgloss.Style
	EmptyDir     lipgloss.Style
}

// DefaultBrowserStyles returns the default browser styles.
func DefaultBrowserStyles() BrowserStyles {
	return BrowserStyles{
		Cursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true),
		Directory: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#99CCFF")),
		File: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")),
		VGMFile: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true),
		SelectedDir: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true),
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")),
		EmptyDir: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")).
			Italic(true),
	}
}

// FileSelectedMsg is sent when a file is selected.
type FileSelectedMsg struct {
	Path string
}

// DirChangedMsg is sent when the directory changes.
type DirChangedMsg struct {
	Path string
}

// BrowserReadDirMsg is sent when directory contents are read.
type BrowserReadDirMsg struct {
	Dir     string
	Entries []FileEntry
	Err     error
}

// NewBrowser creates a new browser starting at the given directory.
func NewBrowser(startDir string) Browser {
	if startDir == "" {
		startDir, _ = os.UserHomeDir()
		if startDir == "" {
			startDir = "/"
		}
	}

	// Ensure the path is absolute
	absDir, err := filepath.Abs(startDir)
	if err == nil {
		startDir = absDir
	}

	b := Browser{
		currentDir: startDir,
		entries:    []FileEntry{},
		selected:   0,
		min:        0,
		max:        10,
		width:      30,
		height:     10,
		focused:    false,
		showHidden: false,
		KeyMap:     DefaultBrowserKeyMap(),
		Styles:     DefaultBrowserStyles(),
	}

	return b
}

// Init initializes the browser and returns a command to read the directory.
func (b Browser) Init() tea.Cmd {
	return b.readDir(b.currentDir)
}

// readDir returns a command to read a directory's contents.
func (b Browser) readDir(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := readDirFiltered(path, b.showHidden)
		return BrowserReadDirMsg{
			Dir:     path,
			Entries: entries,
			Err:     err,
		}
	}
}

// readDirFiltered reads directory contents, filtering and sorting appropriately.
func readDirFiltered(path string, showHidden bool) ([]FileEntry, error) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var entries []FileEntry

	for _, de := range dirEntries {
		name := de.Name()

		// Skip hidden files unless showHidden is true
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		info, err := de.Info()
		if err != nil {
			continue
		}

		isDir := de.IsDir()

		// For files, only include VGM-compatible types
		if !isDir && !isVGMFile(name) {
			continue
		}

		entries = append(entries, FileEntry{
			Name:  name,
			Path:  filepath.Join(path, name),
			IsDir: isDir,
			Size:  info.Size(),
		})
	}

	// Sort: directories first, then alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries, nil
}

// isVGMFile checks if a filename has a VGM-compatible extension.
func isVGMFile(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range vgmExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// Update handles messages and updates the browser state.
func (b Browser) Update(msg tea.Msg) (Browser, tea.Cmd) {
	switch msg := msg.(type) {
	case BrowserReadDirMsg:
		if msg.Err != nil {
			b.err = msg.Err
			return b, nil
		}
		b.currentDir = msg.Dir
		b.entries = msg.Entries
		b.err = nil
		// Reset selection if needed
		if b.selected >= len(b.entries) {
			b.selected = len(b.entries) - 1
			if b.selected < 0 {
				b.selected = 0
			}
		}
		b.updateViewport()
		return b, nil

	case tea.KeyMsg:
		if !b.focused {
			return b, nil
		}
		return b.handleKeyMsg(msg)
	}

	return b, nil
}

// handleKeyMsg handles keyboard input when focused.
func (b Browser) handleKeyMsg(msg tea.KeyMsg) (Browser, tea.Cmd) {
	switch {
	case key.Matches(msg, b.KeyMap.Up):
		b.moveUp()
		return b, nil

	case key.Matches(msg, b.KeyMap.Down):
		b.moveDown()
		return b, nil

	case key.Matches(msg, b.KeyMap.PageUp):
		b.pageUp()
		return b, nil

	case key.Matches(msg, b.KeyMap.PageDown):
		b.pageDown()
		return b, nil

	case key.Matches(msg, b.KeyMap.GoToTop):
		b.goToTop()
		return b, nil

	case key.Matches(msg, b.KeyMap.GoToBottom):
		b.goToBottom()
		return b, nil

	case key.Matches(msg, b.KeyMap.Open):
		return b.openSelected()

	case key.Matches(msg, b.KeyMap.Back):
		return b.goToParent()

	case key.Matches(msg, b.KeyMap.ToggleHidden):
		b.showHidden = !b.showHidden
		return b, b.readDir(b.currentDir)
	}

	return b, nil
}

// moveUp moves selection up one item.
func (b *Browser) moveUp() {
	if b.selected > 0 {
		b.selected--
		if b.selected < b.min {
			b.min--
			b.max--
		}
	}
}

// moveDown moves selection down one item.
func (b *Browser) moveDown() {
	if b.selected < len(b.entries)-1 {
		b.selected++
		if b.selected > b.max {
			b.min++
			b.max++
		}
	}
}

// pageUp moves selection up one page.
func (b *Browser) pageUp() {
	visible := b.visibleCount()
	b.selected -= visible
	if b.selected < 0 {
		b.selected = 0
	}
	b.min -= visible
	if b.min < 0 {
		b.min = 0
	}
	b.max = b.min + visible - 1
	if b.max >= len(b.entries) {
		b.max = len(b.entries) - 1
	}
}

// pageDown moves selection down one page.
func (b *Browser) pageDown() {
	visible := b.visibleCount()
	b.selected += visible
	if b.selected >= len(b.entries) {
		b.selected = len(b.entries) - 1
	}
	b.max += visible
	if b.max >= len(b.entries) {
		b.max = len(b.entries) - 1
	}
	b.min = b.max - visible + 1
	if b.min < 0 {
		b.min = 0
	}
}

// goToTop moves selection to the first item.
func (b *Browser) goToTop() {
	b.selected = 0
	b.min = 0
	b.max = b.visibleCount() - 1
	if b.max >= len(b.entries) {
		b.max = len(b.entries) - 1
	}
}

// goToBottom moves selection to the last item.
func (b *Browser) goToBottom() {
	b.selected = len(b.entries) - 1
	if b.selected < 0 {
		b.selected = 0
	}
	b.max = len(b.entries) - 1
	b.min = b.max - b.visibleCount() + 1
	if b.min < 0 {
		b.min = 0
	}
}

// openSelected opens the selected entry (file or directory).
func (b Browser) openSelected() (Browser, tea.Cmd) {
	if len(b.entries) == 0 {
		return b, nil
	}

	entry := b.entries[b.selected]

	if entry.IsDir {
		// Enter directory
		b.selected = 0
		b.min = 0
		b.max = b.visibleCount() - 1
		return b, tea.Batch(
			b.readDir(entry.Path),
			func() tea.Msg { return DirChangedMsg{Path: entry.Path} },
		)
	}

	// File selected - emit FileSelectedMsg
	return b, func() tea.Msg {
		return FileSelectedMsg{Path: entry.Path}
	}
}

// goToParent navigates to the parent directory.
func (b Browser) goToParent() (Browser, tea.Cmd) {
	parent := filepath.Dir(b.currentDir)
	if parent == b.currentDir {
		// Already at root
		return b, nil
	}

	// Try to find current dir name to restore selection
	currentName := filepath.Base(b.currentDir)
	b.selected = 0
	b.min = 0
	b.max = b.visibleCount() - 1

	return b, tea.Batch(
		b.readDir(parent),
		func() tea.Msg { return DirChangedMsg{Path: parent} },
		// After reading, try to select the directory we came from
		func() tea.Msg { return BrowserSelectNameMsg{Name: currentName} },
	)
}

// BrowserSelectNameMsg is sent to select a specific entry by name after navigating up.
type BrowserSelectNameMsg struct {
	Name string
}

// HandleSelectName handles selecting an entry by name (used after navigating up).
func (b *Browser) HandleSelectName(name string) {
	for i, entry := range b.entries {
		if entry.Name == name {
			b.selected = i
			b.updateViewport()
			return
		}
	}
}

// visibleCount returns the number of visible items.
func (b Browser) visibleCount() int {
	count := b.height - 2 // Account for header line and padding
	if count < 1 {
		count = 1
	}
	return count
}

// updateViewport ensures the selected item is visible.
func (b *Browser) updateViewport() {
	visible := b.visibleCount()
	if visible <= 0 {
		visible = 1
	}

	// Ensure max doesn't exceed entries
	if b.max >= len(b.entries) {
		b.max = len(b.entries) - 1
	}
	if b.max < 0 {
		b.max = 0
	}

	// Ensure min is valid
	b.min = b.max - visible + 1
	if b.min < 0 {
		b.min = 0
		b.max = b.min + visible - 1
		if b.max >= len(b.entries) {
			b.max = len(b.entries) - 1
		}
	}

	// Adjust if selected is outside viewport
	if b.selected < b.min {
		b.min = b.selected
		b.max = b.min + visible - 1
		if b.max >= len(b.entries) {
			b.max = len(b.entries) - 1
		}
	} else if b.selected > b.max {
		b.max = b.selected
		b.min = b.max - visible + 1
		if b.min < 0 {
			b.min = 0
		}
	}
}

// View renders the browser.
func (b Browser) View() string {
	var s strings.Builder

	// Available width for entry names (minus cursor "  " or "> ")
	cursorWidth := 2
	nameWidth := b.width - cursorWidth
	if nameWidth < 5 {
		nameWidth = 5
	}

	// Show current directory (truncated if needed)
	dir := b.currentDir
	maxDirLen := b.width - 2
	if maxDirLen < 10 {
		maxDirLen = 10
	}
	if len(dir) > maxDirLen {
		dir = "..." + dir[len(dir)-maxDirLen+3:]
	}
	s.WriteString(b.Styles.Muted.Render(dir))
	s.WriteRune('\n')

	// Handle errors
	if b.err != nil {
		s.WriteString(b.Styles.Muted.Render("Error: " + b.err.Error()))
		return b.constrainToHeight(s.String())
	}

	// Handle empty directory
	if len(b.entries) == 0 {
		s.WriteString(b.Styles.EmptyDir.Render("(empty)"))
		return b.constrainToHeight(s.String())
	}

	// Render entries - only render visible items within min/max range
	for i := b.min; i <= b.max && i < len(b.entries); i++ {
		entry := b.entries[i]
		isSelected := i == b.selected

		// Cursor
		cursor := "  "
		if isSelected {
			cursor = b.Styles.Cursor.Render("> ")
		}

		// Build display name
		var displayName string
		if entry.IsDir {
			displayName = "[" + entry.Name + "]"
		} else {
			displayName = entry.Name
		}

		// Truncate or scroll the name to fit
		displayName = b.fitName(displayName, nameWidth, isSelected)

		// Apply style
		var styledName string
		if entry.IsDir {
			if isSelected {
				styledName = b.Styles.SelectedDir.Render(displayName)
			} else {
				styledName = b.Styles.Directory.Render(displayName)
			}
		} else {
			if isSelected {
				styledName = b.Styles.Selected.Render(displayName)
			} else {
				styledName = b.Styles.VGMFile.Render(displayName)
			}
		}

		s.WriteString(cursor + styledName)
		s.WriteRune('\n')
	}

	return b.constrainToHeight(s.String())
}

// fitName truncates or scrolls a name to fit within the given width.
// For selected items, it scrolls to show the end of long names.
// For non-selected items, it truncates with "..." suffix.
func (b Browser) fitName(name string, maxWidth int, isSelected bool) string {
	if len(name) <= maxWidth {
		return name
	}

	if isSelected {
		// Scroll: show the end portion with "..." prefix
		visibleLen := maxWidth - 3 // Account for "..."
		if visibleLen < 1 {
			visibleLen = 1
		}
		return "..." + name[len(name)-visibleLen:]
	}

	// Truncate: show the beginning with "..." suffix
	visibleLen := maxWidth - 3 // Account for "..."
	if visibleLen < 1 {
		visibleLen = 1
	}
	return name[:visibleLen] + "..."
}

// constrainToHeight ensures the rendered content fits within the browser's height.
// It truncates lines that exceed the height limit.
func (b Browser) constrainToHeight(content string) string {
	if b.height <= 0 {
		return content
	}

	// Remove trailing newline to avoid off-by-one in split
	content = strings.TrimSuffix(content, "\n")

	lines := strings.Split(content, "\n")
	maxLines := b.height
	if maxLines <= 0 {
		maxLines = 1
	}

	// Truncate to fit within height
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	// Pad with empty lines if needed to maintain consistent height
	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// SetSize sets the browser dimensions.
func (b *Browser) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.max = b.min + b.visibleCount() - 1
	b.updateViewport()
}

// Focus sets the browser as focused.
func (b *Browser) Focus() {
	b.focused = true
}

// Blur removes focus from the browser.
func (b *Browser) Blur() {
	b.focused = false
}

// IsFocused returns whether the browser is focused.
func (b Browser) IsFocused() bool {
	return b.focused
}

// CurrentDir returns the current directory path.
func (b Browser) CurrentDir() string {
	return b.currentDir
}

// SelectedEntry returns the currently selected entry, or nil if none.
func (b Browser) SelectedEntry() *FileEntry {
	if len(b.entries) == 0 || b.selected < 0 || b.selected >= len(b.entries) {
		return nil
	}
	entry := b.entries[b.selected]
	return &entry
}
