package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	natsintegration "github.com/coffeyvidzro/monogo/internal/integrations/nats"
	"github.com/coffeyvidzro/monogo/internal/integrations/postgres"
	"github.com/coffeyvidzro/monogo/internal/platform/config"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
)

func main() {
	if err := run(); err != nil {
		logging.New().Error(context.Background(), "worker failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := logging.New().With("process", "worker")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	postgresClient, err := postgres.New(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		return fmt.Errorf("initialize PostgreSQL: %w", err)
	}
	defer postgresClient.Close()

	natsClient, err := natsintegration.New(ctx, natsintegration.DefaultConfig(cfg.NATSURL))
	if err != nil {
		return fmt.Errorf("initialize NATS: %w", err)
	}
	defer func() {
		if err := natsClient.Close(); err != nil {
			logger.Warn(context.Background(), "close NATS", "error", err)
		}
	}()

	limits := natsintegration.DefaultStreamLimits()
	limits.Replicas = cfg.NATSStreamReplicas
	if err := natsClient.Provision(ctx, limits); err != nil {
		return fmt.Errorf("provision NATS streams: %w", err)
	}

	queries := sqlc.New(postgresClient.Pool())
	workerID := cfg.DeploymentID
	if workerID == "" {
		workerID = "worker"
	}

	outboxRepository := outbox.NewRepository(queries)
	outboxPublisher := outbox.NewPublisher(natsClient)
	outboxJob, err := outbox.NewPublisherJob(
		outboxRepository,
		outboxPublisher,
		outbox.DefaultPublisherJobConfig(workerID+"-outbox"),
	)
	if err != nil {
		return fmt.Errorf("initialize outbox publisher: %w", err)
	}

	webhookRepository := webhooks.NewRepository(queries)
	webhookService := webhooks.NewService(webhookRepository, postgresClient.Pool())
	webhookConsumer := webhooks.NewConsumer(natsClient, webhookService)
	webhookDeliveryJob, err := webhooks.NewDeliveryJob(
		webhookRepository,
		webhooks.NewHTTPSender(),
		webhooks.DefaultDeliveryJobConfig(workerID+"-webhooks"),
	)
	if err != nil {
		return fmt.Errorf("initialize webhook delivery worker: %w", err)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		logger.Info(groupCtx, "outbox publisher started")
		return outboxJob.Run(groupCtx)
	})
	group.Go(func() error {
		logger.Info(groupCtx, "webhook consumer started")
		return webhookConsumer.Run(groupCtx)
	})
	group.Go(func() error {
		logger.Info(groupCtx, "webhook delivery worker started")
		return webhookDeliveryJob.Run(groupCtx)
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("worker runtime: %w", err)
	}
	logger.Info(context.Background(), "worker stopped")
	return nil
}
