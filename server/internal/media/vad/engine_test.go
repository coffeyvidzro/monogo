package vad

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type fakeModel struct {
	probabilities []float32
	windows       [][]float32
	reset         bool
	closed        bool
	err           error
}

func (m *fakeModel) Probability(_ context.Context, samples []float32) (float32, error) {
	m.windows = append(m.windows, append([]float32(nil), samples...))
	if m.err != nil {
		return 0, m.err
	}
	probability := m.probabilities[0]
	m.probabilities = m.probabilities[1:]
	return probability, nil
}

func (m *fakeModel) Reset()       { m.reset = true }
func (m *fakeModel) Close() error { m.closed = true; return nil }

func TestEngineDetectsPaddedUtterance(t *testing.T) {
	start := time.Date(2026, time.September, 26, 12, 0, 0, 0, time.UTC)
	model := &fakeModel{probabilities: []float32{0.8, 0.9, 0.8, 0.1, 0.1}}
	engine, err := NewEngine(model, testConfig())
	if err != nil {
		t.Fatal(err)
	}

	var decisions []Decision
	for i := range 5 {
		got, processErr := engine.Process(context.Background(), audioFrame(start.Add(time.Duration(i)*32*time.Millisecond), WindowSamples, 1000))
		if processErr != nil {
			t.Fatal(processErr)
		}
		decisions = append(decisions, got...)
	}

	if len(decisions) != 2 {
		t.Fatalf("got %d decisions, want 2", len(decisions))
	}
	if decisions[0].State != StateSpeech || decisions[0].AudioStart != start || decisions[0].BargeIn {
		t.Fatalf("unexpected speech decision: %+v", decisions[0])
	}
	wantEnd := start.Add(128 * time.Millisecond)
	if decisions[1].State != StateSilence || decisions[1].AudioStart != start || decisions[1].AudioEnd != wantEnd {
		t.Fatalf("unexpected silence decision: %+v; want end %v", decisions[1], wantEnd)
	}
}

func TestEngineDetectsBargeInWithShorterConfirmation(t *testing.T) {
	model := &fakeModel{probabilities: []float32{0.8}}
	engine, err := NewEngine(model, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	engine.SetPlaybackActive(true)

	decisions, err := engine.Process(context.Background(), audioFrame(time.Unix(0, 0), WindowSamples, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || !decisions[0].BargeIn || decisions[0].State != StateSpeech {
		t.Fatalf("unexpected decisions: %+v", decisions)
	}
}

func TestEngineBuffersPartialFramesAndNormalizesPCM(t *testing.T) {
	model := &fakeModel{probabilities: []float32{0.1}}
	engine, err := NewEngine(model, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	start := time.Unix(0, 0)
	if got, err := engine.Process(context.Background(), audioFrame(start, WindowSamples/2, -32768)); err != nil || len(got) != 0 {
		t.Fatalf("first partial frame: decisions=%v err=%v", got, err)
	}
	if got, err := engine.Process(context.Background(), audioFrame(start.Add(16*time.Millisecond), WindowSamples/2, 16384)); err != nil || len(got) != 0 {
		t.Fatalf("second partial frame: decisions=%v err=%v", got, err)
	}
	if len(model.windows) != 1 || len(model.windows[0]) != WindowSamples {
		t.Fatalf("model windows: %d", len(model.windows))
	}
	if model.windows[0][0] != -1 || model.windows[0][WindowSamples-1] != 0.5 {
		t.Fatalf("PCM normalization was %f ... %f", model.windows[0][0], model.windows[0][WindowSamples-1])
	}
}

func TestEngineLifecycleAndErrors(t *testing.T) {
	modelErr := errors.New("inference failed")
	model := &fakeModel{probabilities: []float32{0.1}, err: modelErr}
	engine, err := NewEngine(model, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = engine.Process(context.Background(), audioFrame(time.Now(), WindowSamples, 0)); !errors.Is(err, modelErr) {
		t.Fatalf("Process error = %v", err)
	}
	engine.Reset()
	if !model.reset {
		t.Fatal("Reset did not reset model")
	}
	if err = engine.Close(); err != nil {
		t.Fatal(err)
	}
	if err = engine.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if !model.closed {
		t.Fatal("Close did not close model")
	}
	if _, err = engine.Process(context.Background(), audioFrame(time.Now(), WindowSamples, 0)); err == nil {
		t.Fatal("Process succeeded after Close")
	}
}

func TestNewEngineValidatesConfiguration(t *testing.T) {
	if _, err := NewEngine(nil, testConfig()); err == nil {
		t.Fatal("NewEngine accepted a nil model")
	}
	cfg := testConfig()
	cfg.NegativeThreshold = cfg.Threshold
	if _, err := NewEngine(&fakeModel{}, cfg); err == nil {
		t.Fatal("NewEngine accepted invalid hysteresis")
	}
}

func testConfig() EngineConfig {
	return EngineConfig{
		Threshold: 0.5, NegativeThreshold: 0.35,
		MinSpeech: 64 * time.Millisecond, MinSilence: 64 * time.Millisecond,
		SpeechPadding: 64 * time.Millisecond, BargeInMinSpeech: 32 * time.Millisecond,
	}
}

func audioFrame(at time.Time, samples int, value int16) session.AudioFrame {
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(value))
	}
	return session.AudioFrame{
		Data: data, Format: session.AudioFormat{SampleRateHz: SampleRate, Channels: 1}, CapturedAt: at,
	}
}
