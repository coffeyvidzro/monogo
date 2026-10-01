package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

var (
	ErrSessionNotFound        = errors.New("media session not found")
	ErrSessionAlreadyExists   = errors.New("media session already exists")
	ErrSessionAlreadyAttached = errors.New("media session already attached")
	ErrControlAlreadyAttached = errors.New("media control already attached")
	ErrControlBackpressure    = errors.New("media control event buffer is full")
	ErrManagerDraining        = errors.New("media session manager is draining")
	ErrCapacityExceeded       = errors.New("media session capacity exceeded")
)

type Manager struct {
	mu            sync.Mutex
	engines       map[Engine]Starter
	sessions      map[uuid.UUID]*managedSession
	capacity      int
	attachTimeout time.Duration
	draining      bool
	changed       chan struct{}
}

func NewManager(capacity int, attachTimeout time.Duration, engines map[Engine]Starter) (*Manager, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("media session capacity must be positive")
	}
	if len(engines) == 0 {
		return nil, fmt.Errorf("at least one media engine is required")
	}
	if attachTimeout <= 0 {
		return nil, fmt.Errorf("media attachment timeout must be positive")
	}
	copyEngines := make(map[Engine]Starter, len(engines))
	for name, engine := range engines {
		if engine == nil {
			return nil, fmt.Errorf("media engine %q is nil", name)
		}
		copyEngines[name] = engine
	}
	return &Manager{
		engines: copyEngines, sessions: make(map[uuid.UUID]*managedSession),
		capacity: capacity, attachTimeout: attachTimeout, changed: make(chan struct{}, 1),
	}, nil
}

func (m *Manager) Start(ctx context.Context, cfg Config) error {
	if ctx == nil {
		return fmt.Errorf("media session context is required")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	if m.draining {
		m.mu.Unlock()
		return ErrManagerDraining
	}
	if _, exists := m.sessions[cfg.ID]; exists {
		m.mu.Unlock()
		return ErrSessionAlreadyExists
	}
	if len(m.sessions) >= m.capacity {
		m.mu.Unlock()
		return ErrCapacityExceeded
	}
	starter, exists := m.engines[cfg.Engine]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("media engine %q is not configured", cfg.Engine)
	}
	managedCtx, cancel := context.WithCancel(context.Background())
	managed := &managedSession{
		config: cfg, ctx: managedCtx, cancel: cancel, complete: make(chan struct{}),
		controlDone: make(chan struct{}),
	}
	stream, err := starter.Start(managedCtx, cfg)
	if err != nil {
		m.mu.Unlock()
		cancel()
		return fmt.Errorf("start media engine: %w", err)
	}
	managed.stream = stream
	m.sessions[cfg.ID] = managed
	managed.attachTimer = time.AfterFunc(m.attachTimeout, func() {
		if managed.finishUnattached() {
			m.remove(cfg.ID, managed)
		}
	})
	m.mu.Unlock()
	return nil
}

func (m *Manager) Attach(ctx context.Context, connection Connection) error {
	if ctx == nil {
		return fmt.Errorf("media attachment context is required")
	}
	if connection == nil {
		return fmt.Errorf("media connection is required")
	}
	metadata := connection.Metadata()
	m.mu.Lock()
	managed, exists := m.sessions[metadata.SessionID]
	m.mu.Unlock()
	if !exists {
		return ErrSessionNotFound
	}
	if metadata.CallID != managed.config.CallID || metadata.ChannelID != managed.config.ChannelID ||
		metadata.OrganizationID != managed.config.OrganizationID {
		return fmt.Errorf("media connection identity does not match session")
	}
	if metadata.Format != managed.config.InputFormat || metadata.Format != managed.config.OutputFormat {
		return fmt.Errorf("media connection format does not match session")
	}

	managed.mu.Lock()
	if managed.attached {
		managed.mu.Unlock()
		return ErrSessionAlreadyAttached
	}
	if managed.finished || managed.ctx.Err() != nil {
		managed.mu.Unlock()
		return ErrSessionNotFound
	}
	if managed.stream == nil {
		managed.mu.Unlock()
		return fmt.Errorf("media session engine is not ready")
	}
	managed.attached = true
	if managed.attachTimer != nil {
		managed.attachTimer.Stop()
	}
	managed.connection = connection
	stream := managed.stream
	managed.mu.Unlock()

	err := pump(ctx, managed.ctx, connection, stream, managed.publish)
	managed.finish()
	managed.markComplete()
	m.remove(metadata.SessionID, managed)
	if isNormalDisconnect(err) {
		return nil
	}
	return err
}

type ControlAttachment struct {
	manager *Manager
	id      uuid.UUID
	session *managedSession
	events  <-chan Event
	once    sync.Once
}

func (m *Manager) AttachControl(id uuid.UUID) (*ControlAttachment, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("media session id is required")
	}
	m.mu.Lock()
	managed, exists := m.sessions[id]
	m.mu.Unlock()
	if !exists {
		return nil, ErrSessionNotFound
	}

	managed.mu.Lock()
	defer managed.mu.Unlock()
	if managed.finished || managed.ctx.Err() != nil {
		return nil, ErrSessionNotFound
	}
	if managed.controlOpen {
		return nil, ErrControlAlreadyAttached
	}
	events := make(chan Event, 64)
	managed.control = events
	managed.controlOpen = true
	return &ControlAttachment{manager: m, id: id, session: managed, events: events}, nil
}

func (c *ControlAttachment) Events() <-chan Event {
	return c.events
}

func (c *ControlAttachment) Command(ctx context.Context, command Command) error {
	if ctx == nil {
		return fmt.Errorf("media control context is required")
	}
	if err := command.Validate(); err != nil {
		return err
	}
	switch command.Type {
	case CommandInterrupt:
		return c.session.interrupt(ctx)
	case CommandStop:
		return c.manager.Stop(ctx, c.id)
	default:
		return fmt.Errorf("unsupported media command %q", command.Type)
	}
}

func (c *ControlAttachment) Close() {
	c.once.Do(func() {
		c.session.mu.Lock()
		control := c.session.control
		if c.session.controlOpen && control != nil {
			c.session.controlOpen = false
			c.session.control = nil
			close(control)
		}
		c.session.mu.Unlock()
	})
}

func (m *Manager) Stop(ctx context.Context, id uuid.UUID) error {
	if ctx == nil {
		return fmt.Errorf("media stop context is required")
	}
	m.mu.Lock()
	managed, exists := m.sessions[id]
	m.mu.Unlock()
	if !exists {
		return ErrSessionNotFound
	}
	managed.finish()
	select {
	case <-managed.complete:
	case <-ctx.Done():
		return ctx.Err()
	}
	m.remove(id, managed)
	return nil
}

func (m *Manager) Drain(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("media drain context is required")
	}
	m.mu.Lock()
	m.draining = true
	snapshot := make(map[uuid.UUID]*managedSession, len(m.sessions))
	for id, managed := range m.sessions {
		snapshot[id] = managed
	}
	m.mu.Unlock()
	for _, managed := range snapshot {
		managed.finish()
	}
	for id, managed := range snapshot {
		select {
		case <-managed.complete:
			m.remove(id, managed)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	for {
		m.mu.Lock()
		empty := len(m.sessions) == 0
		m.mu.Unlock()
		if empty {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.changed:
		}
	}
}

func (m *Manager) Ready() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return !m.draining && len(m.sessions) < m.capacity
}

func (m *Manager) Active() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

// Config returns the immutable configuration of a live session. It is used by
// the control endpoint to make repeated create requests idempotent.
func (m *Manager) Config(id uuid.UUID) (Config, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	managed, exists := m.sessions[id]
	if !exists {
		return Config{}, false
	}
	return managed.config, true
}

func (m *Manager) remove(id uuid.UUID, managed *managedSession) {
	m.mu.Lock()
	if current, exists := m.sessions[id]; exists && current == managed {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	select {
	case m.changed <- struct{}{}:
	default:
	}
}

type managedSession struct {
	config   Config
	ctx      context.Context
	cancel   context.CancelFunc
	complete chan struct{}

	mu           sync.Mutex
	control      chan Event
	controlOpen  bool
	controlDone  chan struct{}
	controlOnce  sync.Once
	stream       Stream
	connection   Connection
	attached     bool
	attachTimer  *time.Timer
	finished     bool
	finishOnce   sync.Once
	completeOnce sync.Once
}

func (s *managedSession) finish() {
	s.mu.Lock()
	s.finished = true
	attached := s.attached
	if s.attachTimer != nil {
		s.attachTimer.Stop()
	}
	s.mu.Unlock()
	s.finishResources(attached)
}

func (s *managedSession) finishUnattached() bool {
	s.mu.Lock()
	if s.finished || s.attached {
		s.mu.Unlock()
		return false
	}
	s.finished = true
	s.mu.Unlock()
	s.finishResources(false)
	return true
}

func (s *managedSession) finishResources(attached bool) {
	s.finishOnce.Do(func() {
		s.cancel()
		s.closeControl()
		s.mu.Lock()
		stream, connection := s.stream, s.connection
		s.mu.Unlock()
		if stream != nil {
			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = stream.Close(closeCtx)
			cancel()
		}
		if connection != nil {
			_ = connection.Close()
		}
		if !attached {
			s.markComplete()
		}
	})
}

func (s *managedSession) interrupt(ctx context.Context) error {
	s.mu.Lock()
	if s.finished || s.stream == nil {
		s.mu.Unlock()
		return ErrSessionNotFound
	}
	stream, connection := s.stream, s.connection
	s.mu.Unlock()

	if err := stream.Interrupt(ctx); err != nil {
		return err
	}
	if connection != nil {
		if err := connection.ClearPlayback(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *managedSession) publish(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.controlOpen || s.control == nil {
		return nil
	}
	select {
	case s.control <- event:
		return nil
	default:
		return ErrControlBackpressure
	}
}

func (s *managedSession) closeControl() {
	s.controlOnce.Do(func() {
		s.mu.Lock()
		control := s.control
		s.controlOpen = false
		s.control = nil
		s.mu.Unlock()
		if control != nil {
			close(control)
		}
		close(s.controlDone)
	})
}

func (s *managedSession) markComplete() {
	s.completeOnce.Do(func() { close(s.complete) })
}

func pump(
	parent, sessionCtx context.Context,
	connection Connection,
	stream Stream,
	publish func(Event) error,
) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	group, groupCtx := errgroup.WithContext(ctx)
	var playbackActive atomic.Bool
	var suppressPlayback atomic.Bool

	group.Go(func() error {
		for {
			frame, err := connection.ReceiveAudio(groupCtx)
			if err != nil {
				return err
			}
			if err := stream.SendAudio(groupCtx, frame); err != nil {
				return err
			}
		}
	})
	group.Go(func() error {
		for {
			select {
			case frame, ok := <-stream.Audio():
				if !ok {
					return io.EOF
				}
				if suppressPlayback.Load() {
					continue
				}
				playbackActive.Store(true)
				if err := connection.SendAudio(groupCtx, frame); err != nil {
					return err
				}
			case <-sessionCtx.Done():
				return sessionCtx.Err()
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
	})
	group.Go(func() error {
		for {
			select {
			case event, ok := <-stream.Events():
				if !ok {
					return io.EOF
				}
				if publish != nil {
					if err := publish(event); err != nil {
						return err
					}
				}
				switch event.Type {
				case EventError:
					if event.Failure != nil && event.Failure.Terminal {
						return fmt.Errorf("terminal media provider failure: %s", event.Failure.Message)
					}
				case EventResponseStarted:
					suppressPlayback.Store(false)
					playbackActive.Store(true)
				case EventResponseStopped:
					playbackActive.Store(false)
				case EventSpeechStarted:
					if playbackActive.Swap(false) {
						suppressPlayback.Store(true)
						if err := stream.Interrupt(groupCtx); err != nil {
							return err
						}
						if err := connection.ClearPlayback(groupCtx); err != nil {
							return err
						}
					}
				}
			case <-sessionCtx.Done():
				return sessionCtx.Err()
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
	})
	return group.Wait()
}

func isNormalDisconnect(err error) bool {
	return err == nil || errors.Is(err, context.Canceled) || errors.Is(err, io.EOF)
}
