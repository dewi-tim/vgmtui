// Package components provides UI components for vgmtui.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dewi-tim/vgmtui/internal/library"
)

// LibBrowserKeyMap defines key bindings for the library browser.
type LibBrowserKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	GoToTop    key.Binding
	GoToBottom key.Binding
	Enter      key.Binding // Expand/collapse or select
	Back       key.Binding // Collapse or go to parent
	AddAll     key.Binding // Add entire game/system to playlist
}

// DefaultLibBrowserKeyMap returns the default library browser key bindings.
func DefaultLibBrowserKeyMap() LibBrowserKeyMap {
	return LibBrowserKeyMap{
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
		Enter: key.NewBinding(
			key.WithKeys("enter", "l", "right"),
			key.WithHelp("enter", "expand/select"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "h", "left"),
			key.WithHelp("backspace", "collapse"),
		),
		AddAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add all"),
		),
	}
}

// NodeType represents the type of tree node.
type NodeType int

const (
	NodeSystem NodeType = iota
	NodeGame
	NodeTrack
)

// TreeNode represents a node in the library tree.
type TreeNode struct {
	Type     NodeType
	Name     string
	System   string // For games and tracks
	Game     string // For tracks
	Path     string // For tracks
	Track    *library.Track
	Children []*TreeNode
	Expanded bool
	Parent   *TreeNode
}

// LibBrowser is a tree-based library browser component.
type LibBrowser struct {
	// Library data
	lib  *library.Library
	root []*TreeNode // Root nodes (systems)

	// Flat list for navigation
	flatList []*TreeNode

	// Selection state
	selected int
	min      int
	max      int

	// Dimensions
	width  int
	height int

	// State
	focused bool
	keyMap  LibBrowserKeyMap
	styles  LibBrowserStyles

	// Status
	scanning   bool
	trackCount int
}

// LibBrowserStyles contains styles for the library browser component.
type LibBrowserStyles struct {
	Cursor      lipgloss.Style
	System      lipgloss.Style
	Game        lipgloss.Style
	Track       lipgloss.Style
	Selected    lipgloss.Style
	Muted       lipgloss.Style
	TreeIndent  string
	Expanded    string
	Collapsed   string
	TrackBullet string
}

// DefaultLibBrowserStyles returns the default library browser styles.
func DefaultLibBrowserStyles() LibBrowserStyles {
	return LibBrowserStyles{
		Cursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true),
		System: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true),
		Game: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#99CCFF")),
		Track: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true),
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")),
		TreeIndent:  "  ",
		Expanded:    "[-]",
		Collapsed:   "[+]",
		TrackBullet: " - ",
	}
}

// LibBrowserScanCompleteMsg is sent when library scanning completes.
type LibBrowserScanCompleteMsg struct {
	TrackCount int
	Err        error
}

// LibTrackSelectedMsg is sent when a track is selected.
type LibTrackSelectedMsg struct {
	Track library.Track
}

// LibTracksSelectedMsg is sent when multiple tracks are selected (add all).
type LibTracksSelectedMsg struct {
	Tracks []library.Track
}

// NewLibBrowser creates a new library browser.
func NewLibBrowser(lib *library.Library) *LibBrowser {
	b := &LibBrowser{
		lib:      lib,
		root:     make([]*TreeNode, 0),
		flatList: make([]*TreeNode, 0),
		selected: 0,
		min:      0,
		max:      10,
		width:    30,
		height:   10,
		focused:  false,
		keyMap:   DefaultLibBrowserKeyMap(),
		styles:   DefaultLibBrowserStyles(),
		scanning: false,
	}
	return b
}

// Init initializes the library browser and starts scanning.
func (b *LibBrowser) Init() tea.Cmd {
	return b.Scan()
}

// Scan returns a command that scans the library.
func (b *LibBrowser) Scan() tea.Cmd {
	b.scanning = true
	return func() tea.Msg {
		count, err := b.lib.Scan()
		return LibBrowserScanCompleteMsg{TrackCount: count, Err: err}
	}
}

// buildTree builds the tree structure from the library.
func (b *LibBrowser) buildTree() {
	b.root = make([]*TreeNode, 0)

	systems := b.lib.Systems()
	for _, sysName := range systems {
		sysNode := &TreeNode{
			Type:     NodeSystem,
			Name:     sysName,
			Children: make([]*TreeNode, 0),
			Expanded: false,
		}

		games := b.lib.Games(sysName)
		for _, gameName := range games {
			gameNode := &TreeNode{
				Type:     NodeGame,
				Name:     gameName,
				System:   sysName,
				Children: make([]*TreeNode, 0),
				Expanded: false,
				Parent:   sysNode,
			}

			tracks := b.lib.Tracks(sysName, gameName)
			for i := range tracks {
				trackNode := &TreeNode{
					Type:   NodeTrack,
					Name:   tracks[i].Title,
					System: sysName,
					Game:   gameName,
					Path:   tracks[i].Path,
					Track:  &tracks[i],
					Parent: gameNode,
				}
				gameNode.Children = append(gameNode.Children, trackNode)
			}

			sysNode.Children = append(sysNode.Children, gameNode)
		}

		b.root = append(b.root, sysNode)
	}

	b.rebuildFlatList()
}

// rebuildFlatList rebuilds the flat list from the tree.
func (b *LibBrowser) rebuildFlatList() {
	b.flatList = make([]*TreeNode, 0)
	for _, system := range b.root {
		b.addToFlatList(system, 0)
	}

	// Ensure selected index is valid
	if b.selected >= len(b.flatList) {
		b.selected = len(b.flatList) - 1
	}
	if b.selected < 0 {
		b.selected = 0
	}

	b.updateViewport()
}

// addToFlatList adds a node and its visible children to the flat list.
func (b *LibBrowser) addToFlatList(node *TreeNode, depth int) {
	b.flatList = append(b.flatList, node)
	if node.Expanded {
		for _, child := range node.Children {
			b.addToFlatList(child, depth+1)
		}
	}
}

// Update handles messages and updates the browser state.
func (b *LibBrowser) Update(msg tea.Msg) (*LibBrowser, tea.Cmd) {
	switch msg := msg.(type) {
	case LibBrowserScanCompleteMsg:
		b.scanning = false
		if msg.Err == nil {
			b.trackCount = msg.TrackCount
			b.buildTree()
		}
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
func (b *LibBrowser) handleKeyMsg(msg tea.KeyMsg) (*LibBrowser, tea.Cmd) {
	switch {
	case key.Matches(msg, b.keyMap.Up):
		b.moveUp()
		return b, nil

	case key.Matches(msg, b.keyMap.Down):
		b.moveDown()
		return b, nil

	case key.Matches(msg, b.keyMap.PageUp):
		b.pageUp()
		return b, nil

	case key.Matches(msg, b.keyMap.PageDown):
		b.pageDown()
		return b, nil

	case key.Matches(msg, b.keyMap.GoToTop):
		b.goToTop()
		return b, nil

	case key.Matches(msg, b.keyMap.GoToBottom):
		b.goToBottom()
		return b, nil

	case key.Matches(msg, b.keyMap.Enter):
		return b.handleEnter()

	case key.Matches(msg, b.keyMap.Back):
		return b.handleBack()

	case key.Matches(msg, b.keyMap.AddAll):
		return b.handleAddAll()
	}

	return b, nil
}

// handleEnter handles Enter key - expand/collapse or select track.
func (b *LibBrowser) handleEnter() (*LibBrowser, tea.Cmd) {
	if len(b.flatList) == 0 {
		return b, nil
	}

	node := b.flatList[b.selected]

	switch node.Type {
	case NodeSystem, NodeGame:
		// Toggle expanded state
		node.Expanded = !node.Expanded
		b.rebuildFlatList()
		return b, nil

	case NodeTrack:
		// Select the track
		if node.Track != nil {
			return b, func() tea.Msg {
				return LibTrackSelectedMsg{Track: *node.Track}
			}
		}
	}

	return b, nil
}

// handleBack handles Back key - collapse node or go to parent.
func (b *LibBrowser) handleBack() (*LibBrowser, tea.Cmd) {
	if len(b.flatList) == 0 {
		return b, nil
	}

	node := b.flatList[b.selected]

	// If expanded, collapse it
	if node.Expanded {
		node.Expanded = false
		b.rebuildFlatList()
		return b, nil
	}

	// Otherwise, go to parent
	if node.Parent != nil {
		for i, n := range b.flatList {
			if n == node.Parent {
				b.selected = i
				b.updateViewport()
				return b, nil
			}
		}
	}

	return b, nil
}

// handleAddAll handles adding all tracks from selected game/system.
func (b *LibBrowser) handleAddAll() (*LibBrowser, tea.Cmd) {
	if len(b.flatList) == 0 {
		return b, nil
	}

	node := b.flatList[b.selected]
	var tracks []library.Track

	switch node.Type {
	case NodeSystem:
		// Add all tracks from system
		games := b.lib.Games(node.Name)
		for _, gameName := range games {
			tracks = append(tracks, b.lib.Tracks(node.Name, gameName)...)
		}

	case NodeGame:
		// Add all tracks from game
		tracks = b.lib.Tracks(node.System, node.Name)

	case NodeTrack:
		// Add single track
		if node.Track != nil {
			tracks = []library.Track{*node.Track}
		}
	}

	if len(tracks) > 0 {
		return b, func() tea.Msg {
			return LibTracksSelectedMsg{Tracks: tracks}
		}
	}

	return b, nil
}

// moveUp moves selection up one item.
func (b *LibBrowser) moveUp() {
	if b.selected > 0 {
		b.selected--
		if b.selected < b.min {
			b.min--
			b.max--
		}
	}
}

// moveDown moves selection down one item.
func (b *LibBrowser) moveDown() {
	if b.selected < len(b.flatList)-1 {
		b.selected++
		if b.selected > b.max {
			b.min++
			b.max++
		}
	}
}

// pageUp moves selection up one page.
func (b *LibBrowser) pageUp() {
	visible := b.visibleCount()
	b.selected -= visible
	if b.selected < 0 {
		b.selected = 0
	}
	b.updateViewport()
}

// pageDown moves selection down one page.
func (b *LibBrowser) pageDown() {
	visible := b.visibleCount()
	b.selected += visible
	if b.selected >= len(b.flatList) {
		b.selected = len(b.flatList) - 1
	}
	b.updateViewport()
}

// goToTop moves selection to the first item.
func (b *LibBrowser) goToTop() {
	b.selected = 0
	b.updateViewport()
}

// goToBottom moves selection to the last item.
func (b *LibBrowser) goToBottom() {
	b.selected = len(b.flatList) - 1
	if b.selected < 0 {
		b.selected = 0
	}
	b.updateViewport()
}

// visibleCount returns the number of visible items.
func (b *LibBrowser) visibleCount() int {
	count := b.height - 1 // Account for status line
	if count < 1 {
		count = 1
	}
	return count
}

// updateViewport ensures the selected item is visible.
func (b *LibBrowser) updateViewport() {
	visible := b.visibleCount()
	if visible <= 0 {
		visible = 1
	}

	// Ensure max is properly set based on visible count
	if b.max < b.min+visible-1 {
		b.max = b.min + visible - 1
	}

	// Adjust viewport to keep selection visible
	if b.selected < b.min {
		b.min = b.selected
		b.max = b.min + visible - 1
	} else if b.selected > b.max {
		b.max = b.selected
		b.min = b.max - visible + 1
	}

	// Clamp values
	if b.min < 0 {
		b.min = 0
		b.max = b.min + visible - 1
	}
	if b.max >= len(b.flatList) {
		b.max = len(b.flatList) - 1
	}
	if b.max < b.min {
		b.max = b.min
	}
}

// getDepth returns the depth of a node in the tree.
func (b *LibBrowser) getDepth(node *TreeNode) int {
	depth := 0
	for p := node.Parent; p != nil; p = p.Parent {
		depth++
	}
	return depth
}

// View renders the library browser.
func (b *LibBrowser) View() string {
	var s strings.Builder

	// Show status line with library root for debugging
	if b.scanning {
		s.WriteString(b.styles.Muted.Render(fmt.Sprintf("Scanning %s...", b.lib.Root())))
		return s.String()
	}

	statusLine := fmt.Sprintf("%d tracks in %s", b.trackCount, b.lib.Root())
	s.WriteString(b.styles.Muted.Render(statusLine))
	s.WriteRune('\n')

	// Handle empty library
	if len(b.flatList) == 0 {
		s.WriteString(b.styles.Muted.Render("No tracks found"))
		return b.constrainToHeight(s.String())
	}

	// Render visible nodes
	for i := b.min; i <= b.max && i < len(b.flatList); i++ {
		node := b.flatList[i]
		isSelected := i == b.selected
		depth := b.getDepth(node)

		// Cursor (2 chars visible)
		cursorStr := "  "
		if isSelected {
			cursorStr = "> "
		}

		// Indent based on depth (2 chars per level)
		indentStr := strings.Repeat("  ", depth)

		// Node content
		var content string
		var marker string

		switch node.Type {
		case NodeSystem:
			if node.Expanded {
				marker = "[-]"
			} else {
				marker = "[+]"
			}
			content = fmt.Sprintf("%s %s", marker, node.Name)

		case NodeGame:
			if node.Expanded {
				marker = "[-]"
			} else {
				marker = "[+]"
			}
			content = fmt.Sprintf("%s %s", marker, node.Name)

		case NodeTrack:
			content = fmt.Sprintf(" -  %s", node.Name)
		}

		// Fit to width (cursor=2, indent=2*depth, padding=2)
		maxWidth := b.width - 2 - (2 * depth) - 2
		if maxWidth < 10 {
			maxWidth = 10
		}
		if len(content) > maxWidth {
			content = content[:maxWidth-3] + "..."
		}

		// Apply styling
		var styledContent string
		if isSelected {
			styledContent = b.styles.Selected.Render(content)
		} else {
			switch node.Type {
			case NodeSystem:
				styledContent = b.styles.System.Render(content)
			case NodeGame:
				styledContent = b.styles.Game.Render(content)
			case NodeTrack:
				styledContent = b.styles.Track.Render(content)
			}
		}

		// Build line with cursor styling
		var line string
		if isSelected {
			line = b.styles.Cursor.Render(cursorStr) + indentStr + styledContent
		} else {
			line = cursorStr + indentStr + styledContent
		}

		s.WriteString(line)
		s.WriteRune('\n')
	}

	return b.constrainToHeight(s.String())
}

// constrainToHeight ensures the rendered content fits within the browser's height.
func (b *LibBrowser) constrainToHeight(content string) string {
	if b.height <= 0 {
		return content
	}

	content = strings.TrimSuffix(content, "\n")
	lines := strings.Split(content, "\n")

	maxLines := b.height
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	// Pad with empty lines
	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// SetSize sets the browser dimensions.
func (b *LibBrowser) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.max = b.min + b.visibleCount() - 1
	b.updateViewport()
}

// Focus sets the browser as focused.
func (b *LibBrowser) Focus() {
	b.focused = true
}

// Blur removes focus from the browser.
func (b *LibBrowser) Blur() {
	b.focused = false
}

// IsFocused returns whether the browser is focused.
func (b *LibBrowser) IsFocused() bool {
	return b.focused
}

// KeyMap returns the key map.
func (b *LibBrowser) KeyMap() LibBrowserKeyMap {
	return b.keyMap
}

// SelectedNode returns the currently selected node.
func (b *LibBrowser) SelectedNode() *TreeNode {
	if len(b.flatList) == 0 || b.selected < 0 || b.selected >= len(b.flatList) {
		return nil
	}
	return b.flatList[b.selected]
}
