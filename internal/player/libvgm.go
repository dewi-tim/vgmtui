package player

/*
#cgo CFLAGS: -I${SRCDIR}/../../libvgm
#cgo LDFLAGS: -L${SRCDIR}/../../libvgm/build -lvgm_wrapper -lvgm-player -lvgm-emu -lvgm-utils -lz -lstdc++ -lm

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
func (p *LibvgmPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_stop(p.handle)
	}
}

// Reset resets playback to the beginning.
func (p *LibvgmPlayer) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_reset(p.handle)
	}
}

// FadeOut triggers the fade-out sequence.
func (p *LibvgmPlayer) FadeOut() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		C.vgm_player_fade_out(p.handle)
	}
}

// Seek seeks to a position in the track.
func (p *LibvgmPlayer) Seek(pos time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle != nil {
		seconds := pos.Seconds()
		C.vgm_player_seek(p.handle, C.double(seconds))
	}
}

// Render renders audio samples to a buffer.
// Returns the number of stereo frames actually rendered.
func (p *LibvgmPlayer) Render(frames uint32, buffer []int16) uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil || len(buffer) < int(frames*2) {
		return 0
	}

	rendered := C.vgm_player_render(p.handle, C.uint32_t(frames), (*C.int16_t)(unsafe.Pointer(&buffer[0])))
	return uint32(rendered)
}

// IsPlaying returns true if playback is active.
func (p *LibvgmPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return false
	}
	return C.vgm_player_is_playing(p.handle) != 0
}

// IsFading returns true if fade-out is in progress.
func (p *LibvgmPlayer) IsFading() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return false
	}
	return C.vgm_player_is_fading(p.handle) != 0
}

// IsFinished returns true if playback has finished.
func (p *LibvgmPlayer) IsFinished() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return false
	}
	return C.vgm_player_is_finished(p.handle) != 0
}

// Position returns the current playback position.
func (p *LibvgmPlayer) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return 0
	}
	seconds := float64(C.vgm_player_get_position(p.handle))
	return time.Duration(seconds * float64(time.Second))
}

// Duration returns the total duration including loops.
func (p *LibvgmPlayer) Duration() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return 0
	}
	seconds := float64(C.vgm_player_get_duration(p.handle))
	return time.Duration(seconds * float64(time.Second))
}

// CurrentLoop returns the current loop number.
func (p *LibvgmPlayer) CurrentLoop() uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return 0
	}
	return uint32(C.vgm_player_get_current_loop(p.handle))
}

// HasLoop returns true if the track has a loop point.
func (p *LibvgmPlayer) HasLoop() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

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
func (p *LibvgmPlayer) GetPlaybackInfo() PlaybackInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	info := PlaybackInfo{
		State:   StateStopped,
		Volume:  1.0,
		Speed:   1.0,
		HasLoop: false,
	}

	if p.handle == nil {
		return info
	}

	// Determine state
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
