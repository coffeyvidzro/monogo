package platform

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/coffeyvidzro/monogo/internal/platform/audit"
	"github.com/coffeyvidzro/monogo/internal/platform/entitlements"
	"github.com/coffeyvidzro/monogo/internal/platform/networking"
	"github.com/coffeyvidzro/monogo/internal/platform/retention"
	"github.com/coffeyvidzro/monogo/internal/platform/storage"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
	"github.com/coffeyvidzro/monogo/internal/security/scim"
	"github.com/coffeyvidzro/monogo/internal/security/sso"
)

func RegisterRoutes(
	router chi.Router,
	module *Module,
	organizationAccess func(string) func(http.Handler) http.Handler,
) {
	webhooks.RegisterRoutes(
		router,
		module.Webhooks.Handler,
		organizationAccess("webhooks"),
	)

	audit.RegisterRoutes(
		router,
		module.Audit.Handler,
		organizationAccess("audit"),
	)

	storage.RegisterRoutes(
		router,
		module.Storage.Handler,
		organizationAccess("storage"),
	)

	networking.RegisterRoutes(
		router,
		module.Networking.Handler,
		organizationAccess("networking"),
		module.Entitlements.Middleware.Require(entitlements.CapabilityPrivateNetworking),
	)

	sso.RegisterRoutes(
		router,
		module.SSO.Handler,
		organizationAccess("sso"),
		module.Entitlements.Middleware.Require(entitlements.CapabilitySSO),
	)

	scim.RegisterManagementRoutes(
		router,
		module.SCIM.Handler,
		organizationAccess("scim"),
		module.Entitlements.Middleware.Require(entitlements.CapabilitySCIM),
	)
	scim.RegisterProvisioningRoutes(
		router,
		module.SCIM.Handler,
		module.SCIM.Middleware.RequireToken,
		scim.RequireCapability(func(ctx context.Context, organizationID uuid.UUID) error {
			return module.Entitlements.Service.Require(ctx, organizationID, entitlements.CapabilitySCIM)
		}),
	)

	retention.RegisterRoutes(
		router,
		module.Retention.Handler,
		organizationAccess("retention"),
		module.Entitlements.Middleware.Require(entitlements.CapabilityRetentionPolicies),
	)
}
