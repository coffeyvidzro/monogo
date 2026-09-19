package didww

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListDIDGroupsAndGetDID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Api-Key") != "test-key" || r.Header.Get("Accept") != "application/vnd.api+json" {
			t.Error("missing DIDWW headers")
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch r.URL.Path {
		case "/v3/did_groups":
			if got := r.URL.Query().Get("page[number]"); got != "2" {
				t.Errorf("page = %q", got)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"group-1","type":"did_groups","meta":{"is_available":true,"total_count":3}}]}`))
		case "/v3/dids/did-1":
			_, _ = w.Write([]byte(`{"data":{"id":"did-1","type":"dids","attributes":{"number":"233201234567"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	groups, err := client.ListDIDGroups(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups.Data) != 1 || !groups.Data[0].Meta.IsAvailable {
		t.Fatalf("unexpected groups: %+v", groups)
	}
	did, err := client.GetDID(context.Background(), "did-1")
	if err != nil || did.ID != "did-1" || did.Attributes.Number != "233201234567" {
		t.Fatalf("unexpected DID: %+v, error: %v", did, err)
	}
}

func TestDIDWWValidationAndErrors(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected missing API key error")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("sensitive upstream data"))
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "secret", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetDID(context.Background(), ""); err == nil {
		t.Fatal("expected invalid DID ID error")
	}
	_, err = client.ListDIDs(context.Background(), 1)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected API error: %v", err)
	}
	if strings.Contains(err.Error(), "sensitive") || strings.Contains(err.Error(), "secret") {
		t.Fatal("API error exposed sensitive data")
	}
}
