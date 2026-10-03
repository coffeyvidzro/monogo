package platform

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/audit"
	"github.com/coffeyvidzro/monogo/internal/platform/idempotency"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/platform/runtimes"
	"github.com/coffeyvidzro/monogo/internal/platform/storage"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
)

type Module struct {
	Audit       AuditModule
	Idempotency IdempotencyModule
	Runtimes    RuntimesModule
	Storage     StorageModule
	Webhooks    WebhooksModule
}

type AuditModule struct {
	Repository *audit.Repository
	Service    *audit.Service
	Handler    *audit.Handler
}

type IdempotencyModule struct {
	Repository *idempotency.Repository
	Service    *idempotency.Service
	Middleware *middleware.IdempotencyMiddleware
}

type RuntimesModule struct {
	Repository *runtimes.Repository
	Service    *runtimes.Service
	Handler    *runtimes.Handler
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

	idempotencyRepository := idempotency.NewRepository(queries)
	idempotencyService := idempotency.NewService(
		idempotencyRepository,
		idempotency.DefaultConfig(),
	)

	runtimesRepository := runtimes.NewRepository(queries)
	runtimesService := runtimes.NewService(runtimesRepository)

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
		Idempotency: IdempotencyModule{
			Repository: idempotencyRepository,
			Service:    idempotencyService,
			Middleware: middleware.NewIdempotencyMiddleware(idempotencyService),
		},
		Runtimes: RuntimesModule{
			Repository: runtimesRepository,
			Service:    runtimesService,
			Handler:    runtimes.NewHandler(runtimesService),
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
