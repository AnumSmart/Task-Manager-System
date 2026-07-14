package logger

import (
	"context"
	"log/slog"
)

// multiHandler - комбинирует несколько хендлеров в один
// Позволяет отправлять логи в несколько мест одновременно
type multiHandler struct {
	handlers []slog.Handler
}

// newMultiHandler - создает мульти-хендлер из переданных хендлеров
// Если передан только один хендлер, возвращает его без обертки
func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	// Фильтруем nil хендлеры
	var filtered []slog.Handler
	for _, h := range handlers {
		if h != nil {
			filtered = append(filtered, h)
		}
	}

	// Если хендлеров нет - возвращаем nil
	if len(filtered) == 0 {
		return nil
	}

	// Если только один хендлер - возвращаем его без обертки
	if len(filtered) == 1 {
		return filtered[0]
	}

	// Иначе создаем мульти-хендлер
	return &multiHandler{
		handlers: filtered,
	}
}

// Enabled - проверяет, включен ли хотя бы один хендлер для данного уровня
func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle - обрабатывает запись лога всеми хендлерами
// Если один хендлер возвращает ошибку, остальные все равно продолжают работу
func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var lastErr error

	// Для каждого хендлера создаем клон записи
	for _, handler := range h.handlers {
		// Клонируем запись, чтобы избежать race condition
		clone := r.Clone()

		if err := handler.Handle(ctx, clone); err != nil {
			lastErr = err
			// Продолжаем с другими хендлерами даже при ошибке
			// Используем стандартный логгер для сообщения об ошибке
			// чтобы избежать циклической зависимости
			defaultLogger := slog.Default()
			if defaultLogger != nil {
				defaultLogger.Error("Failed to handle log record",
					"error", err,
					"handler_type", handlerName(handler),
				)
			}
		}
	}

	return lastErr
}

// WithAttrs - создает новый мульти-хендлер с добавленными атрибутами
func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}

	return &multiHandler{
		handlers: newHandlers,
	}
}

// WithGroup - создает новый мульти-хендлер с добавленной группой
func (h *multiHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}

	return &multiHandler{
		handlers: newHandlers,
	}
}

// handlerName - вспомогательная функция для определения типа хендлера
func handlerName(handler slog.Handler) string {
	switch h := handler.(type) {
	case *slog.JSONHandler:
		return "json"
	case *slog.TextHandler:
		return "text"
	case *multiHandler:
		return "multi"
	case interface{ String() string }:
		return h.String()
	}
	return "" //TODO
}
