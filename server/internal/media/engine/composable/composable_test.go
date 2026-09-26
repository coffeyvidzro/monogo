package composable

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/cartesia"
	"github.com/coffeyvidzro/monogo/internal/integrations/deepgram"
	"github.com/coffeyvidzro/monogo/internal/integrations/groq"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func TestComposableRunsFluxTurnThroughGroqAndCartesia(t *testing.T) {
	transcriber := newFakeTranscriber()
	generator := &fakeGenerator{deltas: []string{"hello ", "caller"}}
	synthesizer := &fakeSynthesizer{audio: []byte{1, 0, 2, 0}}
	stream := startTestStream(t, transcriber, generator, synthesizer)

	input := session.AudioFrame{Data: []byte{9, 0}, Format: testFormat()}
	if err := stream.SendAudio(context.Background(), input); err != nil {
		t.Fatalf("SendAudio() error = %v", err)
	}
	if got := <-transcriber.audio; string(got.Data) != string(input.Data) {
		t.Fatalf("transcriber audio = %v", got.Data)
	}
	transcriber.events <- deepgram.Event{TurnEvent: "StartOfTurn", RequestID: "dg-1"}
	transcriber.events <- deepgram.Event{TurnEvent: "EndOfTurn", RequestID: "dg-1", Transcript: deepgram.Transcript{Text: "hi", SpeechFinal: true}}

	select {
	case frame := <-stream.Audio():
		if string(frame.Data) != string(synthesizer.audio) {
			t.Fatalf("output audio = %v", frame.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for synthesized audio")
	}
	if generator.lastUser() != "hi" {
		t.Fatalf("Groq user message = %q", generator.lastUser())
	}
	if synthesizer.lastText() != "hello caller" {
		t.Fatalf("Cartesia text = %q", synthesizer.lastText())
	}
	if got := synthesizer.continuations(); len(got) != 2 || !got[0] || got[1] {
		t.Fatalf("Cartesia continuation flags = %v", got)
	}
	closeTestStream(t, stream)
}

func TestComposableBargeInCancelsResponseAndDropsStaleAudio(t *testing.T) {
	transcriber := newFakeTranscriber()
	synthesizer := &fakeSynthesizer{audio: []byte{7, 0}, waitForCancel: true}
	stream := startTestStream(t, transcriber, &fakeGenerator{deltas: []string{"first response"}}, synthesizer)
	transcriber.events <- deepgram.Event{TurnEvent: "EndOfTurn", Transcript: deepgram.Transcript{Text: "first"}}

	select {
	case <-synthesizer.started:
	case <-time.After(time.Second):
		t.Fatal("Cartesia did not start")
	}
	transcriber.events <- deepgram.Event{TurnEvent: "StartOfTurn"}
	select {
	case <-synthesizer.cancelled:
	case <-time.After(time.Second):
		t.Fatal("barge-in did not cancel Cartesia")
	}
	select {
	case frame := <-stream.Audio():
		t.Fatalf("stale audio reached output: %v", frame.Data)
	case <-time.After(50 * time.Millisecond):
	}
	closeTestStream(t, stream)
}

func startTestStream(t *testing.T, transcriber *fakeTranscriber, generator *fakeGenerator, synthesizer *fakeSynthesizer) session.Stream {
	t.Helper()
	if synthesizer.started == nil {
		synthesizer.started = make(chan struct{})
	}
	if synthesizer.cancelled == nil {
		synthesizer.cancelled = make(chan struct{})
	}
	cfg := session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine: session.EngineComposable, InputFormat: testFormat(), OutputFormat: testFormat(),
		Instructions: "be helpful", Voice: "voice-override",
	}
	got, err := (Engine{Transcriber: transcriber, Generator: generator, Synthesizer: synthesizer}).Start(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return got
}

func closeTestStream(t *testing.T, stream session.Stream) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := stream.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func testFormat() session.AudioFormat { return session.AudioFormat{SampleRateHz: 16000, Channels: 1} }

type fakeTranscriber struct {
	audio     chan session.AudioFrame
	events    chan deepgram.Event
	closeOnce sync.Once
}

func newFakeTranscriber() *fakeTranscriber {
	return &fakeTranscriber{audio: make(chan session.AudioFrame, 1), events: make(chan deepgram.Event, 8)}
}
func (f *fakeTranscriber) Start(context.Context, deepgram.Config, session.AudioFormat) (deepgram.Stream, error) {
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
func (f *fakeTranscriber) Finalize(context.Context) error { return nil }
func (f *fakeTranscriber) Events() <-chan deepgram.Event  { return f.events }
func (f *fakeTranscriber) Close(context.Context) error {
	f.closeOnce.Do(func() { close(f.events) })
	return nil
}

type fakeGenerator struct {
	deltas   []string
	mu       sync.Mutex
	messages []groq.Message
}

func (f *fakeGenerator) Generate(_ context.Context, _ groq.Config, messages []groq.Message) (groq.Stream, error) {
	f.mu.Lock()
	f.messages = append([]groq.Message(nil), messages...)
	f.mu.Unlock()
	events := make(chan groq.Event, len(f.deltas)+1)
	for _, delta := range f.deltas {
		events <- groq.Event{CompletionID: "groq-1", TextDelta: delta}
	}
	events <- groq.Event{Done: true}
	close(events)
	return &fakeGroqStream{events: events}, nil
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

type fakeGroqStream struct{ events chan groq.Event }

func (f *fakeGroqStream) Events() <-chan groq.Event { return f.events }
func (f *fakeGroqStream) Close() error              { return nil }

type fakeSynthesizer struct {
	audio         []byte
	waitForCancel bool
	started       chan struct{}
	cancelled     chan struct{}
	mu            sync.Mutex
	text          string
	more          []bool
}

func (f *fakeSynthesizer) StartSynthesis(ctx context.Context, _ cartesia.Config, format session.AudioFormat) (cartesia.TextStream, error) {
	f.mu.Lock()
	if f.started == nil {
		f.started = make(chan struct{})
	}
	if f.cancelled == nil {
		f.cancelled = make(chan struct{})
	}
	started, cancelled := f.started, f.cancelled
	f.mu.Unlock()
	events := make(chan cartesia.Event, 2)
	close(started)
	result := &fakeCartesiaStream{events: events}
	result.onText = func(text string, more bool) {
		f.mu.Lock()
		f.text += text
		f.more = append(f.more, more)
		f.mu.Unlock()
		if !more && !f.waitForCancel {
			events <- cartesia.Event{Audio: session.AudioFrame{Data: f.audio, Format: format}}
			events <- cartesia.Event{Done: true}
			close(events)
		}
	}
	if f.waitForCancel {
		go func() {
			<-ctx.Done()
			close(cancelled)
			// Deliberately publish after cancellation to verify generation fencing.
			events <- cartesia.Event{Audio: session.AudioFrame{Data: f.audio, Format: format}}
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

type fakeCartesiaStream struct {
	events chan cartesia.Event
	onText func(string, bool)
}

func (f *fakeCartesiaStream) Events() <-chan cartesia.Event { return f.events }
func (f *fakeCartesiaStream) Close() error                  { return nil }
func (f *fakeCartesiaStream) SendText(_ context.Context, text string, more bool) error {
	f.onText(text, more)
	return nil
}
