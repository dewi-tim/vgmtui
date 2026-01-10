package player

/*
#cgo CFLAGS: -I${SRCDIR}/../../libvgm
#cgo LDFLAGS: -L${SRCDIR}/../../libvgm/build -L${SRCDIR}/../../libvgm/build/bin -lvgm_wrapper -lvgm-audio -lvgm-player -lvgm-emu -lvgm-utils -lz -lstdc++ -lm -lpulse-simple -lpulse -lasound -lao -lpthread

#include "wrapper.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"sync"
	"time"
	"unsafe"
)

// Error codes returned by libvgm
var (
	ErrNullPointer = errors.New("libvgm: null pointer")
	ErrFileOpen    = errors.New("libvgm: failed to open file")
	ErrFileFormat  = errors.New("libvgm: unsupported file format")
	ErrMemory      = errors.New("libvgm: memory allocation failed")
	ErrState       = errors.New("libvgm: invalid state")
)

// codeToError converts a C error code to a Go error.
func codeToError(code C.int) error {
	switch code {
	case C.VGM_OK:
		return nil
	case C.VGM_ERR_NULLPTR:
		return ErrNullPointer
	case C.VGM_ERR_FILE:
		return ErrFileOpen
	case C.VGM_ERR_FORMAT:
		return ErrFileFormat
	case C.VGM_ERR_MEMORY:
		return ErrMemory
	case C.VGM_ERR_STATE:
		return ErrState
	default:
		return errors.New("libvgm: unknown error")
	}
}

// LibvgmPlayer wraps the libvgm C player.
type LibvgmPlayer struct {
	handle *C.VgmPlayer
	mu     sync.Mutex
}

// NewLibvgmPlayer creates a new libvgm player instance.
func NewLibvgmPlayer() (*LibvgmPlayer, error) {
	handle := C.vgm_player_create()
	if handle == nil {
		return nil, ErrMemory
	}
	return &LibvgmPlayer{handle: handle}, nil
}

// Close destroys the player and frees all resources.
func (p *LibvgmPlayer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_destroy(p.handle)
		p.handle = nil
	}
}

// SetSampleRate sets the output sample rate in Hz.
func (p *LibvgmPlayer) SetSampleRate(rate uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_set_sample_rate(p.handle, C.uint32_t(rate))
	}
}

// SetLoopCount sets the number of loops to play (0 = infinite).
func (p *LibvgmPlayer) SetLoopCount(count uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_set_loop_count(p.handle, C.uint32_t(count))
	}
}

// SetFadeTime sets the fade-out time in milliseconds.
func (p *LibvgmPlayer) SetFadeTime(ms uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_set_fade_time(p.handle, C.uint32_t(ms))
	}
}

// SetEndSilence sets the end silence time in milliseconds.
func (p *LibvgmPlayer) SetEndSilence(ms uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_set_end_silence(p.handle, C.uint32_t(ms))
	}
}

// SetVolume sets the master volume (0.0 = silent, 1.0 = normal).
func (p *LibvgmPlayer) SetVolume(vol float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_set_volume(p.handle, C.double(vol))
	}
}

// SetSpeed sets the playback speed (1.0 = normal).
func (p *LibvgmPlayer) SetSpeed(speed float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_set_speed(p.handle, C.double(speed))
	}
}

// Load loads a VGM/VGZ/S98/DRO/GYM file.
func (p *LibvgmPlayer) Load(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ErrNullPointer
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	ret := C.vgm_player_load(p.handle, cpath)
	return codeToError(ret)
}

// Unload unloads the current file.
func (p *LibvgmPlayer) Unload() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_unload(p.handle)
	}
}

// Start starts or resumes playback.
func (p *LibvgmPlayer) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ErrNullPointer
	}

	ret := C.vgm_player_start(p.handle)
	return codeToError(ret)
}

// Stop stops playback.
// NOTE: Caller must hold renderMu from AudioPlayer to ensure thread safety.
func (p *LibvgmPlayer) Stop() {
	if p.handle != nil {
		C.vgm_player_stop(p.handle)
	}
}

// Reset resets playback to the beginning.
// NOTE: Caller must hold renderMu from AudioPlayer to ensure thread safety.
func (p *LibvgmPlayer) Reset() {
	if p.handle != nil {
		C.vgm_player_reset(p.handle)
	}
}

// FadeOut triggers the fade-out sequence.
// NOTE: Caller must hold renderMu from AudioPlayer to ensure thread safety.
func (p *LibvgmPlayer) FadeOut() {
	if p.handle != nil {
		C.vgm_player_fade_out(p.handle)
	}
}

// Seek seeks to a position in the track.
// NOTE: Caller must hold renderMu from AudioPlayer to ensure thread safety.
func (p *LibvgmPlayer) Seek(pos time.Duration) {
	if p.handle != nil {
		seconds := pos.Seconds()
		C.vgm_player_seek(p.handle, C.double(seconds))
	}
}

// Render renders audio samples to a buffer.
// Returns the number of stereo frames actually rendered.
// NOTE: This method is intentionally NOT locked - the caller (AudioPlayer.Read)
// must hold renderMu to ensure thread safety. This avoids nested locking which
// causes audio stuttering due to lock contention with UI/tick operations.
func (p *LibvgmPlayer) Render(frames uint32, buffer []int16) uint32 {
	// No lock here - caller holds renderMu
	if p.handle == nil || len(buffer) < int(frames*2) {
		return 0
	}

	rendered := C.vgm_player_render(p.handle, C.uint32_t(frames), (*C.int16_t)(unsafe.Pointer(&buffer[0])))
	return uint32(rendered)
}

// RenderDirect renders audio samples directly to a buffer without any Go-side processing.
// This is the fastest path - renders directly to the output buffer like vgmplay's FillBuffer.
// NOTE: Caller must hold renderMu to ensure thread safety.
func (p *LibvgmPlayer) RenderDirect(frames uint32, buffer []int16) uint32 {
	if p.handle == nil || len(buffer) < int(frames*2) {
		return 0
	}
	return uint32(C.vgm_player_render(p.handle, C.uint32_t(frames), (*C.int16_t)(unsafe.Pointer(&buffer[0]))))
}

// IsPlaying returns true if playback is active.
// This is a lock-free query - libvgm's state is internally consistent.
func (p *LibvgmPlayer) IsPlaying() bool {
	if p.handle == nil {
		return false
	}
	return C.vgm_player_is_playing(p.handle) != 0
}

// IsFading returns true if fade-out is in progress.
// This is a lock-free query - libvgm's state is internally consistent.
func (p *LibvgmPlayer) IsFading() bool {
	if p.handle == nil {
		return false
	}
	return C.vgm_player_is_fading(p.handle) != 0
}

// IsFinished returns true if playback has finished.
// This is a lock-free query - libvgm's state is internally consistent.
func (p *LibvgmPlayer) IsFinished() bool {
	if p.handle == nil {
		return false
	}
	return C.vgm_player_is_finished(p.handle) != 0
}

// Position returns the current playback position.
// This is a lock-free query - libvgm's state is internally consistent.
func (p *LibvgmPlayer) Position() time.Duration {
	if p.handle == nil {
		return 0
	}
	seconds := float64(C.vgm_player_get_position(p.handle))
	return time.Duration(seconds * float64(time.Second))
}

// Duration returns the total duration including loops.
// This is a lock-free query - libvgm's state is internally consistent.
func (p *LibvgmPlayer) Duration() time.Duration {
	if p.handle == nil {
		return 0
	}
	seconds := float64(C.vgm_player_get_duration(p.handle))
	return time.Duration(seconds * float64(time.Second))
}

// CurrentLoop returns the current loop number.
// This is a lock-free query - libvgm's state is internally consistent.
func (p *LibvgmPlayer) CurrentLoop() uint32 {
	if p.handle == nil {
		return 0
	}
	return uint32(C.vgm_player_get_current_loop(p.handle))
}

// HasLoop returns true if the track has a loop point.
// This is a lock-free query - libvgm's state is internally consistent.
func (p *LibvgmPlayer) HasLoop() bool {
	if p.handle == nil {
		return false
	}
	return C.vgm_player_has_loop(p.handle) != 0
}

// LoopPoint returns the loop point position.
func (p *LibvgmPlayer) LoopPoint() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return 0
	}
	seconds := float64(C.vgm_player_get_loop_point(p.handle))
	return time.Duration(seconds * float64(time.Second))
}

// SampleRate returns the configured sample rate.
func (p *LibvgmPlayer) SampleRate() uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return 0
	}
	return uint32(C.vgm_player_get_sample_rate(p.handle))
}

// Title returns the track title.
func (p *LibvgmPlayer) Title() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_title(p.handle))
}

// Game returns the game/album name.
func (p *LibvgmPlayer) Game() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_game(p.handle))
}

// System returns the system/platform name.
func (p *LibvgmPlayer) System() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_system(p.handle))
}

// Composer returns the composer/artist name.
func (p *LibvgmPlayer) Composer() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_composer(p.handle))
}

// Date returns the release date.
func (p *LibvgmPlayer) Date() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_date(p.handle))
}

// VGMBy returns the VGM author.
func (p *LibvgmPlayer) VGMBy() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_vgm_by(p.handle))
}

// Notes returns the notes/comments.
func (p *LibvgmPlayer) Notes() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_notes(p.handle))
}

// Format returns the file format string.
func (p *LibvgmPlayer) Format() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_format(p.handle))
}

// ChipCount returns the number of sound chips.
func (p *LibvgmPlayer) ChipCount() uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return 0
	}
	return uint32(C.vgm_player_get_chip_count(p.handle))
}

// ChipName returns the name of a chip by index.
func (p *LibvgmPlayer) ChipName(index uint32) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_chip_name(p.handle, C.uint32_t(index)))
}

// ChipCore returns the emulation core name for a chip.
func (p *LibvgmPlayer) ChipCore(index uint32) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return ""
	}
	return C.GoString(C.vgm_player_get_chip_core(p.handle, C.uint32_t(index)))
}

// GetTrack returns a Track struct with all metadata.
func (p *LibvgmPlayer) GetTrack(path string) Track {
	p.mu.Lock()
	defer p.mu.Unlock()

	track := Track{Path: path}

	if p.handle == nil {
		return track
	}

	track.Title = C.GoString(C.vgm_player_get_title(p.handle))
	track.Game = C.GoString(C.vgm_player_get_game(p.handle))
	track.System = C.GoString(C.vgm_player_get_system(p.handle))
	track.Composer = C.GoString(C.vgm_player_get_composer(p.handle))
	track.Date = C.GoString(C.vgm_player_get_date(p.handle))
	track.VGMBy = C.GoString(C.vgm_player_get_vgm_by(p.handle))
	track.Notes = C.GoString(C.vgm_player_get_notes(p.handle))
	track.Format = C.GoString(C.vgm_player_get_format(p.handle))

	// Duration
	seconds := float64(C.vgm_player_get_duration(p.handle))
	track.Duration = time.Duration(seconds * float64(time.Second))

	// Loop info
	track.HasLoop = C.vgm_player_has_loop(p.handle) != 0
	if track.HasLoop {
		loopSeconds := float64(C.vgm_player_get_loop_point(p.handle))
		track.LoopPoint = time.Duration(loopSeconds * float64(time.Second))
	}

	// Chips
	chipCount := uint32(C.vgm_player_get_chip_count(p.handle))
	track.Chips = make([]ChipInfo, chipCount)
	for i := uint32(0); i < chipCount; i++ {
		track.Chips[i] = ChipInfo{
			Name: C.GoString(C.vgm_player_get_chip_name(p.handle, C.uint32_t(i))),
			Core: C.GoString(C.vgm_player_get_chip_core(p.handle, C.uint32_t(i))),
		}
	}

	return track
}

// GetPlaybackInfo returns current playback information.
// This is a lock-free query - libvgm's state is internally consistent.
// This method is called frequently from the tick loop for UI updates.
func (p *LibvgmPlayer) GetPlaybackInfo() PlaybackInfo {
	info := PlaybackInfo{
		State:   StateStopped,
		Volume:  1.0,
		Speed:   1.0,
		HasLoop: false,
	}

	if p.handle == nil {
		return info
	}

	// Determine state - these are all lock-free queries
	if C.vgm_player_is_finished(p.handle) != 0 {
		info.State = StateStopped
	} else if C.vgm_player_is_fading(p.handle) != 0 {
		info.State = StateFading
	} else if C.vgm_player_is_playing(p.handle) != 0 {
		info.State = StatePlaying
	}

	// Position
	posSeconds := float64(C.vgm_player_get_position(p.handle))
	info.Position = time.Duration(posSeconds * float64(time.Second))

	// Duration
	durSeconds := float64(C.vgm_player_get_duration(p.handle))
	info.Duration = time.Duration(durSeconds * float64(time.Second))

	// Loop info
	info.CurrentLoop = int(C.vgm_player_get_current_loop(p.handle))
	info.HasLoop = C.vgm_player_has_loop(p.handle) != 0

	return info
}

// ReadTrackMetadata reads track metadata from a file without affecting any
// existing player state. This creates a temporary player instance just for
// reading metadata, so it can be used while playback is active.
func ReadTrackMetadata(path string) (Track, error) {
	track := Track{Path: path}

	// Create a temporary player
	handle := C.vgm_player_create()
	if handle == nil {
		return track, ErrMemory
	}
	defer C.vgm_player_destroy(handle)

	// Load the file
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	if err := codeToError(C.vgm_player_load(handle, cpath)); err != nil {
		return track, err
	}

	// Read metadata
	track.Title = C.GoString(C.vgm_player_get_title(handle))
	track.Game = C.GoString(C.vgm_player_get_game(handle))
	track.System = C.GoString(C.vgm_player_get_system(handle))
	track.Composer = C.GoString(C.vgm_player_get_composer(handle))
	track.Date = C.GoString(C.vgm_player_get_date(handle))
	track.VGMBy = C.GoString(C.vgm_player_get_vgm_by(handle))
	track.Notes = C.GoString(C.vgm_player_get_notes(handle))
	track.Format = C.GoString(C.vgm_player_get_format(handle))

	// Duration
	seconds := float64(C.vgm_player_get_duration(handle))
	track.Duration = time.Duration(seconds * float64(time.Second))

	// Loop info
	track.HasLoop = C.vgm_player_has_loop(handle) != 0
	if track.HasLoop {
		loopSeconds := float64(C.vgm_player_get_loop_point(handle))
		track.LoopPoint = time.Duration(loopSeconds * float64(time.Second))
	}

	// Chip info (now available after load)
	chipCount := uint32(C.vgm_player_get_chip_count(handle))
	track.Chips = make([]ChipInfo, chipCount)
	for i := uint32(0); i < chipCount; i++ {
		track.Chips[i] = ChipInfo{
			Name: C.GoString(C.vgm_player_get_chip_name(handle, C.uint32_t(i))),
			Core: C.GoString(C.vgm_player_get_chip_core(handle, C.uint32_t(i))),
		}
	}

	return track, nil
}

// =============================================================================
// Audio Driver API
// =============================================================================

// Audio driver error codes
var (
	ErrAudioInit      = errors.New("audio: failed to initialize audio system")
	ErrAudioNoDrivers = errors.New("audio: no audio drivers available")
	ErrAudioDrvCreate = errors.New("audio: failed to create audio driver")
	ErrAudioDrvStart  = errors.New("audio: failed to start audio driver")
	ErrAudioBind      = errors.New("audio: failed to bind player")
)

// Driver type constants
const (
	AudioDriverTypeOut  = 0x01 // Stream to speakers
	AudioDriverTypeDisk = 0x02 // Write to disk
)

// Driver signature constants
const (
	AudioDriverSigALSA  = 0x22 // ALSA
	AudioDriverSigPulse = 0x23 // PulseAudio
)

// AudioDriverInfo contains information about an available audio driver.
type AudioDriverInfo struct {
	ID        uint32
	Name      string
	Signature uint8
	Type      uint8
}

// AudioDriver wraps libvgm's audio driver for direct audio output.
type AudioDriver struct {
	handle *C.VgmAudioDriver
	mu     sync.Mutex
}

// audioCodeToError converts audio error codes to Go errors.
func audioCodeToError(code C.int) error {
	switch code {
	case C.VGM_AUDIO_OK:
		return nil
	case C.VGM_AUDIO_ERR_INIT:
		return ErrAudioInit
	case C.VGM_AUDIO_ERR_NO_DRIVERS:
		return ErrAudioNoDrivers
	case C.VGM_AUDIO_ERR_DRV_CREATE:
		return ErrAudioDrvCreate
	case C.VGM_AUDIO_ERR_DRV_START:
		return ErrAudioDrvStart
	case C.VGM_AUDIO_ERR_BIND:
		return ErrAudioBind
	case C.VGM_AUDIO_ERR_NULLPTR:
		return ErrNullPointer
	default:
		return errors.New("audio: unknown error")
	}
}

// InitAudioSystem initializes the libvgm audio subsystem.
// Must be called before using any audio driver functions.
func InitAudioSystem() error {
	ret := C.vgm_audio_init()
	return audioCodeToError(ret)
}

// DeinitAudioSystem shuts down the audio subsystem.
func DeinitAudioSystem() {
	C.vgm_audio_deinit()
}

// GetAudioDrivers returns a list of available audio drivers.
func GetAudioDrivers() []AudioDriverInfo {
	count := uint32(C.vgm_audio_get_driver_count())
	drivers := make([]AudioDriverInfo, 0, count)

	for i := uint32(0); i < count; i++ {
		name := C.GoString(C.vgm_audio_get_driver_name(C.uint32_t(i)))
		sig := uint8(C.vgm_audio_get_driver_sig(C.uint32_t(i)))
		typ := uint8(C.vgm_audio_get_driver_type(C.uint32_t(i)))

		// Only include output drivers (not disk writers)
		if typ == AudioDriverTypeOut {
			drivers = append(drivers, AudioDriverInfo{
				ID:        i,
				Name:      name,
				Signature: sig,
				Type:      typ,
			})
		}
	}
	return drivers
}

// NewAudioDriver creates a new audio driver instance.
func NewAudioDriver(driverID uint32) (*AudioDriver, error) {
	handle := C.vgm_audio_driver_create(C.uint32_t(driverID))
	if handle == nil {
		return nil, ErrAudioDrvCreate
	}
	return &AudioDriver{handle: handle}, nil
}

// Close destroys the audio driver and frees all resources.
func (d *AudioDriver) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_driver_destroy(d.handle)
		d.handle = nil
	}
}

// SetSampleRate sets the output sample rate in Hz.
func (d *AudioDriver) SetSampleRate(rate uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_driver_set_sample_rate(d.handle, C.uint32_t(rate))
	}
}

// SetChannels sets the number of output channels.
func (d *AudioDriver) SetChannels(channels uint8) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_driver_set_channels(d.handle, C.uint8_t(channels))
	}
}

// SetBits sets the bits per sample.
func (d *AudioDriver) SetBits(bits uint8) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_driver_set_bits(d.handle, C.uint8_t(bits))
	}
}

// SetBufferTime sets the buffer time in microseconds.
func (d *AudioDriver) SetBufferTime(usec uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_driver_set_buffer_time(d.handle, C.uint32_t(usec))
	}
}

// SetBufferCount sets the number of buffers.
func (d *AudioDriver) SetBufferCount(count uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_driver_set_buffer_count(d.handle, C.uint32_t(count))
	}
}

// Start starts the audio driver with the specified device.
// Use deviceID 0 for the default device.
func (d *AudioDriver) Start(deviceID uint32) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle == nil {
		return ErrNullPointer
	}

	ret := C.vgm_audio_driver_start(d.handle, C.uint32_t(deviceID))
	return audioCodeToError(ret)
}

// Stop stops the audio driver.
func (d *AudioDriver) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle == nil {
		return ErrNullPointer
	}

	ret := C.vgm_audio_driver_stop(d.handle)
	return audioCodeToError(ret)
}

// Pause pauses audio output.
func (d *AudioDriver) Pause() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle == nil {
		return ErrNullPointer
	}

	ret := C.vgm_audio_driver_pause(d.handle)
	return audioCodeToError(ret)
}

// Resume resumes audio output.
func (d *AudioDriver) Resume() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle == nil {
		return ErrNullPointer
	}

	ret := C.vgm_audio_driver_resume(d.handle)
	return audioCodeToError(ret)
}

// GetLatency returns the current latency in milliseconds.
func (d *AudioDriver) GetLatency() uint32 {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle == nil {
		return 0
	}

	return uint32(C.vgm_audio_driver_get_latency(d.handle))
}

// BindPlayer binds a player to the audio driver.
// The driver's internal callback will render audio from the player.
func (d *AudioDriver) BindPlayer(player *LibvgmPlayer) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle == nil || player == nil || player.handle == nil {
		return ErrNullPointer
	}

	ret := C.vgm_audio_driver_bind_player(d.handle, player.handle)
	return audioCodeToError(ret)
}

// UnbindPlayer unbinds the player from the audio driver.
func (d *AudioDriver) UnbindPlayer() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_driver_unbind_player(d.handle)
	}
}

// SafeSeek seeks to a position (thread-safe, acquires render mutex).
func (d *AudioDriver) SafeSeek(pos time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		seconds := pos.Seconds()
		C.vgm_audio_safe_seek(d.handle, C.double(seconds))
	}
}

// SafeReset resets playback (thread-safe, acquires render mutex).
func (d *AudioDriver) SafeReset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_safe_reset(d.handle)
	}
}

// SafeFadeOut triggers fade-out (thread-safe, acquires render mutex).
func (d *AudioDriver) SafeFadeOut() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		C.vgm_audio_safe_fade_out(d.handle)
	}
}
