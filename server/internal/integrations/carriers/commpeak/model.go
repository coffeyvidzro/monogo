package commpeak

import (
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

// Config contains Leamout-managed credentials; no customer carrier credentials are used.
type Config struct {
	Authorization string
	BaseURL       string
	HTTPClient    *http.Client
}

// TerminationCDRQuery selects a bounded range of outbound call records.
type TerminationCDRQuery struct {
	From         time.Time
	To           time.Time
	SIPAccountID string
	Page         int
	PerPage      int
}

func (q TerminationCDRQuery) validate() error {
	if q.From.IsZero() || q.To.IsZero() || q.To.Before(q.From) {
		return fmt.Errorf("commpeak CDR date range is required and must be ordered")
	}
	if q.Page < 0 || q.PerPage < 0 || q.PerPage > 1000 {
		return fmt.Errorf("commpeak CDR pagination is invalid")
	}
	if strings.ContainsAny(q.SIPAccountID, "\r\n") {
		return fmt.Errorf("commpeak SIP account ID is invalid")
	}
	return nil
}


// SIPEndpoint is a provisioned carrier destination or Leamout ingress.
// It contains transport configuration only; it does not select a call route.
type SIPEndpoint struct {
	Host      string
	Port      uint16
	Transport string
}

// SIPCredentials are internal trunk credentials. Never serialize them into
// customer-facing API responses or logs.
type SIPCredentials struct {
	Username string
	Password string
}

// TerminationConfig describes an already-provisioned CommPeak outbound trunk.
// The provider's SIP hostname and credentials must come from the account setup.
type TerminationConfig struct {
	Endpoint    SIPEndpoint
	Credentials *SIPCredentials
}

// OriginationConfig describes CommPeak's provisioned inbound delivery to
// Leamout. Source networks must be confirmed from the actual provider account.
type OriginationConfig struct {
	Ingress SIPEndpoint
	Sources []netip.Prefix
}
