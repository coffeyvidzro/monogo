package carriers

// ConnectionScope identifies who owns a carrier connection.
type ConnectionScope string

const (
	// ConnectionScopeOrganization is customer-owned BYOC connectivity.
	ConnectionScopeOrganization ConnectionScope = "organization"

	// ConnectionScopePlatform is Leamout-managed upstream connectivity.
	ConnectionScopePlatform ConnectionScope = "platform"
)

func (s ConnectionScope) Valid() bool {
	return s == ConnectionScopeOrganization || s == ConnectionScopePlatform
}
