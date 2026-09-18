package platform

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/audit"
	"github.com/coffeyvidzro/monogo/internal/platform/idempotency"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
)

type Module struct {
	Audit       AuditModule
	Idempotency IdempotencyModule
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

type WebhooksModule struct {
	Repository *webhooks.Repository
	Service    *webhooks.Service
	Handler    *webhooks.Handler
}

func New(db *pgxpool.Pool, queries *sqlc.Queries) *Module {
	auditRepository := audit.NewRepository(db)
	auditService := audit.NewService(auditRepository)

	idempotencyRepository := idempotency.NewRepository(queries)
	idempotencyService := idempotency.NewService(
		idempotencyRepository,
		idempotency.DefaultConfig(),
	)

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
		Webhooks: WebhooksModule{
			Repository: webhooksRepository,
			Service:    webhooksService,
			Handler:    webhooks.NewHandler(webhooksService),
		},
	}
}
