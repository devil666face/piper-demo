package main

/*
#cgo CFLAGS: -I./include
#cgo LDFLAGS: -L. -lpiper -lonnxruntime
#include <stdlib.h>
#include <piper.h>

typedef struct {
    piper_synthesizer *synth;
    piper_synthesize_options opts;
} SynthHandle;

static SynthHandle* SynthCreate(const char *onnx, const char *json, const char *espeak) {
    SynthHandle* handle = (SynthHandle*)malloc(sizeof(SynthHandle));
    handle->synth = piper_create(onnx, json, espeak);
    handle->opts = piper_default_synthesize_options(handle->synth);
    return handle;
}

static void SynthSetLength(SynthHandle* handle, float length) {
    handle->opts.length_scale = length;
}

static int SynthStart(SynthHandle* handle, const char *text) {
    return piper_synthesize_start(handle->synth, text, &handle->opts);
}

static int SynthNext(SynthHandle* handle, piper_audio_chunk *chunk) {
    return piper_synthesize_next(handle->synth, chunk);
}

static void SynthFree(SynthHandle* handle) {
    piper_free(handle->synth);
    free(handle);
}
*/
import "C"
import (
	"embed"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"piper/pkg/fs"
	"unsafe"
)

//go:embed espeak-ng-data
var Data embed.FS

//go:embed onnx
var Onnx embed.FS

const (
	dataDir = "."
	onnxDir = "."
)

func writeWav(filename string, samplesAll []int16) error {
	// WAV header
	const (
		sampleRate    = 22050
		numChannels   = 1
		bitsPerSample = 16
		audioFormat   = 1 // PCM
	)
	dataSize := len(samplesAll) * 2

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	writeLE := func(v any) error {
		return binary.Write(file, binary.LittleEndian, v)
	}

	// RIFF header
	if _, err := file.Write([]byte("RIFF")); err != nil {
		return fmt.Errorf("failed to write RIFF: %w", err)
	}
	if err := writeLE(uint32(36 + dataSize)); err != nil {
		return fmt.Errorf("failed to write chunk size: %w", err)
	}
	if _, err := file.Write([]byte("WAVE")); err != nil {
		return fmt.Errorf("failed to write WAVE: %w", err)
	}
	// fmt chunk
	if _, err := file.Write([]byte("fmt ")); err != nil {
		return fmt.Errorf("failed to write fmt: %w", err)
	}
	if err := writeLE(uint32(16)); err != nil { // Subchunk1Size
		return fmt.Errorf("failed to write fmt size: %w", err)
	}
	if err := writeLE(uint16(audioFormat)); err != nil { // AudioFormat
		return fmt.Errorf("failed to write audio format: %w", err)
	}
	if err := writeLE(uint16(numChannels)); err != nil { // NumChannels
		return fmt.Errorf("failed to write num channels: %w", err)
	}
	if err := writeLE(uint32(sampleRate)); err != nil { // SampleRate
		return fmt.Errorf("failed to write sample rate: %w", err)
	}
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	if err := writeLE(uint32(byteRate)); err != nil { // ByteRate
		return fmt.Errorf("failed to write byte rate: %w", err)
	}
	blockAlign := numChannels * bitsPerSample / 8
	if err := writeLE(uint16(blockAlign)); err != nil { // BlockAlign
		return fmt.Errorf("failed to write block align: %w", err)
	}
	if err := writeLE(uint16(bitsPerSample)); err != nil { // BitsPerSample
		return fmt.Errorf("failed to write bits per sample: %w", err)
	}
	// data chunk
	if _, err := file.Write([]byte("data")); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	if err := writeLE(uint32(dataSize)); err != nil {
		return fmt.Errorf("failed to write data size: %w", err)
	}
	// Write PCM data
	for _, s := range samplesAll {
		if err := writeLE(s); err != nil {
			return fmt.Errorf("failed to write sample: %w", err)
		}
	}
	return nil
}

var (
	in = flag.String("text", "Привет, мир!", "your text for voice")
)

func _main() error {
	flag.Parse()

	if _, err := fs.EmbedToFS(dataDir, Data); err != nil {
		return fmt.Errorf("failed to prepare dir: %w", err)
	}
	if _, err := fs.EmbedToFS(onnxDir, Onnx); err != nil {
		return fmt.Errorf("failed to prepare dir: %w", err)
	}

	var (
		onnx   = C.CString("onnx/ru_RU-dmitri-medium.onnx")
		json   = C.CString("onnx/ru_RU-dmitri-medium.onnx.json")
		espeak = C.CString("espeak-ng-data")
	)
	defer C.free(unsafe.Pointer(onnx))
	defer C.free(unsafe.Pointer(json))
	defer C.free(unsafe.Pointer(espeak))

	var (
		synth = C.SynthCreate(onnx, json, espeak)
		text  = C.CString(*in)
		chunk C.piper_audio_chunk
	)

	defer C.SynthFree(synth)
	defer C.free(unsafe.Pointer(text))

	C.SynthSetLength(synth, C.float(1.0)) // длина речи, можно менять
	C.SynthStart(synth, text)

	var samplesAll []int16

	for {
		status := C.SynthNext(synth, &chunk)
		if status == C.PIPER_DONE {
			break
		}
		samples := unsafe.Slice((*float32)(unsafe.Pointer(chunk.samples)), int(chunk.num_samples))
		for _, s := range samples {
			// Клиппинг и преобразование float32 [-1,1] в int16
			fs := s
			if fs > 1.0 {
				fs = 1.0
			}
			if fs < -1.0 {
				fs = -1.0
			}
			i16 := int16(fs * 32767)
			samplesAll = append(samplesAll, i16)
		}
	}

	if err := writeWav("output.wav", samplesAll); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := _main(); err != nil {
		log.Fatalln(err)
	}
}
