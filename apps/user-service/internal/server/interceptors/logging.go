// LoggingInterceptor логирует начало и конец каждого gRPC вызова.
//
// Логирует:
//   - Имя вызываемого метода
//   - Длительность выполнения
//   - Статус ошибки (если есть)
//
// Interceptor не изменяет запрос/ответ и не блокирует выполнение.
//
// Пример лога:
//
//	[gRPC] --> /user.v1.UserService/CreateUser
//	[gRPC] <-- /user.v1.UserService/CreateUser | Duration: 45ms | Success
package interceptors

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor - интерсептор для логирования всех gRPC запросов
func LoggingInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	// Время начала запроса
	start := time.Now()

	// Логируем начало запроса
	log.Printf("[gRPC] --> %s", info.FullMethod)

	// Вызываем реальный обработчик
	resp, err := handler(ctx, req)

	// Время выполнения
	duration := time.Since(start)

	// Получаем статус код ошибки (если есть)
	st, _ := status.FromError(err)

	// Логируем завершение запроса
	if err != nil {
		log.Printf("[gRPC] <-- %s | Duration: %v | Error: %v (code: %s)",
			info.FullMethod, duration, err, st.Code().String())
	} else {
		log.Printf("[gRPC] <-- %s | Duration: %v | Success",
			info.FullMethod, duration)
	}

	return resp, err
}
