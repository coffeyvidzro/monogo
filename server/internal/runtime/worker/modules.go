package worker

import (
	"context"
	"fmt"
	"time"

	commercialwallets "github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/integrations/minio"
	natsintegration "github.com/coffeyvidzro/monogo/internal/integrations/nats"
	"github.com/coffeyvidzro/monogo/internal/integrations/postgres"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/platform/config"
	"github.com/coffeyvidzro/monogo/internal/platform/idempotency"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
	"github.com/coffeyvidzro/monogo/internal/telecom/numbers"
	"github.com/coffeyvidzro/monogo/internal/telecom/recordings"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/coffeyvidzro/monogo/internal/telecom/trunks"
)

type modules struct {
	postgres                *postgres.Client
	redis                   *redisintegration.Client
	nats                    *natsintegration.Client
	freeSwitch              *freeswitch.Client
	callsService            *calls.Service
	callConsumer            *calls.Consumer
	callReconciliation      *calls.ReconciliationJob
	outbox                  *outbox.PublisherJob
	webhookConsumer         *webhooks.Consumer
	webhookDelivery         *webhooks.DeliveryJob
	recordingConsumer       *recordings.Consumer
	recordingReconciliation *recordings.ReconciliationJob
	recordingIngestion      *recordings.IngestionJob
	idempotencyCleanup      *idempotency.CleanupJob
	trunkHealth             *trunks.HealthCheckJob
	numberReconciliation    *numbers.ReconciliationJob
	reservationExpiration   *commercialwallets.ExpirationJob
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

	natsClient, err := natsintegration.New(ctx, natsintegration.DefaultConfig(cfg.NATSURL))
	if err != nil {
		_ = redisClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("initialize NATS: %w", err)
	}
	if err := natsClient.Provision(ctx, natsintegration.DefaultStreamLimits()); err != nil {
		_ = natsClient.Close()
		_ = redisClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("provision NATS streams: %w", err)
	}

	freeSwitch, err := freeswitch.New(
		freeswitch.DefaultConfig(cfg.FreeSWITCHESLAddress, cfg.FreeSWITCHESLPassword),
	)
	if err != nil {
		_ = natsClient.Close()
		_ = redisClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("initialize FreeSWITCH: %w", err)
	}
	if err := freeSwitch.Connect(ctx); err != nil {
		_ = freeSwitch.Close()
		_ = natsClient.Close()
		_ = redisClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("connect FreeSWITCH: %w", err)
	}

	closeDependencies := func() {
		_ = freeSwitch.Close()
		_ = natsClient.Close()
		_ = redisClient.Close()
		postgresClient.Close()
	}

	queries := sqlc.New(postgresClient.Pool())
	walletRepository := commercialwallets.NewRepository(postgresClient.Pool())
	reservationExpiration, err := commercialwallets.NewExpirationJob(
		walletRepository,
		commercialwallets.NewService(postgresClient.Pool()),
		100,
		30*time.Second,
	)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize wallet reservation expiration: %w", err)
	}

	routingRepository := routing.NewRepository(queries)
	routingService := routing.NewService(routingRepository, nil)
	callsRepository := calls.NewRepository(queries, postgresClient.Pool())
	callController := calling.NewController(freeSwitch)
	admissionLimiter := calling.NewAdmissionLimiter(redisClient)
	callsService := calls.NewService(
		callsRepository,
		routingService,
		callController,
		calling.NewChannelStore(redisClient),
		admissionLimiter,
	)
	callConsumer := calls.NewConsumer(callsService)
	callReconciliation, err := calls.NewReconciliationJob(
		callsRepository,
		callsService,
		calls.DefaultReconciliationJobConfig(),
	)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize call reconciliation: %w", err)
	}

	recordingsRepository := recordings.NewRepository(postgresClient.Pool())
	objectClient, err := minio.New(ctx, minio.DefaultConfig(
		cfg.Domain, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey,
	))
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize recording object storage: %w", err)
	}
	recordingStorage := recordings.NewObjectStorage(objectClient)
	recordingsService := recordings.NewService(recordingsRepository, recordingStorage)
	recordingIngestion, err := recordings.NewIngestionJob(
		recordingsRepository,
		recordingStorage,
		recordings.DefaultIngestionConfig(recordings.DefaultStagingPath),
	)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize recording ingestion: %w", err)
	}
	recordingReconciliation, err := recordings.NewReconciliationJob(
		recordingsRepository,
		recordings.DefaultReconciliationJobConfig(),
	)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize recording reconciliation: %w", err)
	}

	idempotencyCleanup, err := idempotency.NewCleanupJob(
		idempotency.NewRepository(queries),
		time.Hour,
	)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize idempotency cleanup: %w", err)
	}

	trunkHealth, err := trunks.NewHealthCheckJob(
		trunks.NewRepository(queries),
		trunks.SIPOptionsProber{},
		trunks.DefaultHealthCheckConfig(),
	)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize trunk health checks: %w", err)
	}

	outboxJob, err := outbox.NewPublisherJob(
		outbox.NewRepository(queries),
		outbox.NewPublisher(natsClient),
		outbox.DefaultPublisherJobConfig("worker-outbox"),
	)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize outbox publisher: %w", err)
	}

	webhookRepository := webhooks.NewRepository(queries)
	webhookService := webhooks.NewService(webhookRepository, postgresClient.Pool())
	webhookDeliveryJob, err := webhooks.NewDeliveryJob(
		webhookRepository,
		webhooks.NewHTTPSender(),
		webhooks.DefaultDeliveryJobConfig("worker-webhooks"),
	)
	if err != nil {
		closeDependencies()
		return nil, fmt.Errorf("initialize webhook delivery worker: %w", err)
	}

	var numberReconciliation *numbers.ReconciliationJob
	if cfg.DIDWW.APIKey != "" {
		provider, providerErr := didww.New(didww.Config{
			APIKey:  cfg.DIDWW.APIKey,
			BaseURL: cfg.DIDWW.APIBaseURL,
		})
		if providerErr != nil {
			closeDependencies()
			return nil, fmt.Errorf("initialize DIDWW managed numbers: %w", providerErr)
		}
		numberService := numbers.NewService(numbers.NewRepository(queries), provider)
		numberService.ConfigureManaged(postgresClient.Pool())
		numberReconciliation, err = numbers.NewReconciliationJob(numberService, 50)
		if err != nil {
			closeDependencies()
			return nil, fmt.Errorf("initialize managed number reconciliation: %w", err)
		}
	}

	return &modules{
		postgres:                postgresClient,
		redis:                   redisClient,
		nats:                    natsClient,
		freeSwitch:              freeSwitch,
		callsService:            callsService,
		callConsumer:            callConsumer,
		callReconciliation:      callReconciliation,
		outbox:                  outboxJob,
		webhookConsumer:         webhooks.NewConsumer(natsClient, webhookService),
		webhookDelivery:         webhookDeliveryJob,
		recordingConsumer:       recordings.NewConsumer(recordingsService),
		recordingReconciliation: recordingReconciliation,
		recordingIngestion:      recordingIngestion,
		idempotencyCleanup:      idempotencyCleanup,
		trunkHealth:             trunkHealth,
		numberReconciliation:    numberReconciliation,
		reservationExpiration:   reservationExpiration,
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
	if m.nats != nil {
		if err := m.nats.Close(); err != nil {
			logger.Warn(context.Background(), "close NATS", "error", err)
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
