package composable

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	providersdk "github.com/coffeyvidzro/monogo/internal/providers"
	"github.com/google/uuid"
)

func TestComposableRunsTurnThroughRegisteredProviders(t *testing.T) {
	transcriber := newFakeTranscriber()
	generator := &fakeGenerator{deltas: []string{"hello. ", "caller"}}
	synthesizer := &fakeSynthesizer{audio: []byte{1, 0, 2, 0}}
	stream := startTestStream(t, transcriber, generator, synthesizer)

	input := session.AudioFrame{Data: []byte{9, 0}, Format: testFormat()}
	if err := stream.SendAudio(context.Background(), input); err != nil {
		t.Fatalf("SendAudio() error = %v", err)
	}
	if got := <-transcriber.audio; string(got.Data) != string(input.Data) {
		t.Fatalf("transcriber audio = %v", got.Data)
	}
	transcriber.events <- providersdk.STTEvent{
		Type:       providersdk.STTEventSpeechStarted,
		ProviderID: "stt-1",
	}
	transcriber.events <- providersdk.STTEvent{
		Type:       providersdk.STTEventSpeechStopped,
		ProviderID: "stt-1",
		Text:       "hi",
	}

	select {
	case frame := <-stream.Audio():
		if string(frame.Data) != string(synthesizer.audio) {
			t.Fatalf("output audio = %v", frame.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for synthesized audio")
	}
	if generator.lastUser() != "hi" {
		t.Fatalf("LLM user message = %q", generator.lastUser())
	}
	if synthesizer.lastText() != "hello.caller" {
		t.Fatalf("TTS text = %q", synthesizer.lastText())
	}
	if got := synthesizer.continuations(); len(got) != 2 || !got[0] || got[1] {
		t.Fatalf("TTS continuation flags = %v", got)
	}

	foundResponseDelta := false
	for !foundResponseDelta {
		select {
		case event := <-stream.Events():
			if event.Type == session.EventResponseDelta {
				if event.Response == nil || event.Response.Text == "" {
					t.Fatalf("response delta payload = %+v", event.Response)
				}
				foundResponseDelta = true
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for response delta event")
		}
	}
	closeTestStream(t, stream)
}

func TestComposableBargeInCancelsResponseAndDropsStaleAudio(t *testing.T) {
	transcriber := newFakeTranscriber()
	synthesizer := &fakeSynthesizer{audio: []byte{7, 0}, waitForCancel: true}
	stream := startTestStream(t, transcriber, &fakeGenerator{deltas: []string{"first response"}}, synthesizer)
	transcriber.events <- providersdk.STTEvent{
		Type: providersdk.STTEventSpeechStopped,
		Text: "first",
	}

	select {
	case <-synthesizer.started:
	case <-time.After(time.Second):
		t.Fatal("TTS did not start")
	}
	transcriber.events <- providersdk.STTEvent{Type: providersdk.STTEventSpeechStarted}
	select {
	case <-synthesizer.cancelled:
	case <-time.After(time.Second):
		t.Fatal("barge-in did not cancel TTS")
	}
	select {
	case frame := <-stream.Audio():
		t.Fatalf("stale audio reached output: %v", frame.Data)
	case <-time.After(50 * time.Millisecond):
	}
	closeTestStream(t, stream)
}

func TestComposableRejectsUnregisteredProvider(t *testing.T) {
	registry, err := providersdk.NewRegistry(newFakeTranscriber(), &fakeGenerator{}, &fakeSynthesizer{})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	cfg := testSessionConfig()
	cfg.Providers = []session.ProviderRuntime{{
		Role: "llm", Provider: "missing", APIKey: "secret",
	}}
	_, err = (Engine{Registry: registry}).Start(context.Background(), cfg)
	if err == nil {
		t.Fatal("Start() error = nil")
	}
}

func startTestStream(
	t *testing.T,
	transcriber *fakeTranscriber,
	generator *fakeGenerator,
	synthesizer *fakeSynthesizer,
) session.Stream {
	t.Helper()
	if synthesizer.started == nil {
		synthesizer.started = make(chan struct{})
	}
	if synthesizer.cancelled == nil {
		synthesizer.cancelled = make(chan struct{})
	}
	registry, err := providersdk.NewRegistry(transcriber, generator, synthesizer)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	got, err := (Engine{Registry: registry}).Start(context.Background(), testSessionConfig())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return got
}

func testSessionConfig() session.Config {
	return session.Config{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		CallID:         uuid.New(),
		ChannelID:      uuid.New(),
		Engine:         session.EngineComposable,
		InputFormat:    testFormat(),
		OutputFormat:   testFormat(),
		Instructions:   "be helpful",
		Voice:          "voice-override",
	}
}

func closeTestStream(t *testing.T, stream session.Stream) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := stream.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func testFormat() session.AudioFormat {
	return session.AudioFormat{SampleRateHz: 16000, Channels: 1}
}

type fakeTranscriber struct {
	audio     chan session.AudioFrame
	events    chan providersdk.STTEvent
	closeOnce sync.Once
}

func newFakeTranscriber() *fakeTranscriber {
	return &fakeTranscriber{
		audio:  make(chan session.AudioFrame, 1),
		events: make(chan providersdk.STTEvent, 8),
	}
}

func (f *fakeTranscriber) Descriptor() providersdk.Descriptor {
	return providersdk.Descriptor{ID: "deepgram", Kind: providersdk.KindSTT}
}

func (f *fakeTranscriber) StartSTT(
	context.Context,
	providersdk.Runtime,
	session.AudioFormat,
	string,
) (providersdk.STTStream, error) {
	return f, nil
}

func (f *fakeTranscriber) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	select {
	case f.audio <- frame:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *fakeTranscriber) Finalize(context.Context) error      { return nil }
func (f *fakeTranscriber) Events() <-chan providersdk.STTEvent { return f.events }
func (f *fakeTranscriber) Close(context.Context) error {
	f.closeOnce.Do(func() { close(f.events) })
	return nil
}

type fakeGenerator struct {
	deltas   []string
	mu       sync.Mutex
	messages []providersdk.Message
}

func (f *fakeGenerator) Descriptor() providersdk.Descriptor {
	return providersdk.Descriptor{ID: "groq", Kind: providersdk.KindLLM}
}

func (f *fakeGenerator) Generate(
	_ context.Context,
	req providersdk.LLMRequest,
) (providersdk.LLMStream, error) {
	f.mu.Lock()
	f.messages = append([]providersdk.Message(nil), req.Messages...)
	f.mu.Unlock()
	events := make(chan providersdk.LLMEvent, len(f.deltas)+1)
	for _, delta := range f.deltas {
		events <- providersdk.LLMEvent{ResponseID: "llm-1", TextDelta: delta}
	}
	events <- providersdk.LLMEvent{Done: true}
	close(events)
	return &fakeLLMStream{events: events}, nil
}

func (f *fakeGenerator) lastUser() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.messages) - 1; i >= 0; i-- {
		if f.messages[i].Role == "user" {
			return f.messages[i].Content
		}
	}
	return ""
}

type fakeLLMStream struct {
	events chan providersdk.LLMEvent
}

func (f *fakeLLMStream) Events() <-chan providersdk.LLMEvent { return f.events }
func (f *fakeLLMStream) Close() error                        { return nil }

type fakeSynthesizer struct {
	audio         []byte
	waitForCancel bool
	started       chan struct{}
	cancelled     chan struct{}
	mu            sync.Mutex
	text          string
	more          []bool
}

func (f *fakeSynthesizer) Descriptor() providersdk.Descriptor {
	return providersdk.Descriptor{ID: "cartesia", Kind: providersdk.KindTTS}
}

func (f *fakeSynthesizer) StartTTS(
	ctx context.Context,
	req providersdk.TTSRequest,
) (providersdk.TTSStream, error) {
	f.mu.Lock()
	if f.started == nil {
		f.started = make(chan struct{})
	}
	if f.cancelled == nil {
		f.cancelled = make(chan struct{})
	}
	started, cancelled := f.started, f.cancelled
	f.mu.Unlock()
	events := make(chan providersdk.TTSEvent, 2)
	close(started)
	result := &fakeTTSStream{events: events}
	result.onText = func(text string, more bool) {
		f.mu.Lock()
		f.text += text
		f.more = append(f.more, more)
		f.mu.Unlock()
		if !more && !f.waitForCancel {
			events <- providersdk.TTSEvent{
				Audio: session.AudioFrame{Data: f.audio, Format: req.Format},
			}
			events <- providersdk.TTSEvent{Done: true}
			close(events)
		}
	}
	if f.waitForCancel {
		go func() {
			<-ctx.Done()
			close(cancelled)
			// Deliberately publish after cancellation to verify generation fencing.
			events <- providersdk.TTSEvent{
				Audio: session.AudioFrame{Data: f.audio, Format: req.Format},
			}
			close(events)
		}()
	}
	return result, nil
}

func (f *fakeSynthesizer) continuations() []bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]bool(nil), f.more...)
}

func (f *fakeSynthesizer) lastText() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.text
}

type fakeTTSStream struct {
	events chan providersdk.TTSEvent
	onText func(string, bool)
}

func (f *fakeTTSStream) Events() <-chan providersdk.TTSEvent { return f.events }
func (f *fakeTTSStream) Close() error                        { return nil }
func (f *fakeTTSStream) SendText(_ context.Context, text string, more bool) error {
	f.onText(text, more)
	return nil
}
