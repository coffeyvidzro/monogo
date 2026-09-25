// Package media assembles and runs the realtime media plane.
package media

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/platform/logging"
)

// Run owns the process lifecycle for the media plane. Network listeners and
// provider adapters will be assembled here as they are introduced; until then,
// the process remains alive so shutdown behavior and packaging can be exercised.
func Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("media runtime context is required")
	}

	logger := logging.New().With("process", "media")
	logger.Info(ctx, "media worker started")
	<-ctx.Done()
	logger.Info(context.Background(), "media worker stopped")
	return nil
}
