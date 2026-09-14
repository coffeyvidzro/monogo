package credentials

import "testing"

func TestValidateScopesAcceptsAPIResourceScopes(t *testing.T) {
	scopes := []string{
		"calls:read",
		"calls:write",
		"numbers:read",
		"carriers:write",
		"webhooks:read",
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
		{name: "credential self-management", scopes: []string{"credentials:write"}},
		{name: "duplicate", scopes: []string{"calls:read", "calls:read"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateScopes(tt.scopes); err == nil {
				t.Fatal("expected invalid scopes to be rejected")
			}
		})
	}
}
