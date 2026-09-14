package logging

import (
	"context"
	"log/slog"
	"os"
)

// Logger is Leamout's application logging facade.
//
// Keep application code dependent on this type rather than constructing
// sloggers throughout the codebase. The implementation can evolve without
// changing callers.
type Logger struct {
	logger *slog.Logger
}

// New creates a logger that writes structured JSON logs to stdout.
func New() *Logger {
	return &Logger{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})),
	}
}

// NewWithHandler creates a Logger backed by the supplied slog handler.
// It is primarily useful for tests and alternate output configuration.
func NewWithHandler(handler slog.Handler) *Logger {
	return &Logger{logger: slog.New(handler)}
}

// Slog returns the underlying slog logger for infrastructure integrations.
func (l *Logger) Slog() *slog.Logger {
	return l.logger
}

// With returns a logger with the supplied attributes attached to every entry.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{logger: l.logger.With(args...)}
}

// Debug logs a debug-level message.
func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

// Info logs an informational message.
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

// Warn logs a warning-level message.
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

// Error logs an error-level message.
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}
