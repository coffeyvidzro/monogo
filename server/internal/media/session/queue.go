package session

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrInputLatencyBudgetExceeded  = errors.New("media input latency budget exceeded")
	ErrOutputLatencyBudgetExceeded = errors.New("media output latency budget exceeded")
	ErrFrameDurationExceeded       = errors.New("media frame duration exceeded")
	ErrProviderWriteTimeout        = errors.New("media provider write timeout")
	ErrPlaybackWriteTimeout        = errors.New("media playback write timeout")
	ErrProviderFailure             = errors.New("media provider failure")
)

type RuntimeLimits struct {
	MaxFrameDuration     time.Duration
	InputQueueDuration   time.Duration
	OutputQueueDuration  time.Duration
	ProviderStartTimeout time.Duration
	ProviderWriteTimeout time.Duration
	PlaybackWriteTimeout time.Duration
}

func DefaultRuntimeLimits() RuntimeLimits {
	return RuntimeLimits{
		MaxFrameDuration:     200 * time.Millisecond,
		InputQueueDuration:   400 * time.Millisecond,
		OutputQueueDuration:  800 * time.Millisecond,
		ProviderStartTimeout: 10 * time.Second,
		ProviderWriteTimeout: 2 * time.Second,
		PlaybackWriteTimeout: 2 * time.Second,
	}
}

func (l RuntimeLimits) validate() error {
	if l.MaxFrameDuration <= 0 ||
		l.InputQueueDuration <= 0 ||
		l.OutputQueueDuration <= 0 ||
		l.ProviderStartTimeout <= 0 ||
		l.ProviderWriteTimeout <= 0 ||
		l.PlaybackWriteTimeout <= 0 {
		return errors.New("media runtime limits must be positive")
	}
	return nil
}

type audioQueue struct {
	mu          sync.Mutex
	frames      chan AudioFrame
	queued      time.Duration
	maxDuration time.Duration
	overflowErr error
}

func newAudioQueue(maxDuration time.Duration, overflowErr error) *audioQueue {
	return &audioQueue{
		frames:      make(chan AudioFrame, 256),
		maxDuration: maxDuration,
		overflowErr: overflowErr,
	}
}

func (q *audioQueue) Push(frame AudioFrame) error {
	duration := frame.Duration()
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.queued+duration > q.maxDuration {
		return q.overflowErr
	}
	select {
	case q.frames <- frame:
		q.queued += duration
		return nil
	default:
		return q.overflowErr
	}
}

func (q *audioQueue) Pop(ctxDone <-chan struct{}) (AudioFrame, bool) {
	select {
	case frame := <-q.frames:
		q.mu.Lock()
		q.queued -= frame.Duration()
		if q.queued < 0 {
			q.queued = 0
		}
		q.mu.Unlock()
		return frame, true
	case <-ctxDone:
		return AudioFrame{}, false
	}
}

func (q *audioQueue) Clear() {
	for {
		select {
		case frame := <-q.frames:
			q.mu.Lock()
			q.queued -= frame.Duration()
			if q.queued < 0 {
				q.queued = 0
			}
			q.mu.Unlock()
		default:
			return
		}
	}
}
