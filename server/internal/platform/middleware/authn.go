package middleware

import (
	"net/http"
	"strings"

	"github.com/leamout/leamout/internal/security/authn"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

type AuthnMiddleware struct {
	resolver *authn.Resolver
}

func NewAuthnMiddleware(resolver *authn.Resolver) *AuthnMiddleware {
	return &AuthnMiddleware{
		resolver: resolver,
	}
}

func (m *AuthnMiddleware) RequireAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, bearerAttempted, ok := m.resolveBearerToken(r)
		if bearerAttempted && !ok {
			httputil.Error(w, apperror.NewUnauthorized("authentication required"))
			return
		}
		if !ok {
			principal, ok = m.resolveSessionCookie(r)
		}
		if !ok {
			httputil.Error(w, apperror.NewUnauthorized("authentication required"))
			return
		}

		ctx := authn.WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthnMiddleware) RequireOrganizationToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, _, ok := m.resolveBearerToken(r)
		if !ok {
			httputil.Error(w, apperror.NewUnauthorized("organization token required"))
			return
		}

		ctx := authn.WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthnMiddleware) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := m.resolveSessionCookie(r)
		if !ok {
			httputil.Error(w, apperror.NewUnauthorized("authentication required"))
			return
		}

		ctx := authn.WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthnMiddleware) resolveSessionCookie(r *http.Request) (authn.Principal, bool) {
	cookie, err := r.Cookie(authn.SessionCookieName)
	if err != nil || cookie.Value == "" {
		return authn.Principal{}, false
	}

	principal, err := m.resolver.Resolve(r.Context(), authn.CredentialInput{
		Type:  authn.CredentialSession,
		Value: cookie.Value,
	})
	if err != nil {
		return authn.Principal{}, false
	}

	return principal, true
}

func (m *AuthnMiddleware) resolveBearerToken(r *http.Request) (authn.Principal, bool, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return authn.Principal{}, false, false
	}

	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return authn.Principal{}, false, false
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return authn.Principal{}, true, false
	}

	principal, err := m.resolver.Resolve(r.Context(), authn.CredentialInput{
		Type:  authn.CredentialOrganizationToken,
		Value: token,
	})
	if err != nil {
		return authn.Principal{}, true, false
	}

	return principal, true, true
}
