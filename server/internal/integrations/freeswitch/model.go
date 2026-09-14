package freeswitch

import (
	"fmt"
	"strings"
	"time"
)

const (
	ContentTypeAuthRequest  = "auth/request"
	ContentTypeCommandReply = "command/reply"
	ContentTypeAPIResponse  = "api/response"
	ContentTypeEventPlain   = "text/event-plain"
)

type Config struct {
	Address           string
	Password          string
	ConnectTimeout    time.Duration
	CommandTimeout    time.Duration
	ReconnectMinDelay time.Duration
	ReconnectMaxDelay time.Duration
}

func DefaultConfig(address, password string) Config {
	return Config{
		Address:           strings.TrimSpace(address),
		Password:          password,
		ConnectTimeout:    5 * time.Second,
		CommandTimeout:    5 * time.Second,
		ReconnectMinDelay: 250 * time.Millisecond,
		ReconnectMaxDelay: 5 * time.Second,
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Address) == "" {
		return fmt.Errorf("FreeSWITCH address is required")
	}
	if c.Password == "" {
		return fmt.Errorf("FreeSWITCH password is required")
	}
	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("FreeSWITCH connect timeout must be positive")
	}
	if c.CommandTimeout <= 0 {
		return fmt.Errorf("FreeSWITCH command timeout must be positive")
	}
	if c.ReconnectMinDelay <= 0 {
		return fmt.Errorf("FreeSWITCH reconnect minimum delay must be positive")
	}
	if c.ReconnectMaxDelay < c.ReconnectMinDelay {
		return fmt.Errorf("FreeSWITCH reconnect maximum delay must be greater than or equal to minimum delay")
	}
	return nil
}

type Frame struct {
	ContentType string
	Headers     map[string]string
	Body        string
}

func (f Frame) Header(name string) string {
	if value := f.Headers[name]; value != "" {
		return value
	}
	if f.ContentType == ContentTypeEventPlain {
		return plainEventHeader(f.Body, name)
	}
	return ""
}

func (f Frame) ReplyText() string {
	return f.Header("Reply-Text")
}

func (f Frame) OK() bool {
	return strings.HasPrefix(strings.ToUpper(f.ReplyText()), "+OK")
}

type Reply struct {
	Text string
	Body string
}

type Job struct {
	ID string
}

type Event struct {
	Name    string
	Headers map[string]string
	Body    string
}

func (e Event) Header(name string) string {
	if value := e.Headers[name]; value != "" {
		return value
	}
	return plainEventHeader(e.Body, name)
}

type EventFormat string

const (
	EventFormatPlain EventFormat = "plain"
	EventFormatJSON  EventFormat = "json"
)

type OriginateRequest struct {
	Endpoint    string
	Destination string
	CallerID    string
	Variables   map[string]string
}

func (r OriginateRequest) Validate() error {
	if strings.TrimSpace(r.Endpoint) == "" {
		return fmt.Errorf("FreeSWITCH originate endpoint is required")
	}
	if strings.TrimSpace(r.Destination) == "" {
		return fmt.Errorf("FreeSWITCH originate destination is required")
	}
	return nil
}

type TransferRequest struct {
	CallID      string
	Destination string
	Dialplan    string
	Context     string
}

func (r TransferRequest) Validate() error {
	if strings.TrimSpace(r.CallID) == "" {
		return fmt.Errorf("FreeSWITCH transfer call ID is required")
	}
	if strings.TrimSpace(r.Destination) == "" {
		return fmt.Errorf("FreeSWITCH transfer destination is required")
	}
	return nil
}

type RecordRequest struct {
	CallID string
	Path   string
	Action string
}

func (r RecordRequest) Validate() error {
	if strings.TrimSpace(r.CallID) == "" {
		return fmt.Errorf("FreeSWITCH record call ID is required")
	}
	if strings.TrimSpace(r.Path) == "" {
		return fmt.Errorf("FreeSWITCH record path is required")
	}
	if r.Action != "" && r.Action != "start" && r.Action != "stop" {
		return fmt.Errorf("FreeSWITCH record action must be start or stop, got %q", r.Action)
	}
	return nil
}

type Channel struct {
	UUID  string
	Name  string
	State string
}

type Call struct {
	UUID         string
	CallerName   string
	CallerNumber string
	Destination  string
	State        string
}

type Endpoint struct {
	Name string
	Type string
	Data string
}

type SIPProfileStatus struct {
	Profile string
	Raw     string
}

type ConferenceRequest struct {
	Name      string
	Command   string
	Arguments []string
}

func (r ConferenceRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("FreeSWITCH conference name is required")
	}
	if strings.TrimSpace(r.Command) == "" {
		return fmt.Errorf("FreeSWITCH conference command is required")
	}
	return nil
}

type ConferenceResult struct {
	Text string
	Body string
}

type ConferenceMember struct {
	ID       string
	CallerID string
	Muted    bool
	Deaf     bool
}

type ConferenceMembers struct {
	Conference string
	Members    []ConferenceMember
}
