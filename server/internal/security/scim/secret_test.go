package scim

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGeneratedTokenStoresOnlyHashAndPrefix(t *testing.T) {
	secret, prefix, hash, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}
	if !strings.HasPrefix(secret, tokenPrefix) || !strings.HasPrefix(secret, prefix+"_") {
		t.Fatalf("generated token prefix = %q", prefix)
	}
	if hash == secret || hashToken(secret) != hash {
		t.Fatal("generated token hash is invalid")
	}
	payload, err := json.Marshal(Token{
		Name:   "Directory",
		Prefix: prefix,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), secret) || strings.Contains(string(payload), hash) {
		t.Fatalf("token response contains secret material: %s", payload)
	}
}
