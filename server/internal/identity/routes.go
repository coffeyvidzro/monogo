package identity

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/identity/auth"
	"github.com/coffeyvidzro/monogo/internal/identity/session"
	"github.com/coffeyvidzro/monogo/internal/identity/users"
	"github.com/go-chi/chi/v5"
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
