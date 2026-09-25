// Package vad defines voice-activity detection contracts used for turn-taking.
package vad

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

// State is the detector's stable voice-activity state after hysteresis.
type State string

const (
	StateSilence State = "silence"
	StateSpeech  State = "speech"
)

// Decision reports a stable state transition and its model confidence.
type Decision struct {
	State      State
	Confidence float32
	OccurredAt time.Time
}

// Detector consumes PCM frames in order. Reset discards utterance state while
// retaining loaded model resources.
type Detector interface {
	Process(context.Context, session.AudioFrame) ([]Decision, error)
	Reset()
	Close() error
}
