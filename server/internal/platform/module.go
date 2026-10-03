package platform

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"net/netip"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/audit"
	"github.com/coffeyvidzro/monogo/internal/platform/entitlements"
	"github.com/coffeyvidzro/monogo/internal/platform/idempotency"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/platform/networking"
	"github.com/coffeyvidzro/monogo/internal/platform/storage"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/internal/security/scim"
)

type Module struct {
	Audit        AuditModule
	Entitlements EntitlementsModule
	Idempotency  IdempotencyModule
	Networking   NetworkingModule
	Storage      StorageModule
	SCIM         SCIMModule
	Webhooks     WebhooksModule
}

type AuditModule struct {
	Repository *audit.Repository
	Service    *audit.Service
	Handler    *audit.Handler
}

type EntitlementsModule struct {
	Repository *entitlements.Repository
	Service    *entitlements.Service
	Middleware *entitlements.Middleware
}

type IdempotencyModule struct {
	Repository *idempotency.Repository
	Service    *idempotency.Service
	Middleware *middleware.IdempotencyMiddleware
}

type NetworkingModule struct {
	Repository *networking.Repository
	Service    *networking.Service
	Handler    *networking.Handler
	Middleware *networking.Middleware
}

type StorageModule struct {
	Repository *storage.Repository
	Service    *storage.Service
	Handler    *storage.Handler
}

type SCIMModule struct {
	Repository *scim.Repository
	Service    *scim.Service
	Handler    *scim.Handler
	Middleware *scim.Middleware
}

type WebhooksModule struct {
	Repository *webhooks.Repository
	Service    *webhooks.Service
	Handler    *webhooks.Handler
}

func New(
	db *pgxpool.Pool,
	queries *sqlc.Queries,
	credentialCipher *encryption.Cipher,
	trustedProxies []netip.Prefix,
) *Module {
	auditRepository := audit.NewRepository(db)
	auditService := audit.NewService(auditRepository)

	entitlementsRepository := entitlements.NewRepository(queries)
	entitlementsService := entitlements.NewService(entitlementsRepository)

	idempotencyRepository := idempotency.NewRepository(queries)
	idempotencyService := idempotency.NewService(
		idempotencyRepository,
		idempotency.DefaultConfig(),
	)

	networkingRepository := networking.NewRepository(queries)
	networkingService := networking.NewService(networkingRepository)

	storageRepository := storage.NewRepository(queries)
	storageService := storage.NewService(storageRepository, credentialCipher)

	scimRepository := scim.NewRepository(queries)
	scimService := scim.NewService(scimRepository)

	webhooksRepository := webhooks.NewRepository(queries)
	webhooksService := webhooks.NewService(webhooksRepository)

	return &Module{
		Audit: AuditModule{
			Repository: auditRepository,
			Service:    auditService,
			Handler:    audit.NewHandler(auditService),
		},
		Entitlements: EntitlementsModule{
			Repository: entitlementsRepository,
			Service:    entitlementsService,
			Middleware: entitlements.NewMiddleware(entitlementsService),
		},
		Idempotency: IdempotencyModule{
			Repository: idempotencyRepository,
			Service:    idempotencyService,
			Middleware: middleware.NewIdempotencyMiddleware(idempotencyService),
		},
		Networking: NetworkingModule{
			Repository: networkingRepository,
			Service:    networkingService,
			Handler:    networking.NewHandler(networkingService),
			Middleware: networking.NewMiddleware(
				entitlementsService,
				networkingService,
				trustedProxies,
			),
		},
		Storage: StorageModule{
			Repository: storageRepository,
			Service:    storageService,
			Handler:    storage.NewHandler(storageService),
		},
		SCIM: SCIMModule{
			Repository: scimRepository,
			Service:    scimService,
			Handler:    scim.NewHandler(scimService),
			Middleware: scim.NewMiddleware(scimService),
		},
		Webhooks: WebhooksModule{
			Repository: webhooksRepository,
			Service:    webhooksService,
			Handler:    webhooks.NewHandler(webhooksService),
		},
	}
}
