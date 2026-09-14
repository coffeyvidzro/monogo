package authz

import (
	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/security/authn"
)

// Principal is the authorization view of an authenticated caller.
// Authentication owns identity and credential provenance; authorization adds
// the resource-specific role and any credential-imposed scopes.
type Principal struct {
	Identity authn.Principal
	Role     Role
	Scopes   []Scope
}

// Access evaluates authorization decisions for principals.
type Access struct{}

func (Access) Allows(principal Principal, permission Permission) bool {
	if !principal.Valid() {
		return false
	}
	return Allows(principal.Role, permission)
}

func (Access) AllowsScoped(principal Principal, permission Permission, scope Scope) bool {
	if !principal.Valid() || !Allows(principal.Role, permission) {
		return false
	}
	if principal.Scopes == nil {
		return true
	}
	return HasScope(principal.Scopes, scope)
}

func (p Principal) Valid() bool {
	return p.Identity.Subject.ID != uuid.Nil && p.Identity.Subject.Type != ""
}
