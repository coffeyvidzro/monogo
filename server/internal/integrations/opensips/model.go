package opensips

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	URL            string
	ConnectTimeout time.Duration
	RequestTimeout time.Duration
}

func DefaultConfig(url string) Config {
	return Config{
		URL:            strings.TrimSpace(url),
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}
}

type Command struct {
	Name   string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

func (c Command) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("OpenSIPS command name is required")
	}
	return nil
}

type Response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (r Response) Validate() error {
	if r.JSONRPC != "2.0" {
		return fmt.Errorf("invalid OpenSIPS JSON-RPC version %q", r.JSONRPC)
	}
	if r.Error != nil {
		return fmt.Errorf("OpenSIPS MI error %d: %s", r.Error.Code, r.Error.Message)
	}
	return nil
}

type Event struct {
	Name      string            `json:"name"`
	Timestamp time.Time         `json:"timestamp"`
	Params    map[string]string `json:"params,omitempty"`
}

type SubscriberCacheRequest struct {
	Username string
	Domain   string
}

type Route struct {
	ID       string
	Carrier  string
	Prefix   string
	URI      string
	Priority int
	Enabled  bool
}

type Dialog struct {
	ID       string `json:"ID,omitempty"`
	CallID   string `json:"callid,omitempty"`
	FromTag  string `json:"from_tag,omitempty"`
	ToTag    string `json:"to_tag,omitempty"`
	State    string `json:"state,omitempty"`
	Duration int64  `json:"duration,omitempty"`
}

type DialogList struct {
	Count   int      `json:"count"`
	Dialogs []Dialog `json:"Dialogs"`
}

type AccessEntry struct {
	Address   string
	Reason    string
	ExpiresAt time.Time
}

type Statistics struct {
	ActiveDialogs    int64
	EarlyDialogs     int64
	ProcessedDialogs int64
	ExpiredDialogs   int64
	FailedDialogs    int64
}
