/*
 * wrapper.h - C wrapper for libvgm's C++ PlayerA class
 *
 * This provides a C API for use with CGO bindings in Go.
 */

#ifndef VGM_WRAPPER_H
#define VGM_WRAPPER_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Opaque player handle */
typedef struct VgmPlayer VgmPlayer;

/* Error codes */
#define VGM_OK              0
#define VGM_ERR_NULLPTR     1
#define VGM_ERR_FILE        2
#define VGM_ERR_FORMAT      3
#define VGM_ERR_MEMORY      4
#define VGM_ERR_STATE       5

/* Playback state flags */
#define VGM_STATE_STOPPED   0x00
#define VGM_STATE_PLAYING   0x01
#define VGM_STATE_PAUSED    0x04
#define VGM_STATE_FADING    0x10
#define VGM_STATE_FINISHED  0x20

/*
 * Lifecycle functions
 */

/* Create a new player instance. Returns NULL on failure. */
VgmPlayer* vgm_player_create(void);

/* Destroy a player instance and free all resources. */
void vgm_player_destroy(VgmPlayer* p);

/*
 * Configuration functions (call before loading a file)
 */

/* Set the output sample rate in Hz (default: 44100) */
void vgm_player_set_sample_rate(VgmPlayer* p, uint32_t rate);

/* Set the number of loops to play (0 = infinite, default: 2) */
void vgm_player_set_loop_count(VgmPlayer* p, uint32_t count);

/* Set the fade-out time in milliseconds (default: 4000) */
void vgm_player_set_fade_time(VgmPlayer* p, uint32_t ms);

/* Set the end silence time in milliseconds (default: 1000) */
void vgm_player_set_end_silence(VgmPlayer* p, uint32_t ms);

/* Set master volume (0.0 = silent, 1.0 = normal, >1.0 = amplified) */
void vgm_player_set_volume(VgmPlayer* p, double vol);

/* Set playback speed (1.0 = normal, 0.5 = half, 2.0 = double) */
void vgm_player_set_speed(VgmPlayer* p, double speed);

/*
 * File operations
 */

/* Load a VGM/VGZ/S98/DRO/GYM file. Returns 0 on success. */
int vgm_player_load(VgmPlayer* p, const char* path);

/* Unload the current file and reset state. */
void vgm_player_unload(VgmPlayer* p);

/*
 * Playback control
 */

/* Start or resume playback. Returns 0 on success. */
int vgm_player_start(VgmPlayer* p);

/* Stop playback. */
void vgm_player_stop(VgmPlayer* p);

/* Reset playback to the beginning. */
void vgm_player_reset(VgmPlayer* p);

/* Trigger fade-out sequence. */
void vgm_player_fade_out(VgmPlayer* p);

/* Seek to a position in seconds. */
void vgm_player_seek(VgmPlayer* p, double seconds);

/*
 * Audio rendering
 */

/*
 * Render audio samples to a buffer.
 * buffer: pointer to int16_t array for stereo samples (L, R, L, R, ...)
 * frames: number of stereo frames to render
 * Returns: number of frames actually rendered
 */
uint32_t vgm_player_render(VgmPlayer* p, uint32_t frames, int16_t* buffer);

/*
 * State queries
 */

/* Check if playback is active (not stopped/finished). Returns 1 if playing. */
int vgm_player_is_playing(VgmPlayer* p);

/* Check if currently fading out. Returns 1 if fading. */
int vgm_player_is_fading(VgmPlayer* p);

/* Check if playback has finished. Returns 1 if finished. */
int vgm_player_is_finished(VgmPlayer* p);

/* Get current playback position in seconds. */
double vgm_player_get_position(VgmPlayer* p);

/* Get total duration in seconds (including configured loops). */
double vgm_player_get_duration(VgmPlayer* p);

/* Get the current loop number (0 = first play, 1 = first loop, etc.). */
uint32_t vgm_player_get_current_loop(VgmPlayer* p);

/* Check if the file has a loop point. Returns 1 if it loops. */
int vgm_player_has_loop(VgmPlayer* p);

/* Get the loop point position in seconds. Returns 0 if no loop. */
double vgm_player_get_loop_point(VgmPlayer* p);

/* Get the configured sample rate. */
uint32_t vgm_player_get_sample_rate(VgmPlayer* p);

/*
 * Metadata functions
 * Note: All returned strings are owned by the player and valid until
 * the file is unloaded. Returns empty string "" if tag is not present.
 */

/* Get track title (GD3: TITLE) */
const char* vgm_player_get_title(VgmPlayer* p);

/* Get game/album name (GD3: GAME) */
const char* vgm_player_get_game(VgmPlayer* p);

/* Get system/platform name (GD3: SYSTEM) */
const char* vgm_player_get_system(VgmPlayer* p);

/* Get composer/artist name (GD3: ARTIST) */
const char* vgm_player_get_composer(VgmPlayer* p);

/* Get release date (GD3: DATE) */
const char* vgm_player_get_date(VgmPlayer* p);

/* Get VGM author (GD3: ENCODED_BY) */
const char* vgm_player_get_vgm_by(VgmPlayer* p);

/* Get notes/comments (GD3: COMMENT) */
const char* vgm_player_get_notes(VgmPlayer* p);

/* Get file format string (e.g., "VGM 1.71", "S98 v3") */
const char* vgm_player_get_format(VgmPlayer* p);

/*
 * Sound chip information
 */

/* Get number of sound chips used in the current file. */
uint32_t vgm_player_get_chip_count(VgmPlayer* p);

/* Get the name of a sound chip by index. Returns "" if index is invalid. */
const char* vgm_player_get_chip_name(VgmPlayer* p, uint32_t index);

/* Get the emulation core name for a chip by index. Returns "" if invalid. */
const char* vgm_player_get_chip_core(VgmPlayer* p, uint32_t index);

#ifdef __cplusplus
}
#endif

#endif /* VGM_WRAPPER_H */
