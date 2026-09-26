// Command media runs the low-latency media-plane process.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	runtimemedia "github.com/coffeyvidzro/monogo/internal/runtime/media"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runtimemedia.Run(ctx); err != nil {
		logging.New().Error(context.Background(), "media worker failed", "error", err)
		os.Exit(1)
	}
}
