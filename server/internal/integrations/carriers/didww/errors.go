package didww

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError exposes only structured, non-sensitive upstream details.
type APIError struct {
	StatusCode int
	Code       string
	Pointer    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("didww API returned status %d (code %s)", e.StatusCode, e.Code)
	}
	return fmt.Sprintf("didww API returned status %d", e.StatusCode)
}

func (e *APIError) Retryable() bool {
	return e != nil && (e.StatusCode == http.StatusRequestTimeout ||
		e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= http.StatusInternalServerError)
}

// Retryable describes the HTTP response, not whether a purchase is safe to retry.
func decodeAPIError(resp *http.Response) error {
	var envelope struct {
		Errors []struct {
			Code   string `json:"code"`
			Source struct {
				Pointer string `json:"pointer"`
			} `json:"source"`
		} `json:"errors"`
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err == nil {
		_ = json.Unmarshal(payload, &envelope)
	}
	upstream := &APIError{StatusCode: resp.StatusCode}
	if len(envelope.Errors) > 0 {
		upstream.Code = envelope.Errors[0].Code
		upstream.Pointer = envelope.Errors[0].Source.Pointer
	}
	return upstream
}
