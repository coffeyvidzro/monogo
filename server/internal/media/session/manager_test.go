package session_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/engine/echo"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func TestManagerOwnsOneAttachmentAndEchoes(t *testing.T) {
	cfg := validConfig()
	manager := newManager(t, 1)
	if err := manager.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	connection := newFakeConnection(cfg)
	done := make(chan error, 1)
	go func() { done <- manager.Attach(context.Background(), connection) }()

	connection.incoming <- session.AudioFrame{Data: []byte{1, 0}, Format: cfg.InputFormat}
	select {
	case got := <-connection.outgoing:
		if string(got.Data) != string([]byte{1, 0}) {
			t.Fatalf("echo = %v", got.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for echo")
	}

	second := newFakeConnection(cfg)
	if err := manager.Attach(context.Background(), second); !errors.Is(err, session.ErrSessionAlreadyAttached) {
		t.Fatalf("second Attach() error = %v", err)
	}
	connection.closeInput()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Attach() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Attach() did not stop")
	}
	if manager.Active() != 0 {
		t.Fatalf("active sessions = %d", manager.Active())
	}
}


func TestManagerBargeInInterruptsAndClearsPlayback(t *testing.T) {
	format := session.AudioFormat{SampleRateHz: 24000, Channels: 1}
	cfg := session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine: session.EngineIntegrated, InputFormat: format, OutputFormat: format,
	}
	stream := newFakeStream()
	manager, err := session.NewManager(1, time.Minute, map[session.Engine]session.Starter{
		session.EngineIntegrated: fakeStarter{stream: stream},
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if err := manager.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	connection := newFakeConnection(cfg)
	done := make(chan error, 1)
	go func() { done <- manager.Attach(context.Background(), connection) }()

	stream.events <- session.Event{Type: session.EventResponseStarted}
	stream.events <- session.Event{Type: session.EventSpeechStarted}

	select {
	case <-stream.interrupted:
	case <-time.After(time.Second):
		t.Fatal("stream was not interrupted on barge-in")
	}
	select {
	case <-connection.cleared:
	case <-time.After(time.Second):
		t.Fatal("playback was not cleared on barge-in")
	}

	connection.closeInput()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Attach() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Attach() did not stop")
	}
}

type fakeStarter struct {
	stream *fakeStream
}

func (s fakeStarter) Start(context.Context, session.Config) (session.Stream, error) {
	return s.stream, nil
}

type fakeStream struct {
	audio       chan session.AudioFrame
	events      chan session.Event
	interrupted chan struct{}
	interruptOnce sync.Once
	closeOnce     sync.Once
}

func newFakeStream() *fakeStream {
	return &fakeStream{
		audio: make(chan session.AudioFrame, 1),
		events: make(chan session.Event, 4),
		interrupted: make(chan struct{}),
	}
}

func (s *fakeStream) SendAudio(context.Context, session.AudioFrame) error { return nil }
func (s *fakeStream) Interrupt(context.Context) error {
	s.interruptOnce.Do(func() { close(s.interrupted) })
	return nil
}
func (s *fakeStream) Audio() <-chan session.AudioFrame { return s.audio }
func (s *fakeStream) Events() <-chan session.Event { return s.events }
func (s *fakeStream) Close(context.Context) error {
	s.closeOnce.Do(func() {
		close(s.audio)
		close(s.events)
	})
	return nil
}

func TestManagerCapacityAndDrain(t *testing.T) {
	manager := newManager(t, 1)
	first := validConfig()
	if err := manager.Start(context.Background(), first); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := manager.Start(context.Background(), validConfig()); !errors.Is(err, session.ErrCapacityExceeded) {
		t.Fatalf("capacity Start() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Drain(ctx); err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
	if err := manager.Start(context.Background(), validConfig()); !errors.Is(err, session.ErrManagerDraining) {
		t.Fatalf("draining Start() error = %v", err)
	}
}

func TestManagerRejectsMismatchedIdentity(t *testing.T) {
	manager := newManager(t, 1)
	cfg := validConfig()
	if err := manager.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	connection := newFakeConnection(cfg)
	connection.metadata.OrganizationID = uuid.New()
	if err := manager.Attach(context.Background(), connection); err == nil {
		t.Fatal("Attach() accepted mismatched organization")
	}
}

func TestManagerExpiresUnattachedSession(t *testing.T) {
	manager, err := session.NewManager(1, 10*time.Millisecond, map[session.Engine]session.Starter{
		session.EngineEcho: echo.Engine{},
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if err := manager.Start(context.Background(), validConfig()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for manager.Active() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if manager.Active() != 0 {
		t.Fatal("unattached session did not expire")
	}
}

func newManager(t *testing.T, capacity int) *session.Manager {
	t.Helper()
	manager, err := session.NewManager(capacity, time.Minute, map[session.Engine]session.Starter{
		session.EngineEcho: echo.Engine{},
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager
}

func validConfig() session.Config {
	format := session.AudioFormat{SampleRateHz: 16000, Channels: 1}
	return session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine: session.EngineEcho, InputFormat: format, OutputFormat: format,
	}
}

type fakeConnection struct {
	metadata session.ConnectionMetadata
	incoming chan session.AudioFrame
	outgoing chan session.AudioFrame
	cleared  chan struct{}
	closed   chan struct{}
	once     sync.Once
	clearOnce sync.Once
}

func newFakeConnection(cfg session.Config) *fakeConnection {
	return &fakeConnection{
		metadata: session.ConnectionMetadata{
			SessionID: cfg.ID, OrganizationID: cfg.OrganizationID, CallID: cfg.CallID,
			ChannelID: cfg.ChannelID, Format: cfg.InputFormat,
		},
		incoming: make(chan session.AudioFrame, 1), outgoing: make(chan session.AudioFrame, 1),
		cleared: make(chan struct{}), closed: make(chan struct{}),
	}
}

func (c *fakeConnection) Metadata() session.ConnectionMetadata { return c.metadata }
func (c *fakeConnection) ReceiveAudio(ctx context.Context) (session.AudioFrame, error) {
	select {
	case frame, ok := <-c.incoming:
		if !ok {
			return session.AudioFrame{}, io.EOF
		}
		return frame, nil
	case <-ctx.Done():
		return session.AudioFrame{}, ctx.Err()
	}
}
func (c *fakeConnection) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	select {
	case c.outgoing <- frame:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (c *fakeConnection) ClearPlayback(context.Context) error {
	c.clearOnce.Do(func() { close(c.cleared) })
	return nil
}
func (c *fakeConnection) Close() error { c.once.Do(func() { close(c.closed) }); return nil }
func (c *fakeConnection) closeInput()  { close(c.incoming) }
