// Package library provides VGM music library indexing and management.
package library

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dewi-tim/vgmtui/internal/player"
)

// VGM-compatible file extensions.
var vgmExtensions = []string{".vgm", ".vgz", ".s98", ".dro", ".gym"}

// Track represents a track in the library with full metadata.
type Track struct {
	Path     string
	Title    string
	Game     string
	System   string
	Composer string
	Duration time.Duration
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

		// Create library track
		libTrack := Track{
			Path:     path,
			Title:    track.Title,
			Game:     track.Game,
			System:   track.System,
			Composer: track.Composer,
			Duration: track.Duration,
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
	for _, system := range l.systems {
		for _, game := range system.Games {
			sort.Slice(game.Tracks, func(i, j int) bool {
				return game.Tracks[i].Title < game.Tracks[j].Title
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
