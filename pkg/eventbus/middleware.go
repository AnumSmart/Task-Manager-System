package eventbus

import (
	"context"
	"fmt"
	"log"
	"pkg/events"
	"time"
)

// LoggingMiddleware - логирование начала и конца обработки
func LoggingMiddleware(next EventHandler) EventHandler {
	return HandlerFunc(func(ctx context.Context, event events.Event) error {
		start := time.Now()
		log.Printf("[Handler] Processing %s (id=%s)", event.GetEventType(), event.GetEventID())

		err := next.Handle(ctx, event)

		duration := time.Since(start)
		if err != nil {
			log.Printf("[Handler] Failed %s: %v (duration=%v)", event.GetEventType(), err, duration)
		} else {
			log.Printf("[Handler] Completed %s (duration=%v)", event.GetEventType(), duration)
		}

		return err
	})
}

// RecoveryMiddleware - восстановление после паники
func RecoveryMiddleware(next EventHandler) EventHandler {
	return HandlerFunc(func(ctx context.Context, event events.Event) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic recovered: %v", r)
				log.Printf("[Handler] ⚠️ Panic in %s: %v", event.GetEventType(), r)
			}
		}()
		return next.Handle(ctx, event)
	})
}
