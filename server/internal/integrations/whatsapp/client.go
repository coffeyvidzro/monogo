// Package whatsapp adapts the official WhatsApp Business Platform HTTP API.
package whatsapp

import (
	"net/http"
	"strings"
)

type Client struct {
	config Config
	http   *http.Client
}

func New(config Config, client *http.Client) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: config.Timeout}
	}
	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	return &Client{config: config, http: client}, nil
}
