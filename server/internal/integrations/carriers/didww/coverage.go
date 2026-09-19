package didww

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// DID groups describe number coverage, availability and the catalogue used
// to select the DIDWW product before an individual number is chosen.
type DIDGroup struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Prefix   string   `json:"prefix"`
		AreaName string   `json:"area_name"`
		Features []string `json:"features"`
	} `json:"attributes"`
	Meta struct {
		IsAvailable          bool `json:"is_available"`
		AvailableDIDsEnabled bool `json:"available_dids_enabled"`
		NeedsRegistration    bool `json:"needs_registration"`
		TotalCount           int  `json:"total_count"`
	} `json:"meta"`
}

type DIDGroupList struct {
	Data  []DIDGroup `json:"data"`
	Meta  Meta       `json:"meta"`
	Links Links      `json:"links"`
}

// ListDIDGroups lists geographic number groups. Groups are paginated.
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

func (c *Client) GetDIDGroup(ctx context.Context, id string) (DIDGroup, error) {
	path, err := resourcePath("/did_groups", id)
	if err != nil {
		return DIDGroup{}, err
	}
	var result single[DIDGroup]
	if err := c.get(ctx, path, nil, &result); err != nil {
		return DIDGroup{}, err
	}
	return result.Data, nil
}
