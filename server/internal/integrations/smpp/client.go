// Package smpp adapts github.com/fiorix/go-smpp to the messaging runtime.
package smpp

import (
	"fmt"

	gosmpp "github.com/fiorix/go-smpp/smpp"
)

type Client struct{ transceiver *gosmpp.Transceiver }

func New(config Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Client{transceiver: &gosmpp.Transceiver{Addr: config.Address, User: config.SystemID, Passwd: config.Password, SystemType: config.SystemType, EnquireLink: config.EnquireLink, RespTimeout: config.ResponseTimeout}}, nil
}
func (c *Client) Bind() <-chan gosmpp.ConnStatus { return c.transceiver.Bind() }
func (c *Client) Close() error {
	if c == nil || c.transceiver == nil {
		return nil
	}
	if err := c.transceiver.Close(); err != nil {
		return fmt.Errorf("close SMPP session: %w", err)
	}
	return nil
}
