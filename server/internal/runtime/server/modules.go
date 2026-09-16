package server

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/identity"
	"github.com/coffeyvidzro/monogo/internal/integrations/postgres"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/platform"
	"github.com/coffeyvidzro/monogo/internal/platform/config"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/security/authn"
	"github.com/coffeyvidzro/monogo/internal/tenancy"
)

type modules struct {
	postgres             *postgres.Client
	redis                *redisintegration.Client
	identity             *identity.Module
	tenancy              *tenancy.Module
	platform             *platform.Module
	authn                 *middleware.AuthnMiddleware
	organizationsContext *middleware.OrganizationMiddleware
	rateLimit             *middleware.RateLimitMiddleware
}

func newModules(ctx context.Context, cfg config.Config) (*modules, error) {
	postgresClient, err := postgres.New(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		return nil, fmt.Errorf("initialize PostgreSQL: %w", err)
	}

	redisClient, err := redisintegration.New(ctx, redisintegration.DefaultConfig(cfg.RedisURL))
	if err != nil {
		postgresClient.Close()
		return nil, fmt.Errorf("initialize Redis: %w", err)
	}

	queries := sqlc.New(postgresClient.Pool())
	identityModule := identity.New(queries)
	tenancyModule := tenancy.New(queries)
	platformModule := platform.New(postgresClient.Pool(), queries)

	resolver := authn.NewResolver(identityModule.Session.Service, tenancyModule.Credentials.Service)
	authMiddleware := middleware.NewAuthnMiddleware(resolver)
	organizationMiddleware := middleware.NewOrganizationMiddleware(queries)

	rateLimitStore, err := redisClient.NewRateLimitStore()
	if err != nil {
		_ = redisClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("initialize rate limit store: %w", err)
	}
	rateLimitMiddleware, err := middleware.NewRateLimitMiddleware(rateLimitStore)
	if err != nil {
		_ = redisClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("initialize rate limit middleware: %w", err)
	}

	return &modules{
		postgres:             postgresClient,
		redis:                redisClient,
		identity:             identityModule,
		tenancy:              tenancyModule,
		platform:             platformModule,
		authn:                 authMiddleware,
		organizationsContext: organizationMiddleware,
		rateLimit:             rateLimitMiddleware,
	}, nil
}

func (m *modules) close(logger *logging.Logger) {
	if m == nil {
		return
	}
	if m.redis != nil {
		if err := m.redis.Close(); err != nil {
			logger.Warn(context.Background(), "close Redis", "error", err)
		}
	}
	if m.postgres != nil {
		m.postgres.Close()
	}
}
