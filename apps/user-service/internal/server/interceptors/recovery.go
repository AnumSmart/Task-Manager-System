// RecoveryInterceptor - ловит паники и преобразует их в gRPC ошибки
package interceptors

import (
	"context"
	"log"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RecoveryInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

	// Ловим панику в горутине обработчика
	defer func() {
		if r := recover(); r != nil {
			// Логируем стек вызовов для отладки
			log.Printf("PANIC recovered in method %s: %v\nStack trace:\n%s",
				info.FullMethod, r, debug.Stack())

			// Возвращаем клиенту стандартную ошибку (не раскрываем детали паники)
			err = status.Error(codes.Internal, "internal server error")
			resp = nil
		}
	}()

	// Вызываем следующий обработчик
	return handler(ctx, req)
}
