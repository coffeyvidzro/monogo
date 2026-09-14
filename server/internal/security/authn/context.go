package authn

import (
	"context"

	"github.com/google/uuid"
)

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	if !ok || !principal.IsValid() {
		return Principal{}, false
	}

	return principal, true
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return uuid.Nil, false
	}

	return principal.UserID()
}

func OrganizationIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok || principal.OrganizationID == uuid.Nil {
		return uuid.Nil, false
	}

	return principal.OrganizationID, true
}

func CredentialScopesFromContext(ctx context.Context) ([]string, bool) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok || principal.Credential.Type != CredentialOrganizationToken {
		return nil, false
	}

	return append([]string(nil), principal.Scopes...), true
}
