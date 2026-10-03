package scim

import (
	"context"
	"net/http"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

type principalKey struct{}

type authenticator interface {
	Authenticate(context.Context, string) (Principal, error)
}

type Middleware struct{ service authenticator }

func NewMiddleware(service authenticator) *Middleware {
	return &Middleware{
		service: service,
	}
}

func (m *Middleware) RequireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, secret, ok := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(secret) == "" {
			writeSCIMError(w, apperror.NewUnauthorized("SCIM bearer token required"))
			return
		}
		principal, err := m.service.Authenticate(r.Context(), strings.TrimSpace(secret))
		if err != nil {
			writeSCIMError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), principalKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireCapability(require func(context.Context, uuid.UUID) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				writeSCIMError(w, apperror.NewUnauthorized("SCIM bearer token required"))
				return
			}
			if err := require(r.Context(), principal.OrganizationID); err != nil {
				writeSCIMError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok && principal.TokenID != uuid.Nil && principal.OrganizationID != uuid.Nil
}
