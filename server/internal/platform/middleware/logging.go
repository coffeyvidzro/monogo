package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/httplog/v3"

	"github.com/leamout/leamout/internal/platform/logging"
)

// Logging records structured HTTP request logs using Leamout's platform logger.
// HTTP-specific instrumentation stays at the runtime boundary while the
// platform owns logger configuration.
func Logging(logger *logging.Logger) func(http.Handler) http.Handler {
	return httplog.RequestLogger(logger.Slog(), &httplog.Options{
		Level:         slog.LevelInfo,
		Schema:        httplog.SchemaOTEL,
		RecoverPanics: true,
		LogRequestHeaders: []string{
			"Content-Type",
			"Origin",
		},
		LogResponseHeaders: []string{},
	})
}
