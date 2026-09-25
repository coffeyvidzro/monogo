package server

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/commercial"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/identity"
	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
	"github.com/coffeyvidzro/monogo/internal/integrations/coturn"
	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/integrations/minio"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/paystack"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
	"github.com/coffeyvidzro/monogo/internal/integrations/postgres"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/platform"
	"github.com/coffeyvidzro/monogo/internal/platform/config"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/platform/metrics"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/coffeyvidzro/monogo/internal/security/authn"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/internal/telecom"
	"github.com/coffeyvidzro/monogo/internal/telecom/conferences"
	"github.com/coffeyvidzro/monogo/internal/telecom/realtime"
	"github.com/coffeyvidzro/monogo/internal/telecom/recordings"
	"github.com/coffeyvidzro/monogo/internal/tenancy"
)

type modules struct {
	postgres             *postgres.Client
	redis                *redisintegration.Client
	freeSwitch           *freeswitch.Client
	commercial           *commercial.Module
	identity             *identity.Module
	tenancy              *tenancy.Module
	platform             *platform.Module
	telecom              *telecom.Module
	authn                *middleware.AuthnMiddleware
	organizationsContext *middleware.OrganizationMiddleware
	rateLimit            *middleware.RateLimitMiddleware
	metrics              *metrics.Registry
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

	freeSwitch, err := freeswitch.New(
		freeswitch.DefaultConfig(cfg.FreeSWITCHESLAddress, cfg.FreeSWITCHESLPassword),
	)
	if err != nil {
		_ = redisClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("initialize FreeSWITCH: %w", err)
	}
	if err := freeSwitch.Connect(ctx); err != nil {
		_ = freeSwitch.Close()
		_ = redisClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("connect FreeSWITCH: %w", err)
	}

	closeDependencies := func() {
		_ = freeSwitch.Close()
		_ = redisClient.Close()
		postgresClient.Close()
	}

	credentialCipher, err := encryption.New(cfg.EncryptionKey)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize carrier credential encryption: %w", err)
	}

	coturnClient, err := coturn.New(coturn.Config{
		AuthSecret: cfg.TURNAuthSecret,
		URLs:       cfg.TURNPublicURLs,
	})
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize Coturn integration: %w", err)
	}
	turnService, err := realtime.NewService(coturnClient, redisClient)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize TURN credentials: %w", err)
	}

	objectClient, err := minio.New(ctx, minio.DefaultConfig(
		cfg.Domain, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey,
	))
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize recording object storage: %w", err)
	}
	recordingStorage := recordings.NewObjectStorage(objectClient)

	// The DIDWW client is Leamout-owned; customer-provided carrier credentials
	// must never be used for managed DID inventory or purchase operations.
	didwwInventory, err := didww.New(didww.Config{
		APIKey:  cfg.DIDWW.APIKey,
		BaseURL: cfg.DIDWW.APIBaseURL,
	})
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize DIDWW inventory: %w", err)
	}

	queries := sqlc.New(postgresClient.Pool())
	identityModule := identity.New(
		queries,
		cfg.IsDevelopment(),
		cfg.Domain,
	)
	tenancyModule := tenancy.New(queries)
	stripeConfig := stripe.DefaultConfig(cfg.Stripe.SecretKey, cfg.Stripe.WebhookSecret)
	stripeConfig.BaseURL = cfg.Stripe.APIBaseURL
	stripeClient, err := stripe.New(stripeConfig)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize Stripe payment provider: %w", err)
	}

	paystackConfig := paystack.DefaultConfig(cfg.Paystack.SecretKey)
	paystackConfig.BaseURL = cfg.Paystack.APIBaseURL
	paystackClient, err := paystack.New(paystackConfig)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize Paystack payment provider: %w", err)
	}

	commercialModule := commercial.New(postgresClient.Pool(), queries, stripeClient, paystackClient)
	platformModule := platform.New(postgresClient.Pool(), queries)
	telecomModule, err := telecom.New(telecom.Dependencies{
		DB:                   postgresClient.Pool(),
		Queries:              queries,
		CallsController:      calling.NewController(freeSwitch),
		CallsChannelStore:    calling.NewChannelStore(redisClient),
		CallsAdmission:       calling.NewAdmissionLimiter(redisClient),
		ConferenceController: conferences.NewFreeSWITCHController(freeSwitch),
		CredentialCipher:     credentialCipher,
		DIDWWInventory:       didwwInventory,
		RealtimeService:      turnService,
		RecordingStorage:     recordingStorage,
	})
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize telecom: %w", err)
	}

	resolver := authn.NewResolver(identityModule.Session.Service, tenancyModule.Credentials.Service)
	authMiddleware := middleware.NewAuthnMiddleware(resolver)
	organizationMiddleware := middleware.NewOrganizationMiddleware(queries)

	rateLimitStore, err := redisClient.NewRateLimitStore()
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize rate limit store: %w", err)
	}
	rateLimitMiddleware, err := middleware.NewRateLimitMiddleware(rateLimitStore)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize rate limit middleware: %w", err)
	}

	return &modules{
		postgres:             postgresClient,
		redis:                redisClient,
		freeSwitch:           freeSwitch,
		commercial:           commercialModule,
		identity:             identityModule,
		tenancy:              tenancyModule,
		platform:             platformModule,
		telecom:              telecomModule,
		authn:                authMiddleware,
		organizationsContext: organizationMiddleware,
		rateLimit:            rateLimitMiddleware,
		metrics:              metrics.New(redisClient),
	}, nil
}

func (m *modules) close(logger *logging.Logger) {
	if m == nil {
		return
	}
	if m.freeSwitch != nil {
		if err := m.freeSwitch.Close(); err != nil {
			logger.Warn(context.Background(), "close FreeSWITCH", "error", err)
		}
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
