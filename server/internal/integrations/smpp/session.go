package smpp

import (
	"context"

	gosmpp "github.com/fiorix/go-smpp/smpp"
)

// monitor reflects go-smpp's automatic bind/rebind and enquire_link lifecycle.
func (c *Client) monitor(ctx context.Context, statuses <-chan gosmpp.ConnStatus) {
	for {
		select {
		case <-ctx.Done():
			return
		case status, ok := <-statuses:
			if !ok {
				return
			}
			switch status.Status() {
			case gosmpp.Connected:
				c.setState(StateConnected, nil)
			case gosmpp.Disconnected:
				c.setState(StateDisconnected, status.Error())
			case gosmpp.ConnectionFailed, gosmpp.BindFailed:
				c.setState(StateFailed, status.Error())
			}
		}
	}
}
