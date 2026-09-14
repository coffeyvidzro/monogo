package tenancy

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leamout/leamout/internal/tenancy/credentials"
	"github.com/leamout/leamout/internal/tenancy/members"
	"github.com/leamout/leamout/internal/tenancy/organization"
)

func RegisterRoutes(
	router chi.Router,
	module *Module,
	requireSession func(http.Handler) http.Handler,
	organizationContextAccess func(string) func(http.Handler) http.Handler,
	sessionOrganizationAccess func(string) func(http.Handler) http.Handler,
) {
	organization.RegisterRoutes(
		router,
		module.Organizations.Handler,
		requireSession,
		organizationContextAccess("organization"),
	)

	members.RegisterRoutes(
		router,
		module.Members.Handler,
		sessionOrganizationAccess("members"),
	)

	credentials.RegisterRoutes(
		router,
		module.Credentials.Handler,
		sessionOrganizationAccess("credentials"),
	)
}
