package sso

import (
	"encoding/json"
	"testing"
)

func TestNormalizeCreateAcceptsOIDCConfiguration(t *testing.T) {
	secret := "client-secret"
	req := CreateRequest{
		Name:          "  Workforce SSO  ",
		Protocol:      "OIDC",
		Issuer:        "https://id.example.com/tenant",
		Configuration: json.RawMessage(`{"client_id":"leamout"}`),
		Secret:        &secret,
	}
	if err := normalizeCreate(&req); err != nil {
		t.Fatalf("normalizeCreate() error = %v", err)
	}
	if req.Name != "Workforce SSO" || req.Protocol != ProtocolOIDC {
		t.Fatalf("normalized request = %#v", req)
	}
}

func TestNormalizeCreateRejectsInsecureIssuer(t *testing.T) {
	req := CreateRequest{
		Name:          "Workforce SSO",
		Protocol:      ProtocolOIDC,
		Issuer:        "http://id.example.com",
		Configuration: json.RawMessage(`{}`),
	}
	if err := normalizeCreate(&req); err == nil {
		t.Fatal("normalizeCreate() expected insecure issuer error")
	}
}

func TestNormalizeCreateRejectsNonObjectConfiguration(t *testing.T) {
	req := CreateRequest{
		Name:          "Workforce SSO",
		Protocol:      ProtocolSAML,
		Issuer:        "https://id.example.com/metadata",
		Configuration: json.RawMessage(`[]`),
	}
	if err := normalizeCreate(&req); err == nil {
		t.Fatal("normalizeCreate() expected configuration error")
	}
}

func TestNormalizeUpdateRejectsEmptySecret(t *testing.T) {
	secret := "   "
	req := UpdateRequest{Secret: &secret}
	if err := normalizeUpdate(&req); err == nil {
		t.Fatal("normalizeUpdate() expected empty secret error")
	}
}

func TestNormalizeUpdateRejectsUnknownStatus(t *testing.T) {
	status := "paused"
	req := UpdateRequest{Status: &status}
	if err := normalizeUpdate(&req); err == nil {
		t.Fatal("normalizeUpdate() expected status error")
	}
}
