package platform

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/audit"
	"github.com/coffeyvidzro/monogo/internal/platform/entitlements"
	"github.com/coffeyvidzro/monogo/internal/platform/idempotency"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/platform/networking"
	"github.com/coffeyvidzro/monogo/internal/platform/storage"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
)

type Module struct {
	Audit        AuditModule
	Entitlements EntitlementsModule
	Idempotency  IdempotencyModule
	Networking   NetworkingModule
	Storage      StorageModule
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
}

type StorageModule struct {
	Repository *storage.Repository
	Service    *storage.Service
	Handler    *storage.Handler
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
		},
		Storage: StorageModule{
			Repository: storageRepository,
			Service:    storageService,
			Handler:    storage.NewHandler(storageService),
		},
		Webhooks: WebhooksModule{
			Repository: webhooksRepository,
			Service:    webhooksService,
			Handler:    webhooks.NewHandler(webhooksService),
		},
	}
}
