package middleware

import (
	"net/http"

	"github.com/leamout/leamout/internal/security/authn"
	"github.com/leamout/leamout/internal/security/authz"
)

// RequirePermission returns middleware that requires an authenticated
// principal with the requested permission.
func RequirePermission(permission authz.Permission) func(http.Handler) http.Handler {
	access := authz.Access{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := authn.PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			accessPrincipal := authz.Principal{Identity: principal}
			if !access.Allows(accessPrincipal, permission) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
