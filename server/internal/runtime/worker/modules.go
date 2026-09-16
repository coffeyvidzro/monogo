package worker

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	natsintegration "github.com/coffeyvidzro/monogo/internal/integrations/nats"
	"github.com/coffeyvidzro/monogo/internal/integrations/postgres"
	"github.com/coffeyvidzro/monogo/internal/platform/config"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
)

type modules struct {
	postgres        *postgres.Client
	nats            *natsintegration.Client
	outbox          *outbox.PublisherJob
	webhookConsumer *webhooks.Consumer
	webhookDelivery *webhooks.DeliveryJob
}

func newModules(ctx context.Context, cfg config.Config) (*modules, error) {
	postgresClient, err := postgres.New(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		return nil, fmt.Errorf("initialize PostgreSQL: %w", err)
	}

	natsClient, err := natsintegration.New(ctx, natsintegration.DefaultConfig(cfg.NATSURL))
	if err != nil {
		postgresClient.Close()
		return nil, fmt.Errorf("initialize NATS: %w", err)
	}

	if err := natsClient.Provision(ctx, natsintegration.DefaultStreamLimits()); err != nil {
		_ = natsClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("provision NATS streams: %w", err)
	}

	queries := sqlc.New(postgresClient.Pool())
	workerID := cfg.DeploymentID
	if workerID == "" {
		workerID = "worker"
	}

	outboxJob, err := outbox.NewPublisherJob(
		outbox.NewRepository(queries),
		outbox.NewPublisher(natsClient),
		outbox.DefaultPublisherJobConfig(workerID+"-outbox"),
	)
	if err != nil {
		_ = natsClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("initialize outbox publisher: %w", err)
	}

	webhookRepository := webhooks.NewRepository(queries)
	webhookService := webhooks.NewService(webhookRepository, postgresClient.Pool())
	webhookDeliveryJob, err := webhooks.NewDeliveryJob(
		webhookRepository,
		webhooks.NewHTTPSender(),
		webhooks.DefaultDeliveryJobConfig(workerID+"-webhooks"),
	)
	if err != nil {
		_ = natsClient.Close()
		postgresClient.Close()
		return nil, fmt.Errorf("initialize webhook delivery worker: %w", err)
	}

	return &modules{
		postgres:        postgresClient,
		nats:            natsClient,
		outbox:          outboxJob,
		webhookConsumer: webhooks.NewConsumer(natsClient, webhookService),
		webhookDelivery: webhookDeliveryJob,
	}, nil
}

func (m *modules) close(logger *logging.Logger) {
	if m == nil {
		return
	}
	if m.nats != nil {
		if err := m.nats.Close(); err != nil {
			logger.Warn(context.Background(), "close NATS", "error", err)
		}
	}
	if m.postgres != nil {
		m.postgres.Close()
	}
}
