package interceptors

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	metadataKeyServiceName = "x-service-name"
	metadataKeyServiceKey  = "x-service-key"
)

// UnaryAPIKeyInterceptor создает интерсептор для API Key авторизации
func UnaryAPIKeyInterceptor(allowedServices map[string]string) grpc.UnaryServerInterceptor {
	classifier := NewMethodClassifier()
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		methodName := info.FullMethod

		// Проверяем только API Key методы
		if !classifier.IsAPIKey(methodName) {
			return handler(ctx, req)
		}

		log.Printf("🔑 API Key проверка для метода: %s", methodName)

		// Если мапа разрешённых сервисов пустая - логируем предупреждение
		if allowedServices == nil || len(allowedServices) == 0 {
			log.Printf("⚠️ API Key: no allowed services configured")
			return nil, status.Error(codes.Internal, "API Key authentication not configured")
		}

		// Извлекаем service name и key из metadata
		serviceName, serviceKey, err := extractAPIKeyFromMetadata(ctx)
		if err != nil {
			log.Printf("❌ API Key: ошибка извлечения credentials: %v", err)
			return nil, err
		}

		// Валидируем API key
		expectedKey, exists := allowedServices[serviceName]
		if !exists {
			log.Printf("❌ API Key: сервис '%s' не найден в списке разрешённых", serviceName)
			return nil, status.Error(codes.Unauthenticated, "unknown service")
		}

		if expectedKey != serviceKey {
			log.Printf("❌ API Key: неверный ключ для сервиса '%s'", serviceName)
			return nil, status.Error(codes.Unauthenticated, "invalid service key")
		}

		// Добавляем информацию о сервисе в контекст
		ctx = context.WithValue(ctx, ContextKeyServiceName, serviceName)

		log.Printf("✅ API Key: сервис '%s' авторизован", serviceName)
		return handler(ctx, req)
	}
}

// extractAPIKeyFromMetadata извлекает service name и service key из gRPC metadata
func extractAPIKeyFromMetadata(ctx context.Context) (string, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Получаем service name (пробуем разные варианты регистра)
	serviceNames := md[metadataKeyServiceName]
	if len(serviceNames) == 0 {
		serviceNames = md["X-Service-Name"] // ещё один вариант ввода ключа
		if len(serviceNames) == 0 {
			return "", "", status.Error(codes.Unauthenticated, "missing service name")
		}
	}

	// Получаем service key
	serviceKeys := md[metadataKeyServiceKey]
	if len(serviceKeys) == 0 {
		serviceKeys = md["X-Service-Key"] // ещё один вариант ввода ключа
		if len(serviceKeys) == 0 {
			return "", "", status.Error(codes.Unauthenticated, "missing service key")
		}
	}

	serviceName := serviceNames[0]
	serviceKey := serviceKeys[0]

	if serviceName == "" {
		return "", "", status.Error(codes.Unauthenticated, "empty service name")
	}
	if serviceKey == "" {
		return "", "", status.Error(codes.Unauthenticated, "empty service key")
	}

	return serviceName, serviceKey, nil
}
