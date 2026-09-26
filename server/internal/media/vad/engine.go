package vad

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

// State is the engine's current utterance state.
type State string

const (
	StateSilence State = "silence"
	StateSpeech  State = "speech"
)

// Decision describes a confirmed utterance boundary. AudioStart and AudioEnd
// include the padding selected by the engine and can be used to slice a
// separate rolling audio buffer.
type Decision struct {
	State      State
	Confidence float32
	OccurredAt time.Time
	AudioStart time.Time
	AudioEnd   time.Time
	BargeIn    bool
}

type Detector interface {
	Process(context.Context, session.AudioFrame) ([]Decision, error)
	Reset()
	Close() error
}

// EngineConfig controls hysteresis and utterance timing. Threshold starts an
// utterance; NegativeThreshold ends one, avoiding state changes around a
// single threshold.
type EngineConfig struct {
	Threshold         float32
	NegativeThreshold float32
	MinSpeech         time.Duration
	MinSilence        time.Duration
	SpeechPadding     time.Duration
	BargeInMinSpeech  time.Duration
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		Threshold:         0.5,
		NegativeThreshold: 0.35,
		MinSpeech:         96 * time.Millisecond,
		MinSilence:        320 * time.Millisecond,
		SpeechPadding:     200 * time.Millisecond,
		BargeInMinSpeech:  64 * time.Millisecond,
	}
}

// Engine turns fixed-window model probabilities into stable utterance events.
// Process, Reset, SetPlaybackActive, and Close are safe for concurrent use.
type Engine struct {
	mu sync.Mutex

	model  Model
	config EngineConfig
	buffer []float32

	bufferStart time.Time
	streamStart time.Time
	state       State
	above       time.Duration
	below       time.Duration
	speechStart time.Time
	playback    bool
	closed      bool
}

func NewEngine(model Model, cfg EngineConfig) (*Engine, error) {
	if model == nil {
		return nil, fmt.Errorf("VAD model is required")
	}
	if err := validateEngineConfig(cfg); err != nil {
		return nil, err
	}
	return &Engine{model: model, config: cfg, state: StateSilence}, nil
}

func validateEngineConfig(c EngineConfig) error {
	if c.Threshold <= 0 || c.Threshold > 1 {
		return fmt.Errorf("VAD threshold must be within (0, 1]")
	}
	if c.NegativeThreshold < 0 || c.NegativeThreshold >= c.Threshold {
		return fmt.Errorf("VAD negative threshold must be below threshold")
	}
	if c.MinSpeech <= 0 || c.MinSilence <= 0 || c.SpeechPadding < 0 || c.BargeInMinSpeech <= 0 {
		return fmt.Errorf("VAD durations are invalid")
	}
	return nil
}

// SetPlaybackActive enables barge-in detection. While playback is active, the
// engine confirms speech using BargeInMinSpeech instead of MinSpeech.
func (e *Engine) SetPlaybackActive(active bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.playback = active
}

// Process accepts arbitrary-sized PCM16 frames and only calls the model with
// complete 32 ms (512 sample) windows.
func (e *Engine) Process(ctx context.Context, frame session.AudioFrame) ([]Decision, error) {
	if ctx == nil {
		return nil, fmt.Errorf("VAD context is required")
	}
	if err := frame.Validate(); err != nil {
		return nil, err
	}
	if frame.Format.SampleRateHz != SampleRate || frame.Format.Channels != 1 {
		return nil, fmt.Errorf("VAD requires mono 16 kHz PCM")
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil, fmt.Errorf("VAD engine is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(e.buffer) == 0 {
		e.bufferStart = frame.CapturedAt
		if e.streamStart.IsZero() {
			e.streamStart = frame.CapturedAt
		}
	}
	e.buffer = append(e.buffer, pcm16(frame.Data)...)

	var decisions []Decision
	windowDuration := time.Duration(WindowSamples) * time.Second / SampleRate
	for len(e.buffer) >= WindowSamples {
		window := e.buffer[:WindowSamples]
		windowEnd := e.bufferStart.Add(windowDuration)
		probability, err := e.model.Probability(ctx, window)
		if err != nil {
			return nil, fmt.Errorf("evaluate VAD window: %w", err)
		}
		if decision := e.advance(probability, windowEnd, windowDuration); decision != nil {
			decisions = append(decisions, *decision)
		}
		e.buffer = e.buffer[WindowSamples:]
		e.bufferStart = windowEnd
	}
	return decisions, nil
}

func (e *Engine) advance(probability float32, at time.Time, window time.Duration) *Decision {
	if e.state == StateSilence {
		if probability < e.config.Threshold {
			e.above = 0
			return nil
		}

		e.above += window
		required := e.config.MinSpeech
		bargeIn := e.playback
		if bargeIn && e.config.BargeInMinSpeech < required {
			required = e.config.BargeInMinSpeech
		}
		if e.above < required {
			return nil
		}

		rawStart := at.Add(-e.above)
		padding := e.config.SpeechPadding
		if available := rawStart.Sub(e.streamStart); available < padding {
			padding = max(available, 0)
		}
		e.state = StateSpeech
		e.speechStart = rawStart.Add(-padding)
		e.above = 0
		e.below = 0
		return &Decision{
			State: StateSpeech, Confidence: probability, OccurredAt: at,
			AudioStart: e.speechStart, BargeIn: bargeIn,
		}
	}

	if probability > e.config.NegativeThreshold {
		e.below = 0
		return nil
	}
	e.below += window
	if e.below < e.config.MinSilence {
		return nil
	}

	rawEnd := at.Add(-e.below)
	// Retain at most half of the observed trailing silence. This produces a
	// natural boundary without retaining the entire endpointing delay.
	padding := min(e.config.SpeechPadding, e.below/2)
	audioEnd := rawEnd.Add(padding)
	decision := &Decision{
		State: StateSilence, Confidence: probability, OccurredAt: at,
		AudioStart: e.speechStart, AudioEnd: audioEnd,
	}
	e.state = StateSilence
	e.streamStart = audioEnd
	e.speechStart = time.Time{}
	e.above = 0
	e.below = 0
	return decision
}

// Reset clears model recurrent state and all partial utterance state.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return
	}
	e.model.Reset()
	e.buffer = nil
	e.bufferStart = time.Time{}
	e.streamStart = time.Time{}
	e.state = StateSilence
	e.above = 0
	e.below = 0
	e.speechStart = time.Time{}
	e.playback = false
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	e.closed = true
	return e.model.Close()
}

func pcm16(data []byte) []float32 {
	samples := make([]float32, len(data)/2)
	for i := range samples {
		samples[i] = float32(int16(binary.LittleEndian.Uint16(data[i*2:]))) / 32768
	}
	return samples
}
