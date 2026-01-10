// Package player provides VGM playback functionality using libvgm.
package player

import (
	"time"
)

// Track represents metadata about a VGM track.
type Track struct {
	// File path
	Path string

	// GD3 tag information
	Title    string // Track title (TITLE)
	Game     string // Game/album name (GAME)
	System   string // System/platform (SYSTEM)
	Composer string // Composer/artist (ARTIST)
	Date     string // Release date (DATE)
	VGMBy    string // VGM author (ENCODED_BY)
	Notes    string // Comments (COMMENT)

	// Format information
	Format string // e.g., "VGM 1.71", "S98 v3"

	// Timing information
	Duration  time.Duration // Total duration including loops
	LoopPoint time.Duration // Position where loop begins (0 if no loop)
	HasLoop   bool          // Whether the track loops

	// Sound chip information
	Chips []ChipInfo
}

// ChipInfo represents a sound chip used in a track.
type ChipInfo struct {
	Name string // Chip name, e.g., "YM2612"
	Core string // Emulation core, e.g., "GPGX"
}

// PlayState represents the current playback state.
type PlayState int

const (
	// StateStopped indicates playback is stopped.
	StateStopped PlayState = iota
	// StatePlaying indicates playback is active.
	StatePlaying
	// StatePaused indicates playback is paused.
	StatePaused
	// StateFading indicates fade-out is in progress.
	StateFading
)

// String returns a human-readable name for the play state.
func (s PlayState) String() string {
	switch s {
	case StateStopped:
		return "Stopped"
	case StatePlaying:
		return "Playing"
	case StatePaused:
		return "Paused"
	case StateFading:
		return "Fading"
	default:
		return "Unknown"
	}
}

// PlaybackInfo contains information about the current playback.
type PlaybackInfo struct {
	// Current state
	State PlayState

	// Position information
	Position time.Duration // Current playback position
	Duration time.Duration // Total duration

	// Loop information
	CurrentLoop int  // Current loop number (0 = first play)
	TotalLoops  int  // Configured number of loops
	HasLoop     bool // Whether the track has a loop point

	// Playback settings
	Volume float64 // Volume (0.0 - 1.0+)
	Speed  float64 // Playback speed (1.0 = normal)
}

// Progress returns the playback progress as a value between 0.0 and 1.0.
func (p *PlaybackInfo) Progress() float64 {
	if p.Duration == 0 {
		return 0.0
	}
	progress := float64(p.Position) / float64(p.Duration)
	if progress > 1.0 {
		return 1.0
	}
	if progress < 0.0 {
		return 0.0
	}
	return progress
}

// Remaining returns the remaining playback time.
func (p *PlaybackInfo) Remaining() time.Duration {
	remaining := p.Duration - p.Position
	if remaining < 0 {
		return 0
	}
	return remaining
}
