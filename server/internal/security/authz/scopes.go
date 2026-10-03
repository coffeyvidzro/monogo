package authz

// Scope limits the capabilities exposed by a credential.
type Scope string

const (
	ScopeOrganizationRead Scope = "organization:read"
	ScopeMembersRead      Scope = "members:read"
	ScopeMembersWrite     Scope = "members:write"
	// Credential lifecycle writes intentionally have no token scope. Creating,
	// updating, and revoking organization tokens requires an owner/admin session
	// so a compromised token cannot mint a more privileged replacement.
	ScopeCredentialsRead  Scope = "credentials:read"
	ScopeVoiceAgentsRead  Scope = "voice-agents:read"
	ScopeVoiceAgentsWrite Scope = "voice-agents:write"
	ScopeCallsRead        Scope = "calls:read"
	ScopeCallsWrite       Scope = "calls:write"
	ScopeRecordingsRead   Scope = "recordings:read"
	ScopeRecordingsWrite  Scope = "recordings:write"
	ScopeStorageRead      Scope = "storage:read"
	ScopeStorageWrite     Scope = "storage:write"
	ScopeNumbersRead      Scope = "numbers:read"
	ScopeNumbersWrite     Scope = "numbers:write"
	ScopeTrunksRead       Scope = "trunks:read"
	ScopeTrunksWrite      Scope = "trunks:write"
	ScopeWebhooksRead     Scope = "webhooks:read"
	ScopeWebhooksWrite    Scope = "webhooks:write"
	ScopeAuditRead        Scope = "audit:read"
	ScopeAuditWrite       Scope = "audit:write"
	ScopeWebRTCRead       Scope = "webrtc:read"
	ScopeWebRTCWrite      Scope = "webrtc:write"
	ScopeNetworkingRead   Scope = "networking:read"
	ScopeNetworkingWrite  Scope = "networking:write"
)

func (s Scope) IsValid() bool {
	switch s {
	case ScopeOrganizationRead,
		ScopeMembersRead,
		ScopeMembersWrite,
		ScopeCredentialsRead,
		ScopeVoiceAgentsRead, ScopeVoiceAgentsWrite,
		ScopeCallsRead, ScopeCallsWrite,
		ScopeRecordingsRead, ScopeRecordingsWrite,
		ScopeStorageRead, ScopeStorageWrite,
		ScopeNumbersRead, ScopeNumbersWrite,
		ScopeTrunksRead, ScopeTrunksWrite,
		ScopeWebhooksRead, ScopeWebhooksWrite,
		ScopeAuditRead, ScopeAuditWrite,
		ScopeWebRTCRead, ScopeWebRTCWrite,
		ScopeNetworkingRead, ScopeNetworkingWrite:
		return true
	default:
		return false
	}
}
