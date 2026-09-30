package carriers

import (
	"encoding/base64"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/google/uuid"
)

func TestCarrierCredentialCiphertextIsBoundToTenantResourceAndDirection(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	cipher, err := encryption.New(key)
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	organizationID, connectionID := uuid.New(), uuid.New()
	scope := credentialScope(organizationID, connectionID, "outbound")
	ciphertext, err := cipher.EncryptForScope(scope, "carrier-secret")
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}
	plaintext, err := cipher.DecryptForScope(scope, ciphertext)
	if err != nil {
		t.Fatalf("decrypt credential: %v", err)
	}
	if plaintext != "carrier-secret" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
	if _, err := cipher.DecryptForScope(credentialScope(organizationID, connectionID, "inbound"), ciphertext); err == nil {
		t.Fatal("expected ciphertext copied to another credential direction to fail authentication")
	}
	if _, err := cipher.DecryptForScope(credentialScope(uuid.New(), connectionID, "outbound"), ciphertext); err == nil {
		t.Fatal("expected ciphertext copied to another tenant to fail authentication")
	}
}
