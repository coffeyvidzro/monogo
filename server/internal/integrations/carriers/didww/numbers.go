package didww

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ListDIDGroups reads DIDWW coverage without reserving or purchasing numbers.
func (c *Client) ListDIDGroups(ctx context.Context, page int) (DIDGroupList, error) {
	if page < 1 {
		return DIDGroupList{}, fmt.Errorf("didww page must be positive")
	}
	var result DIDGroupList
	err := c.get(ctx, "/did_groups", url.Values{"page[number]": {strconv.Itoa(page)}}, &result)
	if err != nil {
		return DIDGroupList{}, err
	}
	return result, nil
}

// GetDIDGroup reads availability metadata for one provider group.
func (c *Client) GetDIDGroup(ctx context.Context, id string) (DIDGroup, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, "/?#") {
		return DIDGroup{}, fmt.Errorf("didww DID group ID is invalid")
	}
	var result struct {
		Data DIDGroup `json:"data"`
	}
	if err := c.get(ctx, "/did_groups/"+url.PathEscape(id), nil, &result); err != nil {
		return DIDGroup{}, err
	}
	return result.Data, nil
}

// ListDIDs lists numbers already owned by the DIDWW account, not available stock.
func (c *Client) ListDIDs(ctx context.Context, page int) (DIDList, error) {
	if page < 1 {
		return DIDList{}, fmt.Errorf("didww page must be positive")
	}
	var result DIDList
	err := c.get(ctx, "/dids", url.Values{"page[number]": {strconv.Itoa(page)}}, &result)
	if err != nil {
		return DIDList{}, err
	}
	return result, nil
}

// GetDID retrieves one DIDWW resource for ownership and state verification.
func (c *Client) GetDID(ctx context.Context, id string) (DID, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, "/?#") {
		return DID{}, fmt.Errorf("didww DID ID is invalid")
	}
	var result struct {
		Data DID `json:"data"`
	}
	if err := c.get(ctx, "/dids/"+url.PathEscape(id), nil, &result); err != nil {
		return DID{}, err
	}
	return result.Data, nil
}
