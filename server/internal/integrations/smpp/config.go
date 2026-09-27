package smpp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type BindMode string

const (
	BindTransmitter BindMode = "transmitter"
	BindReceiver    BindMode = "receiver"
	BindTransceiver BindMode = "transceiver"
)

type Config struct {
	Host               string
	Port               int
	SystemID           string
	Password           string
	SystemType         string
	BindMode           BindMode
	EnquireLink        time.Duration
	EnquireLinkTimeout time.Duration
	ResponseTimeout    time.Duration
	ReconnectInterval  time.Duration
	WindowSize         uint
}

func DefaultConfig(host string, port int, systemID, password string) Config {
	return Config{
		Host:               host,
		Port:               port,
		SystemID:           systemID,
		Password:           password,
		BindMode:           BindTransceiver,
		EnquireLink:        30 * time.Second,
		EnquireLinkTimeout: 90 * time.Second,
		ResponseTimeout:    5 * time.Second,
		ReconnectInterval:  5 * time.Second,
		WindowSize:         10,
	}
}
func (c Config) Address() string {
	return net.JoinHostPort(strings.TrimSpace(c.Host), strconv.Itoa(c.Port))
}
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("SMPP host is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("SMPP port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.SystemID) == "" {
		return fmt.Errorf("SMPP system_id is required")
	}
	if c.Password == "" {
		return fmt.Errorf("SMPP password is required")
	}
	if c.BindMode != "" && c.BindMode != BindTransmitter && c.BindMode != BindReceiver && c.BindMode != BindTransceiver {
		return fmt.Errorf("SMPP bind mode must be transmitter, receiver, or transceiver")
	}
	if c.EnquireLink < 10*time.Second {
		return fmt.Errorf("SMPP enquire_link must be at least 10s")
	}
	if c.EnquireLinkTimeout < c.EnquireLink {
		return fmt.Errorf("SMPP enquire_link timeout must not be shorter than enquire_link")
	}
	if c.ResponseTimeout <= 0 || c.ReconnectInterval <= 0 {
		return fmt.Errorf("SMPP response and reconnect timeouts must be positive")
	}
	return nil
}
