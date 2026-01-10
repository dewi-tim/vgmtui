.PHONY: all libvgm build clean test install

GO := /usr/local/go/bin/go
LIBVGM_SRC := $(abspath libvgm/libvgm)
BUILD_DIR := libvgm/build

# Audio driver configuration
AUDIO_CFLAGS := -DAUDDRV_PULSE -DAUDDRV_ALSA
AUDIO_LIBS := -lpulse-simple -lpulse -lasound -lao -lpthread

all: libvgm build

# Build libvgm static libraries and wrapper (now including audio)
libvgm: $(BUILD_DIR)/libvgm_wrapper.a

$(BUILD_DIR)/libvgm_wrapper.a: libvgm/wrapper.cpp libvgm/wrapper.h
	mkdir -p $(BUILD_DIR)
	cd $(BUILD_DIR) && cmake $(LIBVGM_SRC) \
		-DCMAKE_BUILD_TYPE=Release \
		-DBUILD_LIBAUDIO=ON \
		-DAUDDRV_LIBAO=OFF \
		-DBUILD_TESTS=OFF \
		-DBUILD_PLAYER=OFF \
		-DBUILD_VGM2WAV=OFF \
		-DLIBRARY_TYPE=STATIC
	cd $(BUILD_DIR) && cmake --build . --parallel \
		--target vgm-audio --target vgm-player --target vgm-emu --target vgm-utils
	$(CXX) -c libvgm/wrapper.cpp -o $(BUILD_DIR)/wrapper.o \
		-I $(BUILD_DIR) -I $(LIBVGM_SRC) -I $(LIBVGM_SRC)/player \
		-I $(LIBVGM_SRC)/audio -I $(LIBVGM_SRC)/utils \
		$(AUDIO_CFLAGS) \
		-std=c++11
	ar rcs $(BUILD_DIR)/libvgm_wrapper.a $(BUILD_DIR)/wrapper.o

# Build Go binary
build: libvgm
	CGO_ENABLED=1 \
	CGO_CFLAGS="-I$(abspath libvgm) -I$(abspath $(BUILD_DIR)) -I$(LIBVGM_SRC) $(AUDIO_CFLAGS)" \
	CGO_LDFLAGS="-L$(abspath $(BUILD_DIR)) -L$(abspath $(BUILD_DIR))/bin -lvgm_wrapper -lvgm-audio -lvgm-player -lvgm-emu -lvgm-utils -lz -lstdc++ -lm $(AUDIO_LIBS)" \
	$(GO) build -o vgmtui ./cmd/vgmtui

# Run tests
test:
	$(GO) test ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) vgmtui

# Install binary
install: build
	install -Dm755 vgmtui $(DESTDIR)/usr/local/bin/vgmtui
