package worker

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/coffeyvidzro/monogo/internal/platform/logging"
)

func runConsumers(ctx context.Context, logger *logging.Logger, modules *modules) error {
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		logger.Info(groupCtx, "outbox publisher started")
		return modules.outbox.Run(groupCtx)
	})
	group.Go(func() error {
		logger.Info(groupCtx, "webhook consumer started")
		return modules.webhookConsumer.Run(groupCtx)
	})
	group.Go(func() error {
		logger.Info(groupCtx, "webhook delivery worker started")
		return modules.webhookDelivery.Run(groupCtx)
	})

	return group.Wait()
}
