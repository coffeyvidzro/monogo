package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type requestContextKey string

const requestIDKey requestContextKey = "request_id"

func Request() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.NewString()

			ctx := context.WithValue(
				r.Context(),
				requestIDKey,
				requestID,
			)

			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestID(ctx context.Context) string {
	value, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}

	return value
}
