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
	"log"
	"os"
	"unsafe"
)

func main() {
	onnx := C.CString("./ru_RU-dmitri-medium.onnx")
	json := C.CString("./ru_RU-dmitri-medium.onnx.json")
	espeak := C.CString("./espeak-ng-data")
	defer C.free(unsafe.Pointer(onnx))
	defer C.free(unsafe.Pointer(json))
	defer C.free(unsafe.Pointer(espeak))

	synth := C.SynthCreate(onnx, json, espeak)
	defer C.SynthFree(synth)

	text := C.CString("Привет мир")
	defer C.free(unsafe.Pointer(text))
	C.SynthSetLength(synth, C.float(1.0)) // длина речи, можно менять

	C.SynthStart(synth, text)
	var chunk C.piper_audio_chunk
	file, err := os.Create("output.raw")
	if err != nil {
		log.Println("failed to open file:", err)
		return
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
				log.Println("failed to save in file", err)
				return
			}
		}
	}
}
