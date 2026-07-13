package elk

import (
	"context"
	"log/slog"
)

// HandlerWrapper - обертка над slog-хендлером с дополнительными возможностями
type HandlerWrapper struct {
	handler slog.Handler
	config  *Config
}

// NewHandlerWrapper - создает обертку для хендлера
func NewHandlerWrapper(handler slog.Handler, cfg *Config) *HandlerWrapper {
	return &HandlerWrapper{
		handler: handler,
		config:  cfg,
	}
}

// Enabled - реализация slog.Handler
func (h *HandlerWrapper) Enabled(ctx context.Context, level slog.Level) bool {
	if h.handler == nil {
		return false
	}
	return h.handler.Enabled(ctx, level)
}

// Handle - реализация slog.Handler
func (h *HandlerWrapper) Handle(ctx context.Context, r slog.Record) error {
	if h.handler == nil {
		return nil
	}
	return h.handler.Handle(ctx, r)
}

// WithAttrs - реализация slog.Handler
func (h *HandlerWrapper) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.handler == nil {
		return h
	}
	return &HandlerWrapper{
		handler: h.handler.WithAttrs(attrs),
		config:  h.config,
	}
}

// WithGroup - реализация slog.Handler
func (h *HandlerWrapper) WithGroup(name string) slog.Handler {
	if h.handler == nil {
		return h
	}
	return &HandlerWrapper{
		handler: h.handler.WithGroup(name),
		config:  h.config,
	}
}
