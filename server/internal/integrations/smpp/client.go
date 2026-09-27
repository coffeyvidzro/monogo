// Package smpp adapts github.com/fiorix/go-smpp and owns all SMPP protocol state.
package smpp

import (
	"context"
	"errors"
	"sync"
	"time"

	gosmpp "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
)

type binder interface {
	Bind() <-chan gosmpp.ConnStatus
	Close() error
}
type submitter interface {
	Submit(*gosmpp.ShortMessage) (*gosmpp.ShortMessage, error)
}
type Client struct {
	config     Config
	session    binder
	submitter  submitter
	mu         sync.RWMutex
	state      ConnectionState
	lastErr    error
	started    bool
	states     chan StateChange
	inbound    chan Inbound
	deliveries chan Delivery
	errors     chan error
	cancel     context.CancelFunc
	closeOnce  sync.Once
	closeErr   error
}

func New(config Config) (*Client, error) {
	if config.BindMode == "" {
		config.BindMode = BindTransceiver
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	c := &Client{config: config, state: StateDisconnected, states: make(chan StateChange, 16), inbound: make(chan Inbound, 64), deliveries: make(chan Delivery, 64), errors: make(chan error, 16)}
	handler := func(body pdu.Body) { c.handlePDU(body) }
	switch config.BindMode {
	case BindTransmitter:
		tx := &gosmpp.Transmitter{Addr: config.Address(), User: config.SystemID, Passwd: config.Password, SystemType: config.SystemType, EnquireLink: config.EnquireLink, EnquireLinkTimeout: config.EnquireLinkTimeout, RespTimeout: config.ResponseTimeout, BindInterval: config.ReconnectInterval, WindowSize: config.WindowSize}
		c.session = tx
		c.submitter = tx
	case BindReceiver:
		rx := &gosmpp.Receiver{Addr: config.Address(), User: config.SystemID, Passwd: config.Password, SystemType: config.SystemType, EnquireLink: config.EnquireLink, EnquireLinkTimeout: config.EnquireLinkTimeout, BindInterval: config.ReconnectInterval, Handler: handler}
		c.session = rx
	case BindTransceiver:
		tc := &gosmpp.Transceiver{Addr: config.Address(), User: config.SystemID, Passwd: config.Password, SystemType: config.SystemType, EnquireLink: config.EnquireLink, EnquireLinkTimeout: config.EnquireLinkTimeout, RespTimeout: config.ResponseTimeout, BindInterval: config.ReconnectInterval, WindowSize: config.WindowSize, Handler: handler}
		c.session = tc
		c.submitter = tc
	}
	return c, nil
}
func (c *Client) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.mu.Unlock()
	c.setState(StateConnecting, nil)
	statuses := c.session.Bind()
	go c.monitor(runCtx, statuses)
	go func() {
		<-runCtx.Done()
		_ = c.closeSession()
	}()
	return nil
}
func (c *Client) Close() error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	started := c.started
	c.started = false
	c.mu.Unlock()
	if !started {
		c.setState(StateClosed, nil)
		return nil
	}
	return c.closeSession()
}
func (c *Client) closeSession() error {
	c.closeOnce.Do(func() {
		c.closeErr = c.session.Close()
		if errors.Is(c.closeErr, gosmpp.ErrNotConnected) {
			c.closeErr = nil
		}
		c.setState(StateClosed, c.closeErr)
	})
	return c.closeErr
}
func (c *Client) State() (ConnectionState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state, c.lastErr
}
func (c *Client) StateChanges() <-chan StateChange { return c.states }
func (c *Client) Inbound() <-chan Inbound          { return c.inbound }
func (c *Client) Deliveries() <-chan Delivery      { return c.deliveries }
func (c *Client) Errors() <-chan error             { return c.errors }
func (c *Client) setState(state ConnectionState, err error) {
	c.mu.Lock()
	c.state = state
	c.lastErr = err
	c.mu.Unlock()
	select {
	case c.states <- StateChange{State: state, Err: err, At: time.Now().UTC()}:
	default:
	}
}
