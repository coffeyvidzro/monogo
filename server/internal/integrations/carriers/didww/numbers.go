package didww

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type AvailableDIDFilter struct {
	NumberContains string
	DIDGroupID     string
	CountryID      string
	RegionID       string
	CityID         string
	Feature        string
}

// SearchAvailableDIDs is intentionally non-paginated: DIDWW caps the result and
// supplies meta.total_count for the full matching inventory.
func (c *Client) SearchAvailableDIDs(ctx context.Context, filter AvailableDIDFilter) (AvailableDIDList, error) {
	params := url.Values{}
	setFilter(params, "number_contains", filter.NumberContains)
	setFilter(params, "did_group.id", filter.DIDGroupID)
	setFilter(params, "country.id", filter.CountryID)
	setFilter(params, "region.id", filter.RegionID)
	setFilter(params, "city.id", filter.CityID)
	setFilter(params, "did_group.features", filter.Feature)
	var result AvailableDIDList
	if err := c.get(ctx, "/available_dids", params, &result); err != nil {
		return AvailableDIDList{}, err
	}
	return result, nil
}

func (c *Client) GetAvailableDID(ctx context.Context, id string) (AvailableDID, error) {
	path, err := resourcePath("/available_dids", id)
	if err != nil {
		return AvailableDID{}, err
	}
	var result single[AvailableDID]
	if err := c.get(ctx, path, nil, &result); err != nil {
		return AvailableDID{}, err
	}
	return result.Data, nil
}

// ListDIDs returns numbers already owned by the provider account.
func (c *Client) ListDIDs(ctx context.Context, page int) (DIDList, error) {
	if page < 1 {
		return DIDList{}, fmt.Errorf("didww page must be positive")
	}
	var result DIDList
	if err := c.get(ctx, "/dids", url.Values{"page[number]": {strconv.Itoa(page)}}, &result); err != nil {
		return DIDList{}, err
	}
	return result, nil
}

func (c *Client) GetDID(ctx context.Context, id string) (DID, error) {
	path, err := resourcePath("/dids", id)
	if err != nil {
		return DID{}, err
	}
	var result single[DID]
	if err := c.get(ctx, path, url.Values{"include": {"voice_in_trunk"}}, &result); err != nil {
		return DID{}, err
	}
	return result.Data, nil
}

func (c *Client) FindDIDByNumber(ctx context.Context, number string) (DID, error) {
	number = strings.TrimPrefix(strings.TrimSpace(number), "+")
	if number == "" || strings.ContainsAny(number, "\r\n") {
		return DID{}, fmt.Errorf("didww number is required")
	}
	var result DIDList
	if err := c.get(ctx, "/dids", url.Values{
		"filter[number]": {number},
		"include":        {"voice_in_trunk"},
	}, &result); err != nil {
		return DID{}, err
	}
	if len(result.Data) != 1 {
		return DID{}, fmt.Errorf("didww number lookup returned %d resources", len(result.Data))
	}
	return result.Data[0], nil
}

// TerminateDID requests cancellation. Leamout must persist release intent and
// verify provider state independently before closing its local number record.
func (c *Client) TerminateDID(ctx context.Context, id string) (DID, error) {
	path, err := resourcePath("/dids", id)
	if err != nil {
		return DID{}, err
	}
	payload := map[string]any{"data": map[string]any{
		"type": "dids", "id": id, "attributes": map[string]any{"terminated": true},
	}}
	var result single[DID]
	if err := c.do(ctx, http.MethodPatch, path, nil, payload, &result); err != nil {
		return DID{}, err
	}
	return result.Data, nil
}
