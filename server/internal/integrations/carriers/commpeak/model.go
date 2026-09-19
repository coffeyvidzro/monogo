package commpeak

import (
	"fmt"
	"net/http"
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
