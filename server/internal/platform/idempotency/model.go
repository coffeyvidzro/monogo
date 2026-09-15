package idempotency

import (
	"errors"
	"time"
)

const Header = "Idempotency-Key"

var (
	ErrKeyConflict = errors.New("idempotency key was already used for another request")
	ErrInProgress  = errors.New("request with this idempotency key is still processing")
)

type Request struct {
	Scope       string
	Key         string
	Method      string
	Path        string
	RequestHash string
}

type Response struct {
	Status      int
	Body        []byte
	ContentType string
	Headers     map[string][]string
}

type Claim struct {
	Response *Response
	Lease    time.Time
}

type Config struct {
	LockTTL   time.Duration
	RecordTTL time.Duration
}

func DefaultConfig() Config {
	return Config{LockTTL: 5 * time.Minute, RecordTTL: 24 * time.Hour}
}
