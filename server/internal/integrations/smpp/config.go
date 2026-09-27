package smpp

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	Address, SystemID, Password, SystemType string
	EnquireLink, ResponseTimeout            time.Duration
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Address) == "" || strings.TrimSpace(c.SystemID) == "" {
		return fmt.Errorf("SMPP address and system id are required")
	}
	return nil
}
