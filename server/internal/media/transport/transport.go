// Package transport defines the bidirectional connection between FreeSWITCH
// audio forks and the media worker.
package transport

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

// Acceptor accepts authenticated media connections from FreeSWITCH.
type Acceptor interface {
	Accept(context.Context) (session.Connection, error)
	Close() error
}
