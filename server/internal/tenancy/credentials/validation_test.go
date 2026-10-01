package credentials

import "testing"

func TestValidateScopesAcceptsAPIResourceScopes(t *testing.T) {
	scopes := []string{
		"calls:read",
		"calls:write",
		"numbers:read",
		"webhooks:read",
		"voice-agents:read",
		"voice-agents:write",
		"trunks:read",
		"trunks:write",
		"recordings:write",
		"webrtc:read",
	}
	if err := ValidateScopes(scopes); err != nil {
		t.Fatalf("expected API resource scopes to be valid: %v", err)
	}
}

func TestValidateScopesRejectsUnknownAndDuplicateScopes(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
	}{
		{name: "unknown", scopes: []string{"everything:write"}},
		{
			name:   "credential self-management",
			scopes: []string{"credentials:write"},
		},
		{
			name:   "duplicate",
			scopes: []string{"calls:read", "calls:read"},
		},
		{
			name:   "removed carrier scope",
			scopes: []string{"carriers:read"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateScopes(test.scopes); err == nil {
				t.Fatal("expected invalid scopes to be rejected")
			}
		})
	}
}
