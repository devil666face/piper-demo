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
	"fmt"
	"log"
	"os"
	"piper/pkg/fs"
	"unsafe"

	_ "embed"
)

//go:embed espeak-ng-data
var Data embed.FS

//go:embed onnx
var Onnx embed.FS

const (
	dataDir = "."
	onnxDir = "."
)

func _main() error {
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
		text  = C.CString("Привет мир")
		chunk C.piper_audio_chunk
	)

	defer C.SynthFree(synth)
	defer C.free(unsafe.Pointer(text))

	C.SynthSetLength(synth, C.float(1.0)) // длина речи, можно менять
	C.SynthStart(synth, text)

	file, err := os.Create("output.raw")
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	for {
		status := C.SynthNext(synth, &chunk)
		if status == C.PIPER_DONE {
			break
		}
		// Go slice из C указателя
		samples := unsafe.Slice((*float32)(unsafe.Pointer(chunk.samples)), int(chunk.num_samples))
		// Пишем бинарно
		for _, s := range samples {
			_ = s // Пишем как нужно, например:
			if _, err := file.Write((*[4]byte)(unsafe.Pointer(&s))[:]); err != nil {
				return fmt.Errorf("failed to save in file: %w", err)
			}
		}
	}

	return nil
}

func main() {
	if err := _main(); err != nil {
		log.Fatalln(err)
	}
}
