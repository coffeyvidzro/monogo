package didww

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPurchaseDIDAndLookupByReference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.Path != "/v3/orders" {
				t.Errorf("unexpected order path %s", r.URL.Path)
			}
			var payload struct {
				Data struct {
					Attributes struct {
						ExternalReferenceID string `json:"external_reference_id"`
						AllowBackOrdering  bool   `json:"allow_back_ordering"`
						Items              []struct {
							Type       string `json:"type"`
							Attributes struct {
								SKUID            string `json:"sku_id"`
								DidReservationID string `json:"did_reservation_id"`
							} `json:"attributes"`
						} `json:"items"`
					} `json:"attributes"`
				} `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			a := payload.Data.Attributes
			if a.ExternalReferenceID != "operation-1" || a.AllowBackOrdering || len(a.Items) != 1 ||
				a.Items[0].Type != "did_order_items" || a.Items[0].Attributes.SKUID != "sku-1" ||
				a.Items[0].Attributes.DidReservationID != "hold-1" {
				t.Errorf("unexpected purchase: %+v", a)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":"order-1","type":"orders","attributes":{"status":"pending"}}}`))
		case http.MethodGet:
			if r.URL.Query().Get("filter[external_reference_id]") != "operation-1" {
				t.Errorf("lookup filter = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"order-1","type":"orders","attributes":{"status":"completed"}}]}`))
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "key", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	order, err := client.OrderDID(context.Background(), OrderDIDRequest{
		SKUID: "sku-1", ReservationID: "hold-1", ExternalReferenceID: "operation-1",
	})
	if err != nil || order.ID != "order-1" || order.Attributes.Status != "pending" {
		t.Fatalf("order = %+v, error = %v", order, err)
	}
	found, ok, err := client.FindOrderByExternalReference(context.Background(), "operation-1")
	if err != nil || !ok || found.Attributes.Status != "completed" {
		t.Fatalf("order lookup = %+v, found = %t, error = %v", found, ok, err)
	}
}

func TestPurchaseRequiresOneInventoryReference(t *testing.T) {
	client, err := New(Config{APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.OrderDID(context.Background(), OrderDIDRequest{
		SKUID: "sku-1", AvailableDIDID: "stock-1", ReservationID: "hold-1", ExternalReferenceID: "operation-1",
	})
	if err == nil {
		t.Fatal("expected mutually exclusive inventory reference error")
	}
}
