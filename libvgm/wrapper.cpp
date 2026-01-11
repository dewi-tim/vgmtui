/*
 * wrapper.cpp - C wrapper implementation for libvgm's C++ PlayerA class
 *
 * This wraps the PlayerA class to provide a C API for use with CGO.
 */

#include "wrapper.h"

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <vector>
#include <string>
#include <map>

// libvgm headers
#include <stdtype.h>
#include <utils/DataLoader.h>
#include <utils/FileLoader.h>
#include <player/playerbase.hpp>
#include <player/playera.hpp>
#include <player/vgmplayer.hpp>
#include <player/s98player.hpp>
#include <player/droplayer.hpp>
#include <player/gymplayer.hpp>
#include <emu/SoundDevs.h>
#include <emu/SoundEmu.h>
#include <emu/EmuCores.h>

// FourCC constants for player types
#define FCC_VGM  0x56474D00
#define FCC_S98  0x53393800
#define FCC_DRO  0x44524F00
#define FCC_GYM  0x47594D00

// Convert FourCC to string
static std::string FCC2Str(UINT32 fcc) {
    char buf[5];
    buf[0] = (fcc >> 24) & 0xFF;
    buf[1] = (fcc >> 16) & 0xFF;
    buf[2] = (fcc >> 8) & 0xFF;
    buf[3] = fcc & 0xFF;
    buf[4] = '\0';
    // Trim trailing spaces/nulls
    for (int i = 3; i >= 0; i--) {
        if (buf[i] == ' ' || buf[i] == '\0')
            buf[i] = '\0';
        else
            break;
    }
    return std::string(buf);
}

// Internal player structure
struct VgmPlayer {
    PlayerA player;
    DATA_LOADER* dataLoader;

    // Configuration
    uint32_t sampleRate;
    uint32_t loopCount;
    uint32_t fadeSamples;
    uint32_t endSilenceSamples;

    // Cached metadata
    std::map<std::string, std::string> tags;
    std::string formatStr;
    std::vector<std::string> chipNames;
    std::vector<std::string> chipCores;

    // Empty string for returning
    std::string emptyStr;

    VgmPlayer() : dataLoader(nullptr), sampleRate(44100), loopCount(2),
                  fadeSamples(0), endSilenceSamples(0) {}
};

// Helper: Convert milliseconds to samples
static inline uint32_t msToSamples(uint32_t ms, uint32_t sampleRate) {
    return (uint32_t)(((uint64_t)ms * sampleRate + 500) / 1000);
}

/*
 * Lifecycle functions
 */

VgmPlayer* vgm_player_create(void) {
    VgmPlayer* p = new(std::nothrow) VgmPlayer();
    if (!p) return nullptr;

    // Register all player engines
    p->player.RegisterPlayerEngine(new VGMPlayer);
    p->player.RegisterPlayerEngine(new S98Player);
    p->player.RegisterPlayerEngine(new DROPlayer);
    p->player.RegisterPlayerEngine(new GYMPlayer);

    // Set default output settings (44100 Hz, stereo, 16-bit)
    p->player.SetOutputSettings(p->sampleRate, 2, 16, p->sampleRate / 4);

    // Set default fade time (4 seconds)
    p->fadeSamples = msToSamples(4000, p->sampleRate);
    p->player.SetFadeSamples(p->fadeSamples);

    // Set default end silence (1 second)
    p->endSilenceSamples = msToSamples(1000, p->sampleRate);
    p->player.SetEndSilenceSamples(p->endSilenceSamples);

    // Set default loop count
    p->player.SetLoopCount(p->loopCount);

    return p;
}

void vgm_player_destroy(VgmPlayer* p) {
    if (!p) return;

    // Stop playback if active
    p->player.Stop();

    // Unload file
    p->player.UnloadFile();

    // Free data loader
    if (p->dataLoader) {
        DataLoader_Deinit(p->dataLoader);
        p->dataLoader = nullptr;
    }

    // Unregister players
    p->player.UnregisterAllPlayers();

    delete p;
}

/*
 * Configuration functions
 */

void vgm_player_set_sample_rate(VgmPlayer* p, uint32_t rate) {
    if (!p || rate == 0) return;
    p->sampleRate = rate;
    p->player.SetOutputSettings(rate, 2, 16, rate / 4);

    // Recalculate fade and silence samples
    p->fadeSamples = msToSamples(p->fadeSamples * 1000 / p->sampleRate, rate);
    p->endSilenceSamples = msToSamples(p->endSilenceSamples * 1000 / p->sampleRate, rate);
    p->player.SetFadeSamples(p->fadeSamples);
    p->player.SetEndSilenceSamples(p->endSilenceSamples);
}

void vgm_player_set_loop_count(VgmPlayer* p, uint32_t count) {
    if (!p) return;
    p->loopCount = count;
    p->player.SetLoopCount(count);
}

void vgm_player_set_fade_time(VgmPlayer* p, uint32_t ms) {
    if (!p) return;
    p->fadeSamples = msToSamples(ms, p->sampleRate);
    p->player.SetFadeSamples(p->fadeSamples);
}

void vgm_player_set_end_silence(VgmPlayer* p, uint32_t ms) {
    if (!p) return;
    p->endSilenceSamples = msToSamples(ms, p->sampleRate);
    p->player.SetEndSilenceSamples(p->endSilenceSamples);
}

void vgm_player_set_volume(VgmPlayer* p, double vol) {
    if (!p) return;
    // libvgm uses 16.16 fixed point (0x10000 = 1.0)
    INT32 fixedVol = (INT32)(vol * 0x10000);
    p->player.SetMasterVolume(fixedVol);
}

void vgm_player_set_speed(VgmPlayer* p, double speed) {
    if (!p || speed <= 0.0) return;
    p->player.SetPlaybackSpeed(speed);
}

/*
 * File operations
 */

// Helper to extract tags from the player
static void extractTags(VgmPlayer* p) {
    p->tags.clear();

    PlayerBase* player = p->player.GetPlayer();
    if (!player) return;

    const char* const* tagList = player->GetTags();
    if (!tagList) return;

    for (const char* const* t = tagList; *t != nullptr; t += 2) {
        if (t[0] && t[1] && t[1][0] != '\0') {
            p->tags[t[0]] = t[1];
        }
    }
}

// Helper to generate format string
static void generateFormatString(VgmPlayer* p) {
    PlayerBase* player = p->player.GetPlayer();
    if (!player) {
        p->formatStr = "";
        return;
    }

    PLR_SONG_INFO sInf;
    player->GetSongInfo(sInf);

    char verStr[32];
    UINT32 playerType = player->GetPlayerType();

    if (playerType == FCC_VGM) {
        VGMPlayer* vgmplay = dynamic_cast<VGMPlayer*>(player);
        if (vgmplay) {
            const VGM_HEADER* hdr = vgmplay->GetFileHeader();
            snprintf(verStr, sizeof(verStr), "VGM %X.%02X",
                    (hdr->fileVer >> 8) & 0xFF,
                    (hdr->fileVer >> 0) & 0xFF);
        } else {
            snprintf(verStr, sizeof(verStr), "VGM");
        }
    } else if (playerType == FCC_S98) {
        S98Player* s98play = dynamic_cast<S98Player*>(player);
        if (s98play) {
            const S98_HEADER* hdr = s98play->GetFileHeader();
            snprintf(verStr, sizeof(verStr), "S98 v%u", hdr->fileVer);
        } else {
            snprintf(verStr, sizeof(verStr), "S98");
        }
    } else if (playerType == FCC_DRO) {
        DROPlayer* droplay = dynamic_cast<DROPlayer*>(player);
        if (droplay) {
            const DRO_HEADER* hdr = droplay->GetFileHeader();
            snprintf(verStr, sizeof(verStr), "DRO v%u", hdr->verMajor);
        } else {
            snprintf(verStr, sizeof(verStr), "DRO");
        }
    } else if (playerType == FCC_GYM) {
        GYMPlayer* gymplay = dynamic_cast<GYMPlayer*>(player);
        if (gymplay) {
            const GYM_HEADER* hdr = gymplay->GetFileHeader();
            if (!hdr->hasHeader)
                snprintf(verStr, sizeof(verStr), "GYM");
            else if (hdr->uncomprSize == 0)
                snprintf(verStr, sizeof(verStr), "GYMX");
            else
                snprintf(verStr, sizeof(verStr), "GYMX (z)");
        } else {
            snprintf(verStr, sizeof(verStr), "GYM");
        }
    } else {
        snprintf(verStr, sizeof(verStr), "???");
    }

    p->formatStr = verStr;
}

// Helper to enumerate sound chips
static void enumerateChips(VgmPlayer* p) {
    p->chipNames.clear();
    p->chipCores.clear();

    PlayerBase* player = p->player.GetPlayer();
    if (!player) return;

    std::vector<PLR_DEV_INFO> devList;
    player->GetSongDeviceInfo(devList);

    for (size_t i = 0; i < devList.size(); i++) {
        const PLR_DEV_INFO& di = devList[i];

        // Get chip name
        const char* chipName = SndEmu_GetDevName(di.type, 0x01, di.devCfg);
        p->chipNames.push_back(chipName ? chipName : "Unknown");

        // Get core name from FCC
        p->chipCores.push_back(FCC2Str(di.core));
    }
}

int vgm_player_load(VgmPlayer* p, const char* path) {
    if (!p || !path) return VGM_ERR_NULLPTR;

    // Unload any existing file
    vgm_player_unload(p);

    // Create file loader
    p->dataLoader = FileLoader_Init(path);
    if (!p->dataLoader) {
        return VGM_ERR_FILE;
    }

    // Set preload bytes for format detection
    DataLoader_SetPreloadBytes(p->dataLoader, 0x100);

    // Load the file
    UINT8 retVal = DataLoader_Load(p->dataLoader);
    if (retVal) {
        DataLoader_CancelLoading(p->dataLoader);
        DataLoader_Deinit(p->dataLoader);
        p->dataLoader = nullptr;
        return VGM_ERR_FILE;
    }

    // Load into player
    retVal = p->player.LoadFile(p->dataLoader);
    if (retVal) {
        DataLoader_CancelLoading(p->dataLoader);
        DataLoader_Deinit(p->dataLoader);
        p->dataLoader = nullptr;
        return VGM_ERR_FORMAT;
    }

    // Apply loop count (may be modified by VGM header)
    PlayerBase* player = p->player.GetPlayer();
    if (player && player->GetPlayerType() == FCC_VGM) {
        VGMPlayer* vgmplay = dynamic_cast<VGMPlayer*>(player);
        if (vgmplay) {
            p->player.SetLoopCount(vgmplay->GetModifiedLoopCount(p->loopCount));
        }
    }

    // Extract metadata
    extractTags(p);
    generateFormatString(p);

    // Enumerate chips (chip names available after load, core names after start)
    enumerateChips(p);

    return VGM_OK;
}

void vgm_player_unload(VgmPlayer* p) {
    if (!p) return;

    p->player.Stop();
    p->player.UnloadFile();

    if (p->dataLoader) {
        DataLoader_Deinit(p->dataLoader);
        p->dataLoader = nullptr;
    }

    p->tags.clear();
    p->formatStr.clear();
    p->chipNames.clear();
    p->chipCores.clear();
}

/*
 * Playback control
 */

int vgm_player_start(VgmPlayer* p) {
    if (!p || !p->dataLoader) return VGM_ERR_STATE;

    UINT8 ret = p->player.Start();
    if (ret) return VGM_ERR_STATE;

    // Process the VGM initialization block (commands before first wait)
    // This is critical for chips like RF5C164 that need PCM data loaded
    p->player.Render(0, NULL);

    // Enumerate chips after starting (need to start for core info)
    enumerateChips(p);

    return VGM_OK;
}

void vgm_player_stop(VgmPlayer* p) {
    if (!p) return;
    p->player.Stop();
}

void vgm_player_reset(VgmPlayer* p) {
    if (!p) return;
    p->player.Reset();
}

void vgm_player_fade_out(VgmPlayer* p) {
    if (!p) return;
    p->player.FadeOut();
}

void vgm_player_seek(VgmPlayer* p, double seconds) {
    if (!p || seconds < 0.0) return;

    // Convert seconds to samples and seek
    uint32_t samples = (uint32_t)(seconds * p->sampleRate);
    p->player.Seek(PLAYPOS_SAMPLE, samples);
}

/*
 * Audio rendering
 */

uint32_t vgm_player_render(VgmPlayer* p, uint32_t frames, int16_t* buffer) {
    if (!p || !buffer || frames == 0) return 0;

    // Calculate buffer size in bytes (stereo 16-bit)
    uint32_t bufSize = frames * 2 * sizeof(int16_t);

    // Render returns bytes rendered
    uint32_t bytesRendered = p->player.Render(bufSize, buffer);

    // Convert back to frames
    return bytesRendered / (2 * sizeof(int16_t));
}

/*
 * State queries
 */

int vgm_player_is_playing(VgmPlayer* p) {
    if (!p) return 0;
    return (p->player.GetState() & PLAYSTATE_PLAY) ? 1 : 0;
}

int vgm_player_is_fading(VgmPlayer* p) {
    if (!p) return 0;
    return (p->player.GetState() & PLAYSTATE_FADE) ? 1 : 0;
}

int vgm_player_is_finished(VgmPlayer* p) {
    if (!p) return 0;
    return (p->player.GetState() & PLAYSTATE_FIN) ? 1 : 0;
}

double vgm_player_get_position(VgmPlayer* p) {
    if (!p) return 0.0;
    return p->player.GetCurTime(PLAYTIME_LOOP_INCL | PLAYTIME_TIME_FILE);
}

double vgm_player_get_duration(VgmPlayer* p) {
    if (!p) return 0.0;
    return p->player.GetTotalTime(PLAYTIME_LOOP_INCL | PLAYTIME_TIME_FILE | PLAYTIME_WITH_FADE);
}

uint32_t vgm_player_get_current_loop(VgmPlayer* p) {
    if (!p) return 0;
    return p->player.GetCurLoop();
}

int vgm_player_has_loop(VgmPlayer* p) {
    if (!p) return 0;

    PlayerBase* player = p->player.GetPlayer();
    if (!player) return 0;

    return (player->GetLoopTicks() > 0) ? 1 : 0;
}

double vgm_player_get_loop_point(VgmPlayer* p) {
    if (!p) return 0.0;
    return p->player.GetLoopTime();
}

uint32_t vgm_player_get_sample_rate(VgmPlayer* p) {
    if (!p) return 0;
    return p->player.GetSampleRate();
}

/*
 * Metadata functions
 */

static const char* getTag(VgmPlayer* p, const char* tagName) {
    if (!p) return "";

    auto it = p->tags.find(tagName);
    if (it != p->tags.end()) {
        return it->second.c_str();
    }
    return "";
}

const char* vgm_player_get_title(VgmPlayer* p) {
    // Try English first, then Japanese
    const char* title = getTag(p, "TITLE");
    if (title[0] != '\0') return title;
    return getTag(p, "TITLE-JPN");
}

const char* vgm_player_get_game(VgmPlayer* p) {
    const char* game = getTag(p, "GAME");
    if (game[0] != '\0') return game;
    return getTag(p, "GAME-JPN");
}

const char* vgm_player_get_system(VgmPlayer* p) {
    const char* system = getTag(p, "SYSTEM");
    if (system[0] != '\0') return system;
    return getTag(p, "SYSTEM-JPN");
}

const char* vgm_player_get_composer(VgmPlayer* p) {
    const char* artist = getTag(p, "ARTIST");
    if (artist[0] != '\0') return artist;
    return getTag(p, "ARTIST-JPN");
}

const char* vgm_player_get_date(VgmPlayer* p) {
    return getTag(p, "DATE");
}

const char* vgm_player_get_vgm_by(VgmPlayer* p) {
    return getTag(p, "ENCODED_BY");
}

const char* vgm_player_get_notes(VgmPlayer* p) {
    return getTag(p, "COMMENT");
}

const char* vgm_player_get_format(VgmPlayer* p) {
    if (!p) return "";
    return p->formatStr.c_str();
}

/*
 * Sound chip information
 */

uint32_t vgm_player_get_chip_count(VgmPlayer* p) {
    if (!p) return 0;
    return (uint32_t)p->chipNames.size();
}

const char* vgm_player_get_chip_name(VgmPlayer* p, uint32_t index) {
    if (!p || index >= p->chipNames.size()) return "";
    return p->chipNames[index].c_str();
}

const char* vgm_player_get_chip_core(VgmPlayer* p, uint32_t index) {
    if (!p || index >= p->chipCores.size()) return "";
    return p->chipCores[index].c_str();
}

/*
 * =============================================================================
 * Audio Driver Implementation
 * =============================================================================
 */

#include <audio/AudioStream.h>
#include <utils/OSMutex.h>
#include <string.h>

// Audio driver state
struct VgmAudioDriver {
    void* drvData;              // Audio driver instance from AudioDrv_Init
    uint32_t driverID;          // Driver ID used to create this instance
    VgmPlayer* boundPlayer;     // Player bound to this driver
    OS_MUTEX* renderMtx;        // Mutex for thread-safe rendering
    volatile uint8_t paused;    // Pause state flag (read atomically in callback)

    // Audio configuration
    uint32_t sampleRate;
    uint8_t numChannels;
    uint8_t numBitsPerSmpl;
    uint32_t usecPerBuf;
    uint32_t numBuffers;

    VgmAudioDriver() : drvData(nullptr), driverID(0), boundPlayer(nullptr),
                       renderMtx(nullptr), paused(0),
                       sampleRate(44100), numChannels(2), numBitsPerSmpl(16),
                       usecPerBuf(10000), numBuffers(4) {}
};

// Global state
static bool audioSystemInitialized = false;

// FillBuffer callback - called from audio driver's thread
static UINT32 AudioFillBuffer(void* drvStruct, void* userParam, UINT32 bufSize, void* data) {
    VgmAudioDriver* drv = (VgmAudioDriver*)userParam;

    // Early out if paused or no player bound
    if (!drv || drv->paused || !drv->boundPlayer) {
        memset(data, 0, bufSize);
        return bufSize;
    }

    UINT32 renderedBytes = 0;

    // Lock the mutex and render
    if (OSMutex_Lock(drv->renderMtx) == 0) {
        if (drv->boundPlayer) {
            renderedBytes = drv->boundPlayer->player.Render(bufSize, data);
        }
        OSMutex_Unlock(drv->renderMtx);
    }

    // Zero-fill any remaining bytes
    if (renderedBytes < bufSize) {
        memset((uint8_t*)data + renderedBytes, 0, bufSize - renderedBytes);
    }

    return bufSize;
}

/*
 * Audio system lifecycle
 */

int vgm_audio_init(void) {
    if (audioSystemInitialized) {
        return VGM_AUDIO_OK;
    }

    UINT8 ret = Audio_Init();
    if (ret != AERR_OK && ret != AERR_WASDONE) {
        return VGM_AUDIO_ERR_INIT;
    }

    audioSystemInitialized = true;
    return VGM_AUDIO_OK;
}

void vgm_audio_deinit(void) {
    if (audioSystemInitialized) {
        Audio_Deinit();
        audioSystemInitialized = false;
    }
}

uint32_t vgm_audio_get_driver_count(void) {
    if (!audioSystemInitialized) return 0;
    return Audio_GetDriverCount();
}

const char* vgm_audio_get_driver_name(uint32_t drvID) {
    if (!audioSystemInitialized) return "";

    AUDDRV_INFO* drvInfo = nullptr;
    if (Audio_GetDriverInfo(drvID, &drvInfo) != AERR_OK || !drvInfo) {
        return "";
    }
    return drvInfo->drvName ? drvInfo->drvName : "";
}

uint8_t vgm_audio_get_driver_sig(uint32_t drvID) {
    if (!audioSystemInitialized) return 0;

    AUDDRV_INFO* drvInfo = nullptr;
    if (Audio_GetDriverInfo(drvID, &drvInfo) != AERR_OK || !drvInfo) {
        return 0;
    }
    return drvInfo->drvSig;
}

uint8_t vgm_audio_get_driver_type(uint32_t drvID) {
    if (!audioSystemInitialized) return 0;

    AUDDRV_INFO* drvInfo = nullptr;
    if (Audio_GetDriverInfo(drvID, &drvInfo) != AERR_OK || !drvInfo) {
        return 0;
    }
    return drvInfo->drvType;
}

/*
 * Audio driver instance
 */

VgmAudioDriver* vgm_audio_driver_create(uint32_t drvID) {
    if (!audioSystemInitialized) return nullptr;

    VgmAudioDriver* drv = new(std::nothrow) VgmAudioDriver();
    if (!drv) return nullptr;

    // Initialize the audio driver
    UINT8 ret = AudioDrv_Init(drvID, &drv->drvData);
    if (ret != AERR_OK) {
        delete drv;
        return nullptr;
    }

    drv->driverID = drvID;

    // Create render mutex
    ret = OSMutex_Init(&drv->renderMtx, 0);
    if (ret != 0) {
        AudioDrv_Deinit(&drv->drvData);
        delete drv;
        return nullptr;
    }

    return drv;
}

void vgm_audio_driver_destroy(VgmAudioDriver* drv) {
    if (!drv) return;

    // Stop and unbind first
    vgm_audio_driver_stop(drv);
    vgm_audio_driver_unbind_player(drv);

    // Destroy mutex
    if (drv->renderMtx) {
        OSMutex_Deinit(drv->renderMtx);
        drv->renderMtx = nullptr;
    }

    // Deinitialize audio driver
    if (drv->drvData) {
        AudioDrv_Deinit(&drv->drvData);
    }

    delete drv;
}

/*
 * Audio driver configuration
 */

void vgm_audio_driver_set_sample_rate(VgmAudioDriver* drv, uint32_t rate) {
    if (!drv || rate == 0) return;
    drv->sampleRate = rate;
}

void vgm_audio_driver_set_channels(VgmAudioDriver* drv, uint8_t channels) {
    if (!drv || channels == 0) return;
    drv->numChannels = channels;
}

void vgm_audio_driver_set_bits(VgmAudioDriver* drv, uint8_t bits) {
    if (!drv || (bits != 8 && bits != 16)) return;
    drv->numBitsPerSmpl = bits;
}

void vgm_audio_driver_set_buffer_time(VgmAudioDriver* drv, uint32_t usec) {
    if (!drv || usec == 0) return;
    drv->usecPerBuf = usec;
}

void vgm_audio_driver_set_buffer_count(VgmAudioDriver* drv, uint32_t count) {
    if (!drv || count == 0) return;
    drv->numBuffers = count;
}

/*
 * Audio driver control
 */

int vgm_audio_driver_start(VgmAudioDriver* drv, uint32_t deviceID) {
    if (!drv || !drv->drvData) return VGM_AUDIO_ERR_NULLPTR;

    // Get audio options and configure
    AUDIO_OPTS* opts = AudioDrv_GetOptions(drv->drvData);
    if (opts) {
        opts->sampleRate = drv->sampleRate;
        opts->numChannels = drv->numChannels;
        opts->numBitsPerSmpl = drv->numBitsPerSmpl;
        opts->usecPerBuf = drv->usecPerBuf;
        opts->numBuffers = drv->numBuffers;
    }

    // Start the audio device
    UINT8 ret = AudioDrv_Start(drv->drvData, deviceID);
    if (ret != AERR_OK) {
        return VGM_AUDIO_ERR_DRV_START;
    }

    // Set the callback
    ret = AudioDrv_SetCallback(drv->drvData, AudioFillBuffer, drv);
    if (ret != AERR_OK) {
        // Callback failed - some drivers don't support it, but we require it
        AudioDrv_Stop(drv->drvData);
        return VGM_AUDIO_ERR_DRV_START;
    }

    drv->paused = 0;
    return VGM_AUDIO_OK;
}

int vgm_audio_driver_stop(VgmAudioDriver* drv) {
    if (!drv || !drv->drvData) return VGM_AUDIO_ERR_NULLPTR;

    // Clear callback first
    AudioDrv_SetCallback(drv->drvData, nullptr, nullptr);

    // Stop the driver
    UINT8 ret = AudioDrv_Stop(drv->drvData);
    if (ret != AERR_OK) {
        return VGM_AUDIO_ERR_DRV_START;
    }

    return VGM_AUDIO_OK;
}

int vgm_audio_driver_pause(VgmAudioDriver* drv) {
    if (!drv || !drv->drvData) return VGM_AUDIO_ERR_NULLPTR;

    drv->paused = 1;
    AudioDrv_Pause(drv->drvData);
    return VGM_AUDIO_OK;
}

int vgm_audio_driver_resume(VgmAudioDriver* drv) {
    if (!drv || !drv->drvData) return VGM_AUDIO_ERR_NULLPTR;

    drv->paused = 0;
    AudioDrv_Resume(drv->drvData);
    return VGM_AUDIO_OK;
}

uint32_t vgm_audio_driver_get_latency(VgmAudioDriver* drv) {
    if (!drv || !drv->drvData) return 0;
    return AudioDrv_GetLatency(drv->drvData);
}

/*
 * Player binding
 */

int vgm_audio_driver_bind_player(VgmAudioDriver* drv, VgmPlayer* player) {
    if (!drv || !player) return VGM_AUDIO_ERR_NULLPTR;

    OSMutex_Lock(drv->renderMtx);
    drv->boundPlayer = player;
    OSMutex_Unlock(drv->renderMtx);

    return VGM_AUDIO_OK;
}

void vgm_audio_driver_unbind_player(VgmAudioDriver* drv) {
    if (!drv) return;

    OSMutex_Lock(drv->renderMtx);
    drv->boundPlayer = nullptr;
    OSMutex_Unlock(drv->renderMtx);
}

/*
 * Thread-safe player operations
 */

void vgm_audio_safe_seek(VgmAudioDriver* drv, double seconds) {
    if (!drv || !drv->boundPlayer || seconds < 0.0) return;

    OSMutex_Lock(drv->renderMtx);
    uint32_t samples = (uint32_t)(seconds * drv->boundPlayer->sampleRate);
    drv->boundPlayer->player.Seek(PLAYPOS_SAMPLE, samples);
    OSMutex_Unlock(drv->renderMtx);
}

void vgm_audio_safe_reset(VgmAudioDriver* drv) {
    if (!drv || !drv->boundPlayer) return;

    OSMutex_Lock(drv->renderMtx);
    drv->boundPlayer->player.Reset();
    OSMutex_Unlock(drv->renderMtx);
}

void vgm_audio_safe_fade_out(VgmAudioDriver* drv) {
    if (!drv || !drv->boundPlayer) return;

    OSMutex_Lock(drv->renderMtx);
    drv->boundPlayer->player.FadeOut();
    OSMutex_Unlock(drv->renderMtx);
}
