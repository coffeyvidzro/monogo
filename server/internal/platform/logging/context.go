package logging

import "context"

// contextKey prevents collisions with values owned by other packages.
type contextKey struct{}

// With stores a logger in the context.
func With(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the logger stored in ctx, or a default logger when none
// has been attached.
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(contextKey{}).(*Logger); ok && logger != nil {
		return logger
	}

	return New()
}
