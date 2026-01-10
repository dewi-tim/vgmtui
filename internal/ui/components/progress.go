// Package components provides UI components for vgmtui.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressBar wraps the bubbles progress component with time display.
type ProgressBar struct {
	progress progress.Model
	elapsed  time.Duration
	duration time.Duration
	width    int

	// Styles
	TimeStyle     lipgloss.Style
	FilledStyle   lipgloss.Style
	EmptyStyle    lipgloss.Style
	FilledChar    rune
	EmptyChar     rune
}

// NewProgressBar creates a new progress bar with default styling.
func NewProgressBar() ProgressBar {
	p := progress.New(
		progress.WithoutPercentage(),
		progress.WithDefaultGradient(),
	)

	return ProgressBar{
		progress:    p,
		width:       40,
		FilledChar:  '\u2588', // Full block
		EmptyChar:   '\u2591', // Light shade
		TimeStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")),
		FilledStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#7571F9")),
		EmptyStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#606060")),
	}
}

// SetWidth sets the total width available for the progress bar.
func (p *ProgressBar) SetWidth(width int) {
	p.width = width
	// Subtract space for time display: "00:00 " + " 00:00" = 13 chars
	barWidth := width - 13
	if barWidth < 10 {
		barWidth = 10
	}
	p.progress.Width = barWidth
}

// SetElapsed sets the current elapsed time.
func (p *ProgressBar) SetElapsed(d time.Duration) {
	p.elapsed = d
}

// SetDuration sets the total duration.
func (p *ProgressBar) SetDuration(d time.Duration) {
	p.duration = d
}

// Update updates the progress bar state.
func (p ProgressBar) Update(msg tea.Msg) (ProgressBar, tea.Cmd) {
	var cmd tea.Cmd
	var m tea.Model
	m, cmd = p.progress.Update(msg)
	p.progress = m.(progress.Model)
	return p, cmd
}

// View renders the progress bar with time display.
// Format: "01:23 [=====>----] 03:45"
func (p ProgressBar) View() string {
	// Calculate percentage
	var percent float64
	if p.duration > 0 {
		percent = float64(p.elapsed) / float64(p.duration)
		if percent > 1 {
			percent = 1
		}
		if percent < 0 {
			percent = 0
		}
	}

	// Format times
	elapsedStr := formatDuration(p.elapsed)
	durationStr := formatDuration(p.duration)

	// Build custom progress bar
	barWidth := p.width - len(elapsedStr) - len(durationStr) - 2 // 2 spaces
	if barWidth < 5 {
		barWidth = 5
	}

	filledWidth := int(float64(barWidth) * percent)
	emptyWidth := barWidth - filledWidth

	filled := p.FilledStyle.Render(strings.Repeat(string(p.FilledChar), filledWidth))
	empty := p.EmptyStyle.Render(strings.Repeat(string(p.EmptyChar), emptyWidth))

	bar := filled + empty

	return fmt.Sprintf("%s %s %s",
		p.TimeStyle.Render(elapsedStr),
		bar,
		p.TimeStyle.Render(durationStr),
	)
}

// ViewAs renders the progress bar at a specific percentage (0.0 to 1.0).
func (p ProgressBar) ViewAs(percent float64, elapsed, duration time.Duration) string {
	if percent > 1 {
		percent = 1
	}
	if percent < 0 {
		percent = 0
	}

	// Format times
	elapsedStr := formatDuration(elapsed)
	durationStr := formatDuration(duration)

	// Build custom progress bar
	barWidth := p.width - len(elapsedStr) - len(durationStr) - 2 // 2 spaces
	if barWidth < 5 {
		barWidth = 5
	}

	filledWidth := int(float64(barWidth) * percent)
	emptyWidth := barWidth - filledWidth

	filled := p.FilledStyle.Render(strings.Repeat(string(p.FilledChar), filledWidth))
	empty := p.EmptyStyle.Render(strings.Repeat(string(p.EmptyChar), emptyWidth))

	bar := filled + empty

	return fmt.Sprintf("%s %s %s",
		p.TimeStyle.Render(elapsedStr),
		bar,
		p.TimeStyle.Render(durationStr),
	)
}

// formatDuration formats a duration as MM:SS.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	minutes := total / 60
	seconds := total % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
