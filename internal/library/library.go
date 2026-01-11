// Package library provides VGM music library indexing and management.
package library

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dewi-tim/vgmtui/internal/player"
)

// VGM-compatible file extensions.
var vgmExtensions = []string{".vgm", ".vgz", ".s98", ".dro", ".gym"}

// trackNumberPatterns matches common track number formats in filenames.
var trackNumberPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^(\d{1,3})\s*[-._)\]]\s*`),     // "01 - Title", "01_Title", "01.Title", "01) Title"
	regexp.MustCompile(`^\[(\d{1,3})\]`),                // "[01] Title"
	regexp.MustCompile(`^\((\d{1,3})\)`),                // "(01) Title"
	regexp.MustCompile(`(?i)^track\s*(\d{1,3})`),        // "Track 01", "Track01"
	regexp.MustCompile(`^(\d{1,3})$`),                   // "01" (just a number, after extension stripped)
	regexp.MustCompile(`^(\d{1,3})\s`),                  // "01 Title" (number followed by space)
}

// Track represents a track in the library with full metadata.
type Track struct {
	Path        string
	Title       string
	Game        string
	System      string
	Composer    string
	Duration    time.Duration
	TrackNumber int // 1-indexed track number, 0 if unknown
}

// Game represents a game/album containing tracks.
type Game struct {
	Name   string
	System string
	Tracks []Track
}

// System represents a system/platform containing games.
type System struct {
	Name  string
	Games map[string]*Game
}

// Library represents an indexed VGM music library.
type Library struct {
	mu      sync.RWMutex
	root    string
	systems map[string]*System
	tracks  []Track // Flat list for quick access
}

// New creates a new library rooted at the given directory.
func New(root string) *Library {
	return &Library{
		root:    root,
		systems: make(map[string]*System),
		tracks:  make([]Track, 0),
	}
}

// Root returns the library root directory.
func (l *Library) Root() string {
	return l.root
}

// Scan scans the library directory and indexes all VGM files.
// Returns the number of tracks found.
func (l *Library) Scan() (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Clear existing data
	l.systems = make(map[string]*System)
	l.tracks = make([]Track, 0)

	// Walk the directory tree
	err := filepath.Walk(l.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories and hidden files
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a VGM file
		if !isVGMFile(info.Name()) {
			return nil
		}

		// Read metadata
		track, err := player.ReadTrackMetadata(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		// Extract track number from filename
		trackNum := extractTrackNumber(info.Name())

		// Create library track
		libTrack := Track{
			Path:        path,
			Title:       track.Title,
			Game:        track.Game,
			System:      track.System,
			Composer:    track.Composer,
			Duration:    track.Duration,
			TrackNumber: trackNum,
		}

		// Use filename as title if empty
		if libTrack.Title == "" {
			libTrack.Title = strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		}

		// Use parent directory as game if empty
		if libTrack.Game == "" {
			libTrack.Game = filepath.Base(filepath.Dir(path))
		}

		// Use "Unknown" as system if empty
		if libTrack.System == "" {
			libTrack.System = "Unknown"
		}

		// Add to flat list
		l.tracks = append(l.tracks, libTrack)

		// Add to hierarchy
		l.addTrack(libTrack)

		return nil
	})

	if err != nil {
		return 0, err
	}

	// Sort tracks within each game
	// Priority: M3U playlist order > filename track numbers > path (alphabetical)
	for _, system := range l.systems {
		for _, game := range system.Games {
			// Try to get track order from M3U file in game directory
			applyM3UOrder(game)

			// Sort by track number, falling back to path for ties or missing numbers
			sort.SliceStable(game.Tracks, func(i, j int) bool {
				ti, tj := game.Tracks[i].TrackNumber, game.Tracks[j].TrackNumber
				// Both have track numbers: sort by number
				if ti > 0 && tj > 0 {
					return ti < tj
				}
				// Only one has a track number: it comes first
				if ti > 0 {
					return true
				}
				if tj > 0 {
					return false
				}
				// Neither has a track number: sort by path
				return game.Tracks[i].Path < game.Tracks[j].Path
			})
		}
	}

	return len(l.tracks), nil
}

// addTrack adds a track to the library hierarchy.
func (l *Library) addTrack(track Track) {
	// Get or create system
	system, ok := l.systems[track.System]
	if !ok {
		system = &System{
			Name:  track.System,
			Games: make(map[string]*Game),
		}
		l.systems[track.System] = system
	}

	// Get or create game
	game, ok := system.Games[track.Game]
	if !ok {
		game = &Game{
			Name:   track.Game,
			System: track.System,
			Tracks: make([]Track, 0),
		}
		system.Games[track.Game] = game
	}

	// Add track to game
	game.Tracks = append(game.Tracks, track)
}

// Systems returns a sorted list of system names.
func (l *Library) Systems() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.systems))
	for name := range l.systems {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSystem returns a system by name.
func (l *Library) GetSystem(name string) *System {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.systems[name]
}

// Games returns a sorted list of game names for a system.
func (l *Library) Games(systemName string) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	system, ok := l.systems[systemName]
	if !ok {
		return nil
	}

	names := make([]string, 0, len(system.Games))
	for name := range system.Games {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetGame returns a game by system and game name.
func (l *Library) GetGame(systemName, gameName string) *Game {
	l.mu.RLock()
	defer l.mu.RUnlock()

	system, ok := l.systems[systemName]
	if !ok {
		return nil
	}
	return system.Games[gameName]
}

// Tracks returns a sorted list of tracks for a game.
func (l *Library) Tracks(systemName, gameName string) []Track {
	l.mu.RLock()
	defer l.mu.RUnlock()

	system, ok := l.systems[systemName]
	if !ok {
		return nil
	}
	game, ok := system.Games[gameName]
	if !ok {
		return nil
	}
	return game.Tracks
}

// AllTracks returns all tracks in the library.
func (l *Library) AllTracks() []Track {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]Track, len(l.tracks))
	copy(result, l.tracks)
	return result
}

// TrackCount returns the total number of tracks.
func (l *Library) TrackCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return len(l.tracks)
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

// extractTrackNumber extracts a track number from a filename.
// Returns 0 if no track number is found.
// Examples: "01 - Title.vgm" -> 1, "Track01.vgm" -> 1, "(02) Song.vgm" -> 2
func extractTrackNumber(filename string) int {
	// Remove extension first
	name := filename
	for _, ext := range vgmExtensions {
		if strings.HasSuffix(strings.ToLower(name), ext) {
			name = name[:len(name)-len(ext)]
			break
		}
	}

	// Try each pattern
	for _, pattern := range trackNumberPatterns {
		if matches := pattern.FindStringSubmatch(name); matches != nil {
			if num, err := strconv.Atoi(matches[1]); err == nil && num > 0 && num <= 999 {
				return num
			}
		}
	}

	return 0
}

// applyM3UOrder looks for an M3U playlist file in the game's directory
// and assigns track numbers based on the playlist order.
// M3U order takes priority over filename-extracted numbers.
func applyM3UOrder(game *Game) {
	if len(game.Tracks) == 0 {
		return
	}

	// Find the directory containing the tracks
	gameDir := filepath.Dir(game.Tracks[0].Path)

	// Look for M3U files
	entries, err := os.ReadDir(gameDir)
	if err != nil {
		return
	}

	var m3uPath string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lower := strings.ToLower(entry.Name())
		if strings.HasSuffix(lower, ".m3u") || strings.HasSuffix(lower, ".m3u8") {
			m3uPath = filepath.Join(gameDir, entry.Name())
			break // Use the first M3U found
		}
	}

	if m3uPath == "" {
		return
	}

	// Parse M3U and build filename -> position map
	m3uOrder := parseM3U(m3uPath)
	if len(m3uOrder) == 0 {
		return
	}

	// Assign track numbers from M3U order
	for i := range game.Tracks {
		filename := strings.ToLower(filepath.Base(game.Tracks[i].Path))
		if pos, ok := m3uOrder[filename]; ok {
			game.Tracks[i].TrackNumber = pos
		}
	}
}

// parseM3U reads an M3U playlist file and returns a map of
// lowercase filename -> 1-indexed position.
func parseM3U(path string) map[string]int {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	order := make(map[string]int)
	position := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments (including extended M3U tags like #EXTINF)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Normalize path separators (M3U might use Windows backslashes)
		line = strings.ReplaceAll(line, "\\", "/")

		// Extract just the filename
		filename := filepath.Base(line)

		// Only count VGM files
		if isVGMFile(filename) {
			position++
			order[strings.ToLower(filename)] = position
		}
	}

	return order
}
