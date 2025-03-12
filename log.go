package brevx

import (
	"context"
	"log/slog"
)

type noopLoggerHandler struct {
}

func (n noopLoggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return false
}

func (n noopLoggerHandler) Handle(ctx context.Context, record slog.Record) error {
	return nil
}

func (n noopLoggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return n
}

func (n noopLoggerHandler) WithGroup(name string) slog.Handler {
	return n
}

func noopLogger() *slog.Logger {
	return slog.New(noopLoggerHandler{})
}
