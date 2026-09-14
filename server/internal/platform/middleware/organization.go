package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/security/authn"
	"github.com/leamout/leamout/internal/security/authz"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

const organizationIDHeader = "X-Organization-ID"

type organizationContext struct {
	ID   uuid.UUID
	Role authz.Role
}

type membershipReader interface {
	GetOrganizationMember(context.Context, sqlc.GetOrganizationMemberParams) (sqlc.OrganizationMember, error)
}

type OrganizationMiddleware struct {
	memberships membershipReader
}

func NewOrganizationMiddleware(memberships ...membershipReader) *OrganizationMiddleware {
	var reader membershipReader
	if len(memberships) > 0 {
		reader = memberships[0]
	}
	return &OrganizationMiddleware{memberships: reader}
}

// RequireAuthenticated returns middleware that accepts either a session cookie
// paired with X-Organization-ID or an organization bearer token.
func (m *OrganizationMiddleware) RequireAuthenticated(authnMiddleware *AuthnMiddleware) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return authnMiddleware.RequireAuthenticated(m.Require(next))
	}
}

func (m *OrganizationMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := authn.PrincipalFromContext(r.Context())
		if !ok {
			httputil.Error(w, apperror.NewUnauthorized("authentication required"))
			return
		}

		organization, err := m.resolveOrganization(r, principal)
		if err != nil {
			httputil.Error(w, err)
			return
		}
		if organization.ID == uuid.Nil {
			httputil.Error(w, apperror.NewBadRequest("organization context required"))
			return
		}

		ctx := withOrganizationContext(r.Context(), organization)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAccess restricts organization routes by membership role for sessions
// and by credential scope for organization tokens. GET and HEAD are reads; all
// other methods require write access.
func (m *OrganizationMiddleware) RequireAccess(resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := authn.PrincipalFromContext(r.Context())
			if !ok {
				httputil.Error(w, apperror.NewUnauthorized("authentication required"))
				return
			}

			access := "write"
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				access = "read"
			}
			required := resource + ":" + access

			if principal.Credential.Type == authn.CredentialOrganizationToken {
				if !hasScope(principal.Scopes, required) {
					httputil.Error(w, apperror.NewForbidden("credential scope required: "+required))
					return
				}
			} else {
				organization, ok := organizationContextFromContext(r.Context())
				ownerOnly := resource == "organization" && r.Method == http.MethodDelete
				if !ok ||
					(access == "write" && organization.Role == authz.RoleMember) ||
					(ownerOnly && organization.Role != authz.RoleOwner) {
					httputil.Error(w, apperror.NewForbidden("organization permission required"))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *OrganizationMiddleware) resolveOrganization(r *http.Request, principal authn.Principal) (organizationContext, error) {
	organizationID, ok := resolveOrganizationID(r, principal)
	if !ok {
		return organizationContext{}, apperror.NewBadRequest("organization context required")
	}
	if principal.Credential.Type == authn.CredentialOrganizationToken {
		return organizationContext{ID: organizationID}, nil
	}
	userID, ok := principal.UserID()
	if !ok || m.memberships == nil {
		return organizationContext{}, apperror.NewUnauthorized("authentication required")
	}
	membership, err := m.memberships.GetOrganizationMember(r.Context(), sqlc.GetOrganizationMemberParams{
		OrganizationID: organizationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return organizationContext{}, apperror.NewNotFound("organization not found")
		}
		return organizationContext{}, apperror.NewInternal("resolve organization membership", err)
	}
	role := authz.Role(membership.Role)
	if !role.IsValid() {
		return organizationContext{}, apperror.NewForbidden("organization membership is invalid")
	}
	return organizationContext{ID: organizationID, Role: role}, nil
}

func hasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if strings.TrimSpace(scope) == required {
			return true
		}
	}
	return false
}

func resolveOrganizationID(r *http.Request, principal authn.Principal) (uuid.UUID, bool) {
	if principal.Credential.Type == authn.CredentialOrganizationToken {
		if principal.OrganizationID == uuid.Nil {
			return uuid.Nil, false
		}
		return principal.OrganizationID, true
	}

	value := r.Header.Get(organizationIDHeader)
	if value == "" {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, false
	}
	if pathValue := chi.URLParam(r, "organization_id"); pathValue != "" {
		pathID, err := uuid.Parse(pathValue)
		if err != nil || pathID != id {
			return uuid.Nil, false
		}
	}

	return id, true
}

type organizationContextKey struct{}

func withOrganizationContext(ctx context.Context, organization organizationContext) context.Context {
	return context.WithValue(ctx, organizationContextKey{}, organization)
}

func OrganizationIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	organization, ok := organizationContextFromContext(ctx)
	if !ok || organization.ID == uuid.Nil {
		return uuid.Nil, false
	}

	return organization.ID, true
}

func organizationContextFromContext(ctx context.Context) (organizationContext, bool) {
	organization, ok := ctx.Value(organizationContextKey{}).(organizationContext)
	return organization, ok && organization.ID != uuid.Nil
}
