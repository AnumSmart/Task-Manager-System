package eventbus

import (
	"context"
	"pkg/events"
	"sync"
)

// HandlerFunc - функциональный адаптер
type HandlerFunc func(ctx context.Context, event events.Event) error

func (f HandlerFunc) Handle(ctx context.Context, event events.Event) error {
	return f(ctx, event)
}

// HandlerRegistry - реестр обработчиков событий
type HandlerRegistry struct {
	handlers map[string]EventHandler
	mu       sync.RWMutex
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]EventHandler),
	}
}

func (r *HandlerRegistry) Register(eventType string, handler EventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[eventType] = handler
}

func (r *HandlerRegistry) RegisterFunc(eventType string, fn HandlerFunc) {
	r.Register(eventType, fn)
}

func (r *HandlerRegistry) Get(eventType string) (EventHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[eventType]
	return h, ok
}
