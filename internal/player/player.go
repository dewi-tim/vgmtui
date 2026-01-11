package player

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Default audio settings
	DefaultSampleRate   = 44100
	DefaultChannels     = 2
	DefaultBitDepth     = 16
	DefaultLoopCount    = 2
	DefaultFadeTime     = 4000 // ms
	DefaultEndSilence   = 1000 // ms
	DefaultTickInterval = 50 * time.Millisecond

	// Audio buffer settings for libvgm audio driver
	// Using smaller buffers than oto for lower latency
	AudioBufferTimeUsec  = 10000 // 10ms per buffer
	AudioBufferCount     = 8     // 80ms total latency
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

// AudioPlayer implements Player using libvgm with native audio drivers.
type AudioPlayer struct {
	// Atomic state for lock-free access
	playingAtomic uint32 // 1 = playing, 0 = not
	pausedAtomic  uint32 // 1 = paused, 0 = not

	// Mutex for non-hot-path operations (track loading, config changes)
	mu sync.Mutex

	// libvgm player
	vgm *LibvgmPlayer

	// libvgm audio driver (replaces oto)
	audioDriver *AudioDriver

	// Current track info
	track     *Track
	trackPath string

	// Playback config (protected by mu)
	volume    float64
	speed     float64
	loopCount int
	sampleRate int

	// Render goroutine control
	ctx    context.Context
	cancel context.CancelFunc

	// Subscribers for playback info updates
	subscribers map[chan PlaybackInfo]struct{}
	subMu       sync.RWMutex

	// WaitGroup to track tickLoop goroutine
	tickWg sync.WaitGroup
}

// selectAudioDriver finds the best available audio driver.
// Prefers PulseAudio, falls back to ALSA.
func selectAudioDriver() (uint32, error) {
	drivers := GetAudioDrivers()
	if len(drivers) == 0 {
		return 0, ErrAudioNoDrivers
	}

	// First pass: look for PulseAudio
	for _, drv := range drivers {
		if drv.Signature == AudioDriverSigPulse {
			return drv.ID, nil
		}
	}

	// Second pass: look for ALSA
	for _, drv := range drivers {
		if drv.Signature == AudioDriverSigALSA {
			return drv.ID, nil
		}
	}

	// Fallback: use first available output driver
	return drivers[0].ID, nil
}

// NewAudioPlayer creates a new audio player.
func NewAudioPlayer() (*AudioPlayer, error) {
	// Initialize libvgm audio system
	if err := InitAudioSystem(); err != nil {
		return nil, fmt.Errorf("failed to initialize audio system: %w", err)
	}

	// Select best audio driver (PulseAudio > ALSA)
	driverID, err := selectAudioDriver()
	if err != nil {
		DeinitAudioSystem()
		return nil, fmt.Errorf("no audio drivers available: %w", err)
	}

	// Create audio driver instance
	audioDriver, err := NewAudioDriver(driverID)
	if err != nil {
		DeinitAudioSystem()
		return nil, fmt.Errorf("failed to create audio driver: %w", err)
	}

	// Configure audio driver
	audioDriver.SetSampleRate(DefaultSampleRate)
	audioDriver.SetChannels(DefaultChannels)
	audioDriver.SetBits(DefaultBitDepth)
	audioDriver.SetBufferTime(AudioBufferTimeUsec)
	audioDriver.SetBufferCount(AudioBufferCount)

	// Create libvgm player
	vgm, err := NewLibvgmPlayer()
	if err != nil {
		audioDriver.Close()
		DeinitAudioSystem()
		return nil, fmt.Errorf("failed to create libvgm player: %w", err)
	}

	// Bind player to audio driver
	if err := audioDriver.BindPlayer(vgm); err != nil {
		vgm.Close()
		audioDriver.Close()
		DeinitAudioSystem()
		return nil, fmt.Errorf("failed to bind player to audio driver: %w", err)
	}

	// Start audio driver (it will call the render callback when needed)
	if err := audioDriver.Start(0); err != nil {
		vgm.Close()
		audioDriver.Close()
		DeinitAudioSystem()
		return nil, fmt.Errorf("failed to start audio driver: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &AudioPlayer{
		vgm:         vgm,
		audioDriver: audioDriver,
		sampleRate:  DefaultSampleRate,
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

	// Audio driver is started but paused - it will render silence until
	// a track is started with Play()

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

	return p.playLocked()
}

// Pause pauses playback.
func (p *AudioPlayer) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pauseLocked()
}

// Stop stops playback.
func (p *AudioPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopLocked()
}

func (p *AudioPlayer) stopLocked() {
	if atomic.LoadUint32(&p.playingAtomic) == 1 {
		// Set atomic flags first
		atomic.StoreUint32(&p.playingAtomic, 0)
		atomic.StoreUint32(&p.pausedAtomic, 0)

		// Stop libvgm (thread-safe via audio driver's mutex)
		p.vgm.Stop()

		// Pause audio output
		p.audioDriver.Pause()
	}
}

// Toggle toggles between play and pause.
func (p *AudioPlayer) Toggle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	playing := atomic.LoadUint32(&p.playingAtomic) == 1
	paused := atomic.LoadUint32(&p.pausedAtomic) == 1

	if playing {
		if paused {
			p.playLocked()
		} else {
			p.pauseLocked()
		}
	} else {
		p.playLocked()
	}
}

// pauseLocked pauses playback (must be called with mu held).
func (p *AudioPlayer) pauseLocked() {
	if atomic.LoadUint32(&p.playingAtomic) == 1 && atomic.LoadUint32(&p.pausedAtomic) == 0 {
		atomic.StoreUint32(&p.pausedAtomic, 1)
		p.audioDriver.Pause()
	}
}

// playLocked starts or resumes playback (must be called with mu held).
func (p *AudioPlayer) playLocked() error {
	if p.track == nil {
		return fmt.Errorf("no track loaded")
	}

	// If paused, just resume
	if atomic.LoadUint32(&p.pausedAtomic) == 1 {
		atomic.StoreUint32(&p.pausedAtomic, 0)
		p.audioDriver.Resume()
		return nil
	}

	// If already playing, do nothing
	if atomic.LoadUint32(&p.playingAtomic) == 1 {
		return nil
	}

	// Start libvgm playback
	if err := p.vgm.Start(); err != nil {
		return err
	}

	// Update track info with chip info (available after start)
	track := p.vgm.GetTrack(p.trackPath)
	p.track = &track

	// Set atomic state flags
	atomic.StoreUint32(&p.pausedAtomic, 0)
	atomic.StoreUint32(&p.playingAtomic, 1)

	// Resume audio driver (it's already started, just might be paused)
	p.audioDriver.Resume()

	// Start tick goroutine with WaitGroup tracking
	p.tickWg.Add(1)
	go p.tickLoop()

	return nil
}

// Seek seeks to a position in the track.
func (p *AudioPlayer) Seek(pos time.Duration) {
	if pos < 0 {
		pos = 0
	}
	// Use audio driver's thread-safe seek
	p.audioDriver.SafeSeek(pos)
}

// SeekRelative seeks relative to current position.
func (p *AudioPlayer) SeekRelative(delta time.Duration) {
	current := p.vgm.Position()
	newPos := current + delta
	if newPos < 0 {
		newPos = 0
	}
	p.audioDriver.SafeSeek(newPos)
}

// FadeOut triggers a fade-out.
func (p *AudioPlayer) FadeOut() {
	p.audioDriver.SafeFadeOut()
}

// Reset resets playback to the beginning.
func (p *AudioPlayer) Reset() {
	p.audioDriver.SafeReset()
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
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.track
}

// Info returns current playback information.
func (p *AudioPlayer) Info() PlaybackInfo {
	// Get libvgm info - these CGO calls are safe without mutex
	info := p.vgm.GetPlaybackInfo()

	// Lock-free atomic state checks
	paused := atomic.LoadUint32(&p.pausedAtomic) == 1
	playing := atomic.LoadUint32(&p.playingAtomic) == 1

	// Adjust state based on our atomic flags
	if paused {
		info.State = StatePaused
	} else if !playing {
		info.State = StateStopped
	}

	// Brief lock only for config values
	p.mu.Lock()
	info.Volume = p.volume
	info.Speed = p.speed
	info.TotalLoops = p.loopCount
	p.mu.Unlock()

	return info
}

// IsLoaded returns true if a track is loaded.
func (p *AudioPlayer) IsLoaded() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.track != nil
}

// State returns the current playback state.
// This provides the actual player state without tick delay.
// It checks both atomic flags and the underlying libvgm state.
func (p *AudioPlayer) State() PlayState {
	paused := atomic.LoadUint32(&p.pausedAtomic) == 1
	playing := atomic.LoadUint32(&p.playingAtomic) == 1

	if paused {
		return StatePaused
	}
	if !playing {
		return StateStopped
	}

	// Check if libvgm reports track ended (even if our flags say playing)
	// This handles the case where track naturally ended but Stop() wasn't called yet
	info := p.vgm.GetPlaybackInfo()
	if info.State == StateStopped {
		return StateStopped
	}

	return StatePlaying
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

// tickLoop sends periodic playback info updates to subscribers.
func (p *AudioPlayer) tickLoop() {
	defer p.tickWg.Done()

	ticker := time.NewTicker(DefaultTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			// Lock-free state check
			if atomic.LoadUint32(&p.playingAtomic) == 0 {
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

	// Cancel tick loop
	p.cancel()

	// Stop playback
	p.stopLocked()

	p.mu.Unlock()

	// Wait for tickLoop goroutine to exit before closing channels
	p.tickWg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Unbind and close audio driver
	if p.audioDriver != nil {
		p.audioDriver.UnbindPlayer()
		p.audioDriver.Stop()
		p.audioDriver.Close()
		p.audioDriver = nil
	}

	// Close subscribers (safe now that tickLoop has exited)
	p.subMu.Lock()
	for ch := range p.subscribers {
		close(ch)
	}
	p.subscribers = nil
	p.subMu.Unlock()

	// Close libvgm player
	if p.vgm != nil {
		p.vgm.Close()
		p.vgm = nil
	}

	// Deinitialize audio system
	DeinitAudioSystem()

	return nil
}

// Ensure AudioPlayer implements Player
var _ Player = (*AudioPlayer)(nil)
