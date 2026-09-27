package smpp

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("smpp.example.com", 2775, "system", "secret")
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if config.Address() != "smpp.example.com:2775" || config.BindMode != BindTransceiver {
		t.Fatalf("config = %#v", config)
	}
}

func TestConfigRejectsInvalidBindAndEnquireLink(t *testing.T) {
	config := DefaultConfig("smpp.example.com", 2775, "system", "secret")
	config.BindMode = "invalid"
	if config.Validate() == nil {
		t.Fatal("invalid bind mode accepted")
	}
	config.BindMode = BindTransceiver
	config.EnquireLink = time.Second
	if config.Validate() == nil {
		t.Fatal("invalid enquire_link accepted")
	}
}

func TestReceiverBindCannotSubmit(t *testing.T) {
	client, err := New(func() Config {
		c := DefaultConfig("smpp.example.com", 2775, "system", "secret")
		c.BindMode = BindReceiver
		return c
	}())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.Submit(t.Context(), SubmitRequest{
		From: "1",
		To:   "2",
		Text: "hello",
	})
	if err != ErrSubmitUnsupported {
		t.Fatalf("Submit() error = %v", err)
	}
}
