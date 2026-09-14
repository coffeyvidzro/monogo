package identity

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leamout/leamout/internal/identity/auth"
	"github.com/leamout/leamout/internal/identity/session"
	"github.com/leamout/leamout/internal/identity/users"
)

func RegisterRoutes(
	router chi.Router,
	module *Module,
	requireSession func(http.Handler) http.Handler,
) {
	auth.RegisterRoutes(
		router,
		module.Auth.Handler,
		requireSession,
	)

	session.RegisterRoutes(
		router,
		module.Session.Handler,
		requireSession,
	)

	users.RegisterRoutes(
		router,
		module.Users.Handler,
		requireSession,
	)
}
