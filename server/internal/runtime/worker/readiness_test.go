package worker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkerReadinessHandler(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{name: "ready after subscription", status: http.StatusOK, body: "ready\n"},
		{name: "dependency unavailable", err: errors.New("FreeSWITCH disconnected"), status: http.StatusServiceUnavailable, body: "worker dependencies unavailable\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := workerReadinessHandler(func(context.Context) error { return tt.err })
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.status)
			}
			if got := recorder.Body.String(); got != tt.body {
				t.Fatalf("body = %q, want %q", got, tt.body)
			}
			if strings.Contains(recorder.Body.String(), "FreeSWITCH disconnected") {
				t.Fatal("readiness response leaked dependency details")
			}
		})
	}
}

func TestWorkerReadinessRejectsOtherMethods(t *testing.T) {
	handler := workerReadinessHandler(func(context.Context) error {
		t.Fatal("readiness check must not run for POST")
		return nil
	})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/readyz", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}
