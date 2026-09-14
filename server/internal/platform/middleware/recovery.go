package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

// Recovery recovers panics from downstream handlers and returns a safe
// internal server error response.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicErr := fmt.Errorf("panic: %v", recovered)

				_ = debug.Stack()

				httputil.Error(
					w,
					apperror.NewInternal(
						"An unexpected error occurred",
						panicErr,
					),
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
