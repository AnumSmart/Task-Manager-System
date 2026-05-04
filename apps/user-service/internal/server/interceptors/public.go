package interceptors

import (
	"context"
	"log"

	"google.golang.org/grpc"
)

// UnaryPublicInterceptor логирует публичные методы (без авторизации)
func UnaryPublicInterceptor() grpc.UnaryServerInterceptor {
	classifier := NewMethodClassifier()

	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		methodName := info.FullMethod

		// Только для публичных методов
		if classifier.IsPublic(methodName) {
			log.Printf("🌐 Публичный метод вызван: %s", methodName)
		}

		return handler(ctx, req)
	}
}
