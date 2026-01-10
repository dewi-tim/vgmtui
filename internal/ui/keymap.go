package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the application.
type KeyMap struct {
	// Playback controls
	PlayPause key.Binding
	NextTrack key.Binding
	PrevTrack key.Binding
	Stop      key.Binding

	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	TabFocus key.Binding

	// Seek controls
	SeekForward  key.Binding
	SeekBackward key.Binding

	// Volume
	VolumeUp   key.Binding
	VolumeDown key.Binding

	// Help and Quit
	Help key.Binding
	Quit key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Playback
		PlayPause: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "play/pause"),
		),
		NextTrack: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next"),
		),
		PrevTrack: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop"),
		),

		// Navigation
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/left", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/right", "right"),
		),
		TabFocus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "focus"),
		),

		// Seek
		SeekForward: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "+5s"),
		),
		SeekBackward: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "-5s"),
		),

		// Volume
		VolumeUp: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "vol+"),
		),
		VolumeDown: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "vol-"),
		),

		// Help and Quit
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings to show in the short help view.
// Implements the help.KeyMap interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.PlayPause,
		k.NextTrack,
		k.PrevTrack,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns keybindings to show in the full help view.
// Implements the help.KeyMap interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Playback column
		{
			k.PlayPause,
			k.NextTrack,
			k.PrevTrack,
			k.Stop,
		},
		// Navigation column
		{
			k.Up,
			k.Down,
			k.TabFocus,
		},
		// Seek/Volume column
		{
			k.SeekForward,
			k.SeekBackward,
			k.VolumeUp,
			k.VolumeDown,
		},
		// System column
		{
			k.Help,
			k.Quit,
		},
	}
}
