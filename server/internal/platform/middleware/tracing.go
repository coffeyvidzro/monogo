package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const httpServerSpanName = "http.server"

// Tracing instruments inbound HTTP requests with OpenTelemetry server spans.
// Incoming trace context is extracted by otelhttp and made available through
// the request context to downstream application and integration instrumentation.
func Tracing() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(
			next,
			httpServerSpanName,
		)
	}
}
