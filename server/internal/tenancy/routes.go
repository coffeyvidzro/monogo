package tenancy

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/tenancy/credentials"
	"github.com/coffeyvidzro/monogo/internal/tenancy/members"
	"github.com/coffeyvidzro/monogo/internal/tenancy/organization"
	"github.com/go-chi/chi/v5"
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
