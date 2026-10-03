package providers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCredentialResponseNeverSerializesSecretMaterial(t *testing.T) {
	response := credentialResponse(
		Credential{
			ID:              uuid.New(),
			OrganizationID:  uuid.New(),
			Provider:        ProviderDeepgram,
			Name:            "production",
			ConnectionState: ConnectionUnchecked,
		},
		[]uuid.UUID{
			uuid.New(),
		},
	)
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	serialized := string(payload)
	for _, forbidden := range []string{
		"secret",
		"ciphertext",
		"api_key",
	} {
		if strings.Contains(strings.ToLower(serialized), forbidden) {
			t.Fatalf("response contains forbidden field %q: %s", forbidden, serialized)
		}
	}
	if !strings.Contains(serialized, `"capabilities"`) ||
		!strings.Contains(serialized, `"voice_agent_ids"`) {
		t.Fatalf("response is missing integration metadata: %s", serialized)
	}
}
