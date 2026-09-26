// Package echo implements a deterministic loopback engine for transport tests.
package echo

import (
	"context"
	"fmt"
	"sync"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Engine struct {
	Buffer int
}

func (e Engine) Start(ctx context.Context, cfg session.Config) (session.Stream, error) {
	if ctx == nil {
		return nil, fmt.Errorf("echo context is required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	buffer := e.Buffer
	if buffer <= 0 {
		buffer = 8
	}
	streamCtx, cancel := context.WithCancel(ctx)
	s := &stream{
		ctx: streamCtx, cancel: cancel,
		audio: make(chan session.AudioFrame, buffer), events: make(chan session.Event),
	}
	go func() {
		<-streamCtx.Done()
		s.sendMu.Lock()
		defer s.sendMu.Unlock()
		s.closeOnce.Do(func() { close(s.audio); close(s.events) })
	}()
	return s, nil
}

type stream struct {
	ctx       context.Context
	cancel    context.CancelFunc
	audio     chan session.AudioFrame
	events    chan session.Event
	sendMu    sync.RWMutex
	closeOnce sync.Once
}

func (s *stream) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	if err := frame.Validate(); err != nil {
		return err
	}
	copyFrame := frame
	copyFrame.Data = append([]byte(nil), frame.Data...)
	s.sendMu.RLock()
	defer s.sendMu.RUnlock()
	select {
	case s.audio <- copyFrame:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ctx.Done():
		return fmt.Errorf("echo stream is closed")
	}
}

func (s *stream) Interrupt(context.Context) error  { return nil }
func (s *stream) Audio() <-chan session.AudioFrame { return s.audio }
func (s *stream) Events() <-chan session.Event     { return s.events }
func (s *stream) Close(context.Context) error      { s.cancel(); return nil }
