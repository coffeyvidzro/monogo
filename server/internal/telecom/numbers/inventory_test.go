package numbers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func TestSearchAvailableReadsDIDWWInventoryWithoutExposingProviderIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v3/available_dids" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.URL.Query().Get("filter[number_contains]"); got != "5551234" {
			t.Errorf("unexpected inventory filter: %q", got)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[{"id":"private-didww-id","attributes":{"number":"12125551234"}},{"id":"private-duplicate","attributes":{"number":"+12125551234"}},{"id":"invalid","attributes":{"number":"not-a-number"}}]}`))
	}))
	defer server.Close()

	inventory, err := didww.New(didww.Config{APIKey: "test-key", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(nil, inventory)
	numbers, err := service.SearchAvailable(context.Background(), uuid.New(), " 5551234 ")
	if err != nil {
		t.Fatal(err)
	}
	if len(numbers) != 1 || numbers[0].Number != "+12125551234" {
		t.Fatalf("unexpected search results: %+v", numbers)
	}
	payload, err := json.Marshal(numbers)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "private-didww-id") || strings.Contains(string(payload), "provider") {
		t.Fatalf("internal provider details exposed: %s", payload)
	}
}

func TestSearchAvailableRejectsInvalidFiltersAndTenant(t *testing.T) {
	service := NewService(nil, nil)
	for _, value := range []string{"", "12", "1234567890123456", "12a4", "+1234"} {
		if _, err := service.SearchAvailable(context.Background(), uuid.New(), value); err == nil {
			t.Errorf("expected invalid filter to fail: %q", value)
		}
	}
	if _, err := service.SearchAvailable(context.Background(), uuid.Nil, "5551234"); err == nil {
		t.Fatal("expected missing tenant to fail")
	}
}

func TestSearchAvailableUnavailableWithoutPlatformCredentials(t *testing.T) {
	_, err := NewService(nil, nil).SearchAvailable(context.Background(), uuid.New(), "5551234")
	var appError *apperror.AppError
	if !errors.As(err, &appError) || appError.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("expected unavailable inventory, got %v", err)
	}
}

func TestSearchAvailableDoesNotReturnProviderErrorsAsInventory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	inventory, err := didww.New(didww.Config{APIKey: "test-key", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewService(nil, inventory).SearchAvailable(context.Background(), uuid.New(), "5551234")
	var appError *apperror.AppError
	if !errors.As(err, &appError) || appError.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("expected provider outage to fail closed, got %v", err)
	}
}
