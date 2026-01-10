package player

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

const (
	// Default audio settings
	DefaultSampleRate   = 44100
	DefaultChannels     = 2
	DefaultBitDepth     = 16
	DefaultBufferSize   = 4096 // frames
	DefaultLoopCount    = 2
	DefaultFadeTime     = 4000 // ms
	DefaultEndSilence   = 1000 // ms
	DefaultTickInterval = 50 * time.Millisecond
)

// Player is the high-level interface for VGM playback.
type Player interface {
	// Load loads a track from a file path.
	Load(path string) error
	// Unload unloads the current track.
	Unload()

	// Play starts or resumes playback.
	Play() error
	// Pause pauses playback.
	Pause()
	// Stop stops playback.
	Stop()
	// Toggle toggles between play and pause.
	Toggle()

	// Seek seeks to a position in the track.
	Seek(pos time.Duration)
	// SeekRelative seeks relative to current position.
	SeekRelative(delta time.Duration)

	// FadeOut triggers a fade-out.
	FadeOut()
	// Reset resets playback to the beginning.
	Reset()

	// SetVolume sets the volume (0.0 - 1.0+).
	SetVolume(vol float64)
	// SetSpeed sets the playback speed (0.5 - 2.0).
	SetSpeed(speed float64)
	// SetLoopCount sets the number of loops.
	SetLoopCount(count int)

	// Track returns metadata about the current track.
	Track() *Track
	// Info returns current playback information.
	Info() PlaybackInfo
	// IsLoaded returns true if a track is loaded.
	IsLoaded() bool

	// Subscribe returns a channel that receives playback info updates.
	Subscribe() <-chan PlaybackInfo
	// Unsubscribe removes a subscription channel.
	Unsubscribe(ch <-chan PlaybackInfo)

	// Close releases all resources.
	Close() error
}

// AudioPlayer implements Player using libvgm and oto.
type AudioPlayer struct {
	mu sync.RWMutex

	// libvgm player
	vgm *LibvgmPlayer

	// oto audio context and player
	otoCtx    *oto.Context
	otoPlayer *oto.Player

	// Current track info
	track     *Track
	trackPath string

	// Playback state
	playing   bool
	paused    bool
	volume    float64
	speed     float64
	loopCount int

	// Audio buffer
	sampleRate int
	buffer     []int16

	// Render goroutine control
	ctx    context.Context
	cancel context.CancelFunc

	// Subscribers for playback info updates
	subscribers map[chan PlaybackInfo]struct{}
	subMu       sync.RWMutex
}

// NewAudioPlayer creates a new audio player.
func NewAudioPlayer() (*AudioPlayer, error) {
	// Create libvgm player
	vgm, err := NewLibvgmPlayer()
	if err != nil {
		return nil, fmt.Errorf("failed to create libvgm player: %w", err)
	}

	// Create oto audio context
	otoOpts := &oto.NewContextOptions{
		SampleRate:   DefaultSampleRate,
		ChannelCount: DefaultChannels,
		Format:       oto.FormatSignedInt16LE,
	}

	otoCtx, ready, err := oto.NewContext(otoOpts)
	if err != nil {
		vgm.Close()
		return nil, fmt.Errorf("failed to create audio context: %w", err)
	}

	// Wait for audio context to be ready
	<-ready

	ctx, cancel := context.WithCancel(context.Background())

	p := &AudioPlayer{
		vgm:         vgm,
		otoCtx:      otoCtx,
		sampleRate:  DefaultSampleRate,
		buffer:      make([]int16, DefaultBufferSize*DefaultChannels),
		volume:      1.0,
		speed:       1.0,
		loopCount:   DefaultLoopCount,
		ctx:         ctx,
		cancel:      cancel,
		subscribers: make(map[chan PlaybackInfo]struct{}),
	}

	// Configure libvgm
	vgm.SetSampleRate(uint32(DefaultSampleRate))
	vgm.SetLoopCount(uint32(DefaultLoopCount))
	vgm.SetFadeTime(DefaultFadeTime)
	vgm.SetEndSilence(DefaultEndSilence)

	return p, nil
}

// Load loads a track from a file path.
func (p *AudioPlayer) Load(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Stop current playback
	p.stopLocked()

	// Unload previous track
	p.vgm.Unload()

	// Load new file
	if err := p.vgm.Load(path); err != nil {
		return err
	}

	// Get track metadata
	track := p.vgm.GetTrack(path)
	p.track = &track
	p.trackPath = path

	return nil
}

// Unload unloads the current track.
func (p *AudioPlayer) Unload() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopLocked()
	p.vgm.Unload()
	p.track = nil
	p.trackPath = ""
}

// Play starts or resumes playback.
func (p *AudioPlayer) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.track == nil {
		return fmt.Errorf("no track loaded")
	}

	// If paused, just resume
	if p.paused {
		p.paused = false
		p.otoPlayer.Play()
		return nil
	}

	// If already playing, do nothing
	if p.playing {
		return nil
	}

	// Start libvgm playback
	if err := p.vgm.Start(); err != nil {
		return err
	}

	// Update track info with chip info (available after start)
	track := p.vgm.GetTrack(p.trackPath)
	p.track = &track

	// Create audio stream
	p.playing = true
	p.paused = false

	// Create oto player with this as the reader
	p.otoPlayer = p.otoCtx.NewPlayer(p)

	// Start playback
	p.otoPlayer.Play()

	// Start tick goroutine
	go p.tickLoop()

	return nil
}

// Pause pauses playback.
func (p *AudioPlayer) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.playing && !p.paused {
		p.paused = true
		if p.otoPlayer != nil {
			p.otoPlayer.Pause()
		}
	}
}

// Stop stops playback.
func (p *AudioPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopLocked()
}

func (p *AudioPlayer) stopLocked() {
	if p.playing {
		p.playing = false
		p.paused = false
		if p.otoPlayer != nil {
			p.otoPlayer.Close()
			p.otoPlayer = nil
		}
		p.vgm.Stop()
	}
}

// Toggle toggles between play and pause.
func (p *AudioPlayer) Toggle() {
	p.mu.RLock()
	playing := p.playing
	paused := p.paused
	p.mu.RUnlock()

	if playing {
		if paused {
			p.Play()
		} else {
			p.Pause()
		}
	} else {
		p.Play()
	}
}

// Seek seeks to a position in the track.
func (p *AudioPlayer) Seek(pos time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if pos < 0 {
		pos = 0
	}
	p.vgm.Seek(pos)
}

// SeekRelative seeks relative to current position.
func (p *AudioPlayer) SeekRelative(delta time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	current := p.vgm.Position()
	newPos := current + delta
	if newPos < 0 {
		newPos = 0
	}
	p.vgm.Seek(newPos)
}

// FadeOut triggers a fade-out.
func (p *AudioPlayer) FadeOut() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.vgm.FadeOut()
}

// Reset resets playback to the beginning.
func (p *AudioPlayer) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.vgm.Reset()
}

// SetVolume sets the volume (0.0 - 1.0+).
func (p *AudioPlayer) SetVolume(vol float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if vol < 0 {
		vol = 0
	}
	p.volume = vol
	p.vgm.SetVolume(vol)
}

// SetSpeed sets the playback speed (0.5 - 2.0).
func (p *AudioPlayer) SetSpeed(speed float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if speed < 0.1 {
		speed = 0.1
	}
	if speed > 8.0 {
		speed = 8.0
	}
	p.speed = speed
	p.vgm.SetSpeed(speed)
}

// SetLoopCount sets the number of loops.
func (p *AudioPlayer) SetLoopCount(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if count < 0 {
		count = 0
	}
	p.loopCount = count
	p.vgm.SetLoopCount(uint32(count))
}

// Track returns metadata about the current track.
func (p *AudioPlayer) Track() *Track {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.track
}

// Info returns current playback information.
func (p *AudioPlayer) Info() PlaybackInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	info := p.vgm.GetPlaybackInfo()

	// Adjust state based on our paused flag
	if p.paused {
		info.State = StatePaused
	} else if !p.playing {
		info.State = StateStopped
	}

	info.Volume = p.volume
	info.Speed = p.speed
	info.TotalLoops = p.loopCount

	return info
}

// IsLoaded returns true if a track is loaded.
func (p *AudioPlayer) IsLoaded() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.track != nil
}

// Subscribe returns a channel that receives playback info updates.
func (p *AudioPlayer) Subscribe() <-chan PlaybackInfo {
	p.subMu.Lock()
	defer p.subMu.Unlock()

	ch := make(chan PlaybackInfo, 1)
	p.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscription channel.
func (p *AudioPlayer) Unsubscribe(ch <-chan PlaybackInfo) {
	p.subMu.Lock()
	defer p.subMu.Unlock()

	// Find and remove the channel
	for subCh := range p.subscribers {
		if subCh == ch {
			delete(p.subscribers, subCh)
			close(subCh)
			break
		}
	}
}

// Read implements io.Reader for oto.
// This is called by oto to get audio data.
func (p *AudioPlayer) Read(buf []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.playing || p.paused {
		// Return silence when paused
		for i := range buf {
			buf[i] = 0
		}
		return len(buf), nil
	}

	// Calculate frames from bytes (stereo 16-bit)
	frames := len(buf) / 4

	// Ensure buffer is large enough
	if len(p.buffer) < frames*2 {
		p.buffer = make([]int16, frames*2)
	}

	// Render audio from libvgm
	rendered := p.vgm.Render(uint32(frames), p.buffer)

	// Check if playback finished
	if rendered == 0 || p.vgm.IsFinished() {
		p.playing = false
		// Return silence
		for i := range buf {
			buf[i] = 0
		}
		return len(buf), nil
	}

	// Convert int16 samples to bytes (little-endian)
	for i := 0; i < int(rendered)*2; i++ {
		sample := p.buffer[i]
		buf[i*2] = byte(sample)
		buf[i*2+1] = byte(sample >> 8)
	}

	// Zero remaining buffer if not fully filled
	for i := int(rendered) * 4; i < len(buf); i++ {
		buf[i] = 0
	}

	return len(buf), nil
}

// tickLoop sends periodic playback info updates to subscribers.
func (p *AudioPlayer) tickLoop() {
	ticker := time.NewTicker(DefaultTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mu.RLock()
			playing := p.playing
			p.mu.RUnlock()

			if !playing {
				return
			}

			info := p.Info()

			// Send to all subscribers (non-blocking)
			p.subMu.RLock()
			for ch := range p.subscribers {
				select {
				case ch <- info:
				default:
					// Drop if channel is full
				}
			}
			p.subMu.RUnlock()

			// Check if finished
			if info.State == StateStopped {
				return
			}
		}
	}
}

// Close releases all resources.
func (p *AudioPlayer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Cancel tick loop
	p.cancel()

	// Stop playback
	p.stopLocked()

	// Close subscribers
	p.subMu.Lock()
	for ch := range p.subscribers {
		close(ch)
	}
	p.subscribers = nil
	p.subMu.Unlock()

	// Close libvgm
	p.vgm.Close()

	return nil
}

// Ensure AudioPlayer implements Player
var _ Player = (*AudioPlayer)(nil)
