package carriers

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeCreateRejectsIncompleteDigestAuth(t *testing.T) {
	req := CreateRequest{
		ProviderID:        uuid.New(),
		Name:              "Primary carrier",
		InboundAuthMethod: stringPointer("digest"),
	}
	if err := normalizeCreate(&req); err == nil {
		t.Fatal("expected incomplete digest authentication to be rejected")
	}
}

func TestNormalizeCreateNormalizesAndDeduplicatesCodecs(t *testing.T) {
	req := CreateRequest{
		ProviderID: uuid.New(),
		Name:       "  Primary carrier  ",
		Codecs:     []string{"pcmu", "OPUS", "PCMU"},
	}
	if err := normalizeCreate(&req); err != nil {
		t.Fatalf("normalize create: %v", err)
	}
	if req.Name != "Primary carrier" {
		t.Fatalf("unexpected normalized name %q", req.Name)
	}
	if len(req.Codecs) != 2 || req.Codecs[0] != "PCMU" || req.Codecs[1] != "OPUS" {
		t.Fatalf("unexpected normalized codecs %#v", req.Codecs)
	}
}

func TestNormalizeAuthRejectsCredentialsForIPAuth(t *testing.T) {
	username, secret := "carrier", "secret"
	req := AuthRequest{Method: "ip", Username: &username, Secret: &secret}
	if err := normalizeAuth(&req, true); err == nil {
		t.Fatal("expected credentials with IP authentication to be rejected")
	}
}

func TestParseCIDRMasksHostBits(t *testing.T) {
	prefix, err := parseCIDR("192.0.2.19/24")
	if err != nil {
		t.Fatalf("parse CIDR: %v", err)
	}
	if got := prefix.String(); got != "192.0.2.0/24" {
		t.Fatalf("unexpected masked prefix %q", got)
	}
}

func stringPointer(value string) *string {
	return &value
}
