package middleware

import (
	"net/http"

	"github.com/leamout/leamout/internal/platform/metrics"
)

// Metrics records process-level HTTP request and error counters.
// Keep this middleware deliberately small; the platform registry owns metric
// state and exposition, while runtime owns HTTP instrumentation.
func Metrics(registry *metrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			registry.IncRequests()

			writer := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(writer, r)

			if writer.status >= http.StatusInternalServerError {
				registry.IncErrors()
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}

	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(body)
}

// Unwrap lets http.ResponseController reach optional capabilities of the
// underlying ResponseWriter (for example Flush or Hijack).
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
