package worker

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/platform/config"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
)

func Run(ctx context.Context) error {
	logger := logging.New().With("process", "worker")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	modules, err := newModules(ctx, cfg)
	if err != nil {
		return err
	}
	defer modules.close(logger)

	if err := modules.ready(ctx); err != nil {
		return fmt.Errorf("worker readiness: %w", err)
	}

	if err := runConsumers(ctx, logger, modules); err != nil {
		return fmt.Errorf("worker runtime: %w", err)
	}

	logger.Info(context.Background(), "worker stopped")
	return nil
}
