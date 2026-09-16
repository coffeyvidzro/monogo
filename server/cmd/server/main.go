package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	runtimeserver "github.com/coffeyvidzro/monogo/internal/runtime/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runtimeserver.Run(ctx); err != nil {
		logging.New().Error(context.Background(), "server failed", "error", err)
		os.Exit(1)
	}
}
