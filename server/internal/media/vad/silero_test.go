package vad

import (
	"context"
	"os"
	"testing"
)

func TestNewSileroRequiresPaths(t *testing.T) {
	if _, err := NewSilero(SileroConfig{}); err == nil {
		t.Fatal("NewSilero accepted an empty model path")
	}
	if _, err := NewSilero(SileroConfig{ModelPath: "silero_vad.onnx"}); err == nil {
		t.Fatal("NewSilero accepted an empty ONNX Runtime library path")
	}
}

func TestSileroInference(t *testing.T) {
	modelPath := os.Getenv("SILERO_ONNX_MODEL_PATH")
	libraryPath := os.Getenv("ONNXRUNTIME_SHARED_LIBRARY_PATH")
	if modelPath == "" || libraryPath == "" {
		t.Skip("set SILERO_ONNX_MODEL_PATH and ONNXRUNTIME_SHARED_LIBRARY_PATH to run ONNX inference")
	}

	model, err := NewSilero(SileroConfig{ModelPath: modelPath, LibraryPath: libraryPath})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := model.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	probability, err := model.Probability(context.Background(), make([]float32, WindowSamples))
	if err != nil {
		t.Fatal(err)
	}
	if probability < 0 || probability > 1 {
		t.Fatalf("probability = %f", probability)
	}
	model.Reset()
}
