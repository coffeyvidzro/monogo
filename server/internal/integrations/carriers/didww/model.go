package didww

import (
	"net/http"
)

// Config contains platform-managed DIDWW credentials.
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// DIDGroup identifies DIDWW coverage. Availability does not reserve a number.
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
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
}

// DID identifies a provider-owned resource, not a Leamout phone-number record.
type DID struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Number               string `json:"number"`
		Blocked              bool   `json:"blocked"`
		Terminated           bool   `json:"terminated"`
		AwaitingRegistration bool   `json:"awaiting_registration"`
	} `json:"attributes"`
}

type DIDList struct {
	Data  []DID `json:"data"`
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
}
