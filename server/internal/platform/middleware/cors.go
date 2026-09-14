package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

func CORS(allowedOrigins []string, development bool) func(http.Handler) http.Handler {
	if development && len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	return cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,

		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},

		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"Origin",
			"X-Request-ID",
			"X-Organization-ID",
			"X-CSRF-Token",
		},

		ExposedHeaders: []string{
			"X-Request-ID",
		},

		AllowCredentials: true,
		MaxAge:           300,
	})
}
