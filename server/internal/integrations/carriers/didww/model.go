package didww

import (
	"encoding/json"
	"net/http"
	"time"
)

// Config contains Leamout-managed credentials. The transport is not used for SIP calls.
type Config struct {
	APIKey     string
	BaseURL    string
	APIVersion string
	HTTPClient *http.Client
}

type ResourceIdentifier struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Relationship struct {
	Data *ResourceIdentifier `json:"data"`
}

type Meta struct {
	APIVersion   string `json:"api_version"`
	TotalRecords int    `json:"total_records,omitempty"`
}

type Links struct {
	First string `json:"first,omitempty"`
	Last  string `json:"last,omitempty"`
	Next  string `json:"next,omitempty"`
	Prev  string `json:"prev,omitempty"`
}

type single[T any] struct {
	Data T    `json:"data"`
	Meta Meta `json:"meta"`
}

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

type AvailableDID struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Number string `json:"number"`
	} `json:"attributes"`
	Relationships map[string]json.RawMessage `json:"relationships,omitempty"`
}

type AvailableDIDList struct {
	Data []AvailableDID `json:"data"`
	Meta struct {
		APIVersion string `json:"api_version"`
		TotalCount int    `json:"total_count"`
	} `json:"meta"`
}

type DID struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Number                 string     `json:"number"`
		Blocked                bool       `json:"blocked"`
		Terminated             bool       `json:"terminated"`
		AwaitingRegistration   bool       `json:"awaiting_registration"`
		BillingCyclesCount     *int       `json:"billing_cycles_count"`
		ExpiresAt              *time.Time `json:"expires_at"`
		ChannelsIncludedCount  int        `json:"channels_included_count"`
		DedicatedChannelsCount int        `json:"dedicated_channels_count"`
	} `json:"attributes"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
}

type DIDList struct {
	Data  []DID `json:"data"`
	Meta  Meta  `json:"meta"`
	Links Links `json:"links"`
}
