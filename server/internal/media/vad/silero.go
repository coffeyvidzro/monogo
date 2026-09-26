// Package vad implements streaming voice-activity detection.
package vad

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

const (
	SampleRate     = 16000
	WindowSamples  = 512
	contextSamples = 64
)

type Model interface {
	Probability(context.Context, []float32) (float32, error)
	Reset()
	Close() error
}

// SileroConfig identifies the model and ONNX Runtime library loaded by the
// process. Threads defaults to one to keep inference latency predictable.
type SileroConfig struct {
	ModelPath   string
	LibraryPath string
	Threads     int
}

// Silero implements Model using Silero VAD's stateful ONNX graph. One Silero
// instance must be used per audio stream because the recurrent state is local
// to that stream.
type Silero struct {
	mu         sync.Mutex
	session    *ort.AdvancedSession
	input      *ort.Tensor[float32]
	state      *ort.Tensor[float32]
	sampleRate *ort.Tensor[int64]
	output     *ort.Tensor[float32]
	nextState  *ort.Tensor[float32]
	context    []float32
	closed     bool
}

var environment struct {
	sync.Mutex
	initialized bool
	libraryPath string
}

func NewSilero(cfg SileroConfig) (*Silero, error) {
	if cfg.ModelPath == "" {
		return nil, fmt.Errorf("Silero model path is required")
	}
	if cfg.LibraryPath == "" {
		return nil, fmt.Errorf("ONNX Runtime shared library path is required")
	}
	if cfg.Threads <= 0 {
		cfg.Threads = 1
	}
	if err := initializeEnvironment(cfg.LibraryPath); err != nil {
		return nil, err
	}
	input, err := ort.NewEmptyTensor[float32](ort.NewShape(1, WindowSamples+contextSamples))
	if err != nil {
		return nil, fmt.Errorf("create Silero input: %w", err)
	}
	state, err := ort.NewEmptyTensor[float32](ort.NewShape(2, 1, 128))
	if err != nil {
		_ = input.Destroy()
		return nil, fmt.Errorf("create Silero state: %w", err)
	}
	sampleRate, err := ort.NewTensor(ort.NewShape(), []int64{SampleRate})
	if err != nil {
		_ = input.Destroy()
		_ = state.Destroy()
		return nil, fmt.Errorf("create Silero sample rate: %w", err)
	}
	output, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 1))
	if err != nil {
		_ = input.Destroy()
		_ = state.Destroy()
		_ = sampleRate.Destroy()
		return nil, fmt.Errorf("create Silero output: %w", err)
	}
	nextState, err := ort.NewEmptyTensor[float32](ort.NewShape(2, 1, 128))
	if err != nil {
		_ = input.Destroy()
		_ = state.Destroy()
		_ = sampleRate.Destroy()
		_ = output.Destroy()
		return nil, fmt.Errorf("create Silero next state: %w", err)
	}
	options, err := ort.NewSessionOptions()
	if err != nil {
		destroyValues(input, state, sampleRate, output, nextState)
		return nil, fmt.Errorf("create Silero session options: %w", err)
	}
	defer func() {
		_ = options.Destroy()
	}()
	if err := options.SetIntraOpNumThreads(cfg.Threads); err != nil {
		destroyValues(input, state, sampleRate, output, nextState)
		return nil, fmt.Errorf("configure ONNX intra-op threads: %w", err)
	}
	if err := options.SetInterOpNumThreads(1); err != nil {
		destroyValues(input, state, sampleRate, output, nextState)
		return nil, fmt.Errorf("configure ONNX inter-op threads: %w", err)
	}
	sess, err := ort.NewAdvancedSession(
		cfg.ModelPath,
		[]string{"input", "state", "sr"},
		[]string{"output", "stateN"},
		[]ort.Value{input, state, sampleRate},
		[]ort.Value{output, nextState},
		options,
	)
	if err != nil {
		destroyValues(input, state, sampleRate, output, nextState)
		return nil, fmt.Errorf("load Silero model: %w", err)
	}
	return &Silero{
		session:    sess,
		input:      input,
		state:      state,
		sampleRate: sampleRate,
		output:     output,
		nextState:  nextState,
		context:    make([]float32, contextSamples),
	}, nil
}

func initializeEnvironment(libraryPath string) error {
	environment.Lock()
	defer environment.Unlock()
	if environment.initialized {
		if environment.libraryPath != libraryPath {
			return fmt.Errorf("ONNX Runtime already initialized from %q", environment.libraryPath)
		}
		return nil
	}
	ort.SetSharedLibraryPath(libraryPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("initialize ONNX Runtime: %w", err)
	}
	environment.initialized = true
	environment.libraryPath = libraryPath
	return nil
}

func (s *Silero) Probability(ctx context.Context, samples []float32) (float32, error) {
	if ctx == nil {
		return 0, fmt.Errorf("Silero context is required")
	}
	if len(samples) != WindowSamples {
		return 0, fmt.Errorf("Silero requires %d samples, got %d", WindowSamples, len(samples))
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, fmt.Errorf("Silero model is closed")
	}
	input := s.input.GetData()
	copy(input, s.context)
	copy(input[contextSamples:], samples)
	if err := s.session.Run(); err != nil {
		return 0, fmt.Errorf("run Silero inference: %w", err)
	}
	copy(s.state.GetData(), s.nextState.GetData())
	copy(s.context, input[len(input)-contextSamples:])
	probability := s.output.GetData()[0]
	if math.IsNaN(float64(probability)) || math.IsInf(float64(probability), 0) || probability < 0 || probability > 1 {
		return 0, fmt.Errorf("Silero returned invalid probability %f", probability)
	}
	return probability, nil
}

func (s *Silero) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	clear(s.state.GetData())
	clear(s.nextState.GetData())
	clear(s.context)
}

func (s *Silero) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return errors.Join(
		s.session.Destroy(),
		s.input.Destroy(),
		s.state.Destroy(),
		s.sampleRate.Destroy(),
		s.output.Destroy(),
		s.nextState.Destroy(),
	)
}

func destroyValues(values ...ort.Value) {
	for _, value := range values {
		if value != nil {
			_ = value.Destroy()
		}
	}
}
