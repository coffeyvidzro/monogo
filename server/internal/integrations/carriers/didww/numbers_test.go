package didww

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAvailableInventoryAndTermination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-DIDWW-Api-Version"); got != DefaultAPIVersion {
			t.Errorf("API version = %q", got)
		}
		w.Header().Set("Content-Type", jsonAPIMediaType)
		switch r.URL.Path {
		case "/v3/available_dids":
			if r.URL.Query().Get("page[number]") != "" ||
				r.URL.Query().Get("filter[did_group.features]") != "voice_in" {
				t.Errorf("inventory filters = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"stock-1","type":"available_dids","attributes":{"number":"233201234567"}}],"meta":{"api_version":"2026-04-16","total_count":19}}`))
		case "/v3/dids/did-1":
			if r.Method != http.MethodPatch {
				t.Errorf("expected PATCH DID release, got %s", r.Method)
			}
			var document struct {
				Data struct {
					Type       string `json:"type"`
					ID         string `json:"id"`
					Attributes struct {
						Terminated bool `json:"terminated"`
					} `json:"attributes"`
				} `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&document); err != nil {
				t.Error(err)
			}
			if document.Data.Type != "dids" || document.Data.ID != "did-1" || !document.Data.Attributes.Terminated {
				t.Errorf("bad release body: %+v", document.Data)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"did-1","type":"dids","attributes":{"number":"233201234567","terminated":true}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "key", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	stock, err := client.SearchAvailableDIDs(context.Background(), AvailableDIDFilter{Feature: "voice_in"})
	if err != nil || len(stock.Data) != 1 || stock.Meta.TotalRecords != 19 {
		t.Fatalf("inventory = %+v, error = %v", stock, err)
	}
	did, err := client.TerminateDID(context.Background(), "did-1")
	if err != nil || !did.Attributes.Terminated {
		t.Fatalf("terminated DID = %+v, error = %v", did, err)
	}
}
