package didww

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReserveAndReleaseDID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.Path != "/v3/did_reservations" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			var payload struct {
				Data struct {
					Type          string `json:"type"`
					Relationships map[string]struct {
						Data ResourceIdentifier `json:"data"`
					} `json:"relationships"`
				} `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if payload.Data.Type != "did_reservations" ||
				payload.Data.Relationships["available_did"].Data.ID != "available-1" {
				t.Errorf("unexpected reservation request: %+v", payload)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":"hold-1","type":"did_reservations","attributes":{"expires_at":"2026-09-20T00:00:00Z"}}}`))
		case http.MethodDelete:
			if r.URL.Path != "/v3/did_reservations/hold-1" {
				t.Errorf("unexpected delete path %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "key", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	hold, err := client.ReserveDID(context.Background(), "available-1")
	if err != nil || hold.ID != "hold-1" || hold.Attributes.ExpiresAt.IsZero() {
		t.Fatalf("reservation = %+v, error = %v", hold, err)
	}
	if err := client.DeleteReservation(context.Background(), hold.ID); err != nil {
		t.Fatal(err)
	}
}
