// Package transport defines the bidirectional connection between FreeSWITCH
// audio forks and the media worker.
package transport

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

// Metadata is authenticated before a connection is handed to a session.
type Metadata struct {
	SessionID      string
	CallID         string
	OrganizationID string
	RemoteAddress  string
}

// Connection transports PCM frames without persistence or redelivery.
// ReceiveAudio blocks until a frame is available, the context is cancelled,
// or the connection is closed.
type Connection interface {
	Metadata() Metadata
	ReceiveAudio(context.Context) (session.AudioFrame, error)
	SendAudio(context.Context, session.AudioFrame) error
	Close() error
}

// Acceptor accepts authenticated media connections from FreeSWITCH.
type Acceptor interface {
	Accept(context.Context) (Connection, error)
	Close() error
}
