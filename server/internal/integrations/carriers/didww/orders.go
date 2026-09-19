package didww

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Order struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Reference           string    `json:"reference"`
		ExternalReferenceID *string   `json:"external_reference_id"`
		Amount              string    `json:"amount"`
		Status              string    `json:"status"`
		Description         string    `json:"description"`
		CreatedAt           time.Time `json:"created_at"`
	} `json:"attributes"`
}

type OrderList struct {
	Data  []Order `json:"data"`
	Meta  Meta    `json:"meta"`
	Links Links   `json:"links"`
}

type OrderDIDRequest struct {
	SKUID               string
	AvailableDIDID      string
	ReservationID       string
	ExternalReferenceID string
	BillingCyclesCount  *int
	CallbackURL         string
}

// OrderDID creates a provider-side charge. Persist a unique acquisition intent
// and verify uncertain outcomes before any retry. The external reference does
// not establish an idempotency guarantee.
func (c *Client) OrderDID(ctx context.Context, request OrderDIDRequest) (Order, error) {
	if strings.TrimSpace(request.SKUID) == "" {
		return Order{}, fmt.Errorf("didww SKU ID is required")
	}
	if (request.AvailableDIDID == "") == (request.ReservationID == "") {
		return Order{}, fmt.Errorf("didww specify exactly one available DID or reservation ID")
	}
	if request.ExternalReferenceID == "" || len(request.ExternalReferenceID) > 100 {
		return Order{}, fmt.Errorf("didww external order reference must contain 1-100 characters")
	}
	attributes := map[string]any{
		"sku_id": request.SKUID,
	}
	if request.AvailableDIDID != "" {
		attributes["available_did_id"] = request.AvailableDIDID
	} else {
		attributes["did_reservation_id"] = request.ReservationID
	}
	if request.BillingCyclesCount != nil {
		if *request.BillingCyclesCount < 0 || *request.BillingCyclesCount > 999 {
			return Order{}, fmt.Errorf("didww billing cycles must be between 0 and 999")
		}
		attributes["billing_cycles_count"] = *request.BillingCyclesCount
	}
	orderAttributes := map[string]any{
		"allow_back_ordering":   false,
		"external_reference_id": request.ExternalReferenceID,
		"items":                 []any{map[string]any{"type": "did_order_items", "attributes": attributes}},
	}
	if request.CallbackURL != "" {
		u, err := url.Parse(request.CallbackURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
			return Order{}, fmt.Errorf("didww callback URL must be HTTPS")
		}
		orderAttributes["callback_url"] = request.CallbackURL
		orderAttributes["callback_method"] = "post"
	}
	payload := map[string]any{"data": map[string]any{"type": "orders", "attributes": orderAttributes}}
	var result single[Order]
	if err := c.do(ctx, http.MethodPost, "/orders", nil, payload, &result); err != nil {
		return Order{}, err
	}
	return result.Data, nil
}

func (c *Client) GetOrder(ctx context.Context, id string) (Order, error) {
	path, err := resourcePath("/orders", id)
	if err != nil {
		return Order{}, err
	}
	var result single[Order]
	if err := c.get(ctx, path, nil, &result); err != nil {
		return Order{}, err
	}
	return result.Data, nil
}

// FindOrderByExternalReference is a recovery lookup, not permission to issue a
// duplicate purchase if a provider response has been lost.
func (c *Client) FindOrderByExternalReference(ctx context.Context, reference string) (Order, bool, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return Order{}, false, fmt.Errorf("didww external order reference is required")
	}
	var result OrderList
	if err := c.get(ctx, "/orders", url.Values{"filter[external_reference_id]": {reference}}, &result); err != nil {
		return Order{}, false, err
	}
	if len(result.Data) > 1 {
		return Order{}, false, fmt.Errorf("didww external reference resolved to multiple orders")
	}
	if len(result.Data) == 0 {
		return Order{}, false, nil
	}
	return result.Data[0], true, nil
}

func (c *Client) ListOrders(ctx context.Context, page int) (OrderList, error) {
	if page < 1 {
		return OrderList{}, fmt.Errorf("didww page must be positive")
	}
	var result OrderList
	if err := c.get(ctx, "/orders", url.Values{"page[number]": {strconv.Itoa(page)}}, &result); err != nil {
		return OrderList{}, err
	}
	return result, nil
}

// CancelOrder is not the same as releasing an acquired DID.
func (c *Client) CancelOrder(ctx context.Context, id string) error {
	path, err := resourcePath("/orders", id)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}
