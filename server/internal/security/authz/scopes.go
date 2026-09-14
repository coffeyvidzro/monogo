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
	ScopeCredentialsRead        Scope = "credentials:read"
	ScopeLicensingRead          Scope = "licensing:read"
	ScopeLicensingWrite         Scope = "licensing:write"
	ScopeCommercialStateRead    Scope = "commercial-state:read"
	ScopeCommercialStateWrite   Scope = "commercial-state:write"
	ScopeVoiceApplicationsRead  Scope = "voice-applications:read"
	ScopeVoiceApplicationsWrite Scope = "voice-applications:write"
	ScopeCallsRead              Scope = "calls:read"
	ScopeCallsWrite             Scope = "calls:write"
	ScopeRecordingsRead         Scope = "recordings:read"
	ScopeRecordingsWrite        Scope = "recordings:write"
	ScopeSubscribersRead        Scope = "subscribers:read"
	ScopeSubscribersWrite       Scope = "subscribers:write"
	ScopeNumbersRead            Scope = "numbers:read"
	ScopeNumbersWrite           Scope = "numbers:write"
	ScopeSIPDomainsRead         Scope = "sip-domains:read"
	ScopeSIPDomainsWrite        Scope = "sip-domains:write"
	ScopeTrunksRead             Scope = "trunks:read"
	ScopeTrunksWrite            Scope = "trunks:write"
	ScopeCarriersRead           Scope = "carriers:read"
	ScopeCarriersWrite          Scope = "carriers:write"
	ScopeWebhooksRead           Scope = "webhooks:read"
	ScopeWebhooksWrite          Scope = "webhooks:write"
	ScopeAuditRead              Scope = "audit:read"
	ScopeAuditWrite             Scope = "audit:write"
	ScopeConferencesRead        Scope = "conferences:read"
	ScopeConferencesWrite       Scope = "conferences:write"
	ScopeRealtimeRead           Scope = "realtime:read"
	ScopeRealtimeWrite          Scope = "realtime:write"
)

func (s Scope) IsValid() bool {
	switch s {
	case ScopeOrganizationRead,
		ScopeMembersRead,
		ScopeMembersWrite,
		ScopeCredentialsRead,
		ScopeLicensingRead, ScopeLicensingWrite,
		ScopeCommercialStateRead, ScopeCommercialStateWrite,
		ScopeVoiceApplicationsRead, ScopeVoiceApplicationsWrite,
		ScopeCallsRead, ScopeCallsWrite,
		ScopeRecordingsRead, ScopeRecordingsWrite,
		ScopeSubscribersRead, ScopeSubscribersWrite,
		ScopeNumbersRead, ScopeNumbersWrite,
		ScopeSIPDomainsRead, ScopeSIPDomainsWrite,
		ScopeTrunksRead, ScopeTrunksWrite,
		ScopeCarriersRead, ScopeCarriersWrite,
		ScopeWebhooksRead, ScopeWebhooksWrite,
		ScopeAuditRead, ScopeAuditWrite,
		ScopeConferencesRead, ScopeConferencesWrite,
		ScopeRealtimeRead, ScopeRealtimeWrite:
		return true
	default:
		return false
	}
}
