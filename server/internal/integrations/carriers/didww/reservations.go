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

type Reservation struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Description *string   `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
		ExpiresAt   time.Time `json:"expires_at"`
	} `json:"attributes"`
	Relationships map[string]Relationship `json:"relationships"`
}

type ReservationList struct {
	Data  []Reservation `json:"data"`
	Meta  Meta          `json:"meta"`
	Links Links         `json:"links"`
}

// ReserveDID creates a temporary hold, which is not ownership or a purchase.
func (c *Client) ReserveDID(ctx context.Context, availableDIDID string) (Reservation, error) {
	if strings.TrimSpace(availableDIDID) == "" {
		return Reservation{}, fmt.Errorf("didww available DID ID is required")
	}
	payload := map[string]any{"data": map[string]any{
		"type": "did_reservations",
		"relationships": map[string]any{
			"available_did": map[string]any{"data": ResourceIdentifier{Type: "available_dids", ID: availableDIDID}},
		},
	}}
	var result single[Reservation]
	if err := c.do(ctx, http.MethodPost, "/did_reservations", nil, payload, &result); err != nil {
		return Reservation{}, err
	}
	return result.Data, nil
}

func (c *Client) GetReservation(ctx context.Context, id string) (Reservation, error) {
	path, err := resourcePath("/did_reservations", id)
	if err != nil {
		return Reservation{}, err
	}
	var result single[Reservation]
	if err := c.get(ctx, path, nil, &result); err != nil {
		return Reservation{}, err
	}
	return result.Data, nil
}

func (c *Client) ListReservations(ctx context.Context, page int) (ReservationList, error) {
	if page < 1 {
		return ReservationList{}, fmt.Errorf("didww page must be positive")
	}
	var result ReservationList
	if err := c.get(ctx, "/did_reservations", url.Values{"page[number]": {strconv.Itoa(page)}}, &result); err != nil {
		return ReservationList{}, err
	}
	return result, nil
}

func (c *Client) DeleteReservation(ctx context.Context, id string) error {
	path, err := resourcePath("/did_reservations", id)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}
