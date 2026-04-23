package interceptors

import (
	"context"
	"errors"
	"log"
	"pkg/auth"
	"pkg/auth/jwt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// contextKey - тип для ключей контекста
type contextKey string

const (
	metadataKeyAuth = "authorization"
	bearerPrefix    = "Bearer "

	// ContextKeyUserID - ключ для user_id
	ContextKeyUserID contextKey = "user_id"

	// ContextKeyRole - ключ для роли
	ContextKeyRole contextKey = "role"

	// ContextKeyOrganizationID - ключ для organization_id
	ContextKeyOrganizationID contextKey = "organization_id"

	// ContextKeyEmail - ключ для email
	ContextKeyEmail contextKey = "email"

	// ContextKeyClaims - ключ для всех claims
	ContextKeyClaims contextKey = "claims"
)

// UnaryJWTInterceptor создает интерсептор для JWT авторизации
func UnaryJWTInterceptor(authService auth.AuthInterface) grpc.UnaryServerInterceptor {
	// Создаем классификатор один раз при инициализации
	classifier := NewMethodClassifier()

	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		methodName := info.FullMethod

		// 1. Публичные методы - пропускаем
		if classifier.IsPublic(methodName) {
			return handler(ctx, req)
		}

		// 2. API Key методы - пропускаем (не наша ответственность)
		if classifier.IsAPIKey(methodName) {
			return handler(ctx, req)
		}

		// 3. JWT методы - проверяем токен
		if classifier.IsJWT(methodName) {
			// извлечение JWT
			token, err := extractJWTFromMetadata(ctx)
			if err != nil {
				return nil, err // err уже содержит gRPC статус
			}

			// валидация JWT
			claims, err := authService.ValidateToken(ctx, token)
			if err != nil {
				return nil, mapJWTErrorToGRPCStatus(err)
			}

			//добавляем данные пользователя в контекст
			ctx = addClaimsToContext(ctx, claims)

			// Передаем управление дальше (в handler или следующий интерсептор)
			return handler(ctx, req)
		}

		// 4. Неизвестный метод - ошибка
		return nil, status.Errorf(codes.Unimplemented, "unknown method: %s", methodName)
	}
}

// extractJWTFromMetadata извлекает JWT токен из gRPC metadata
// Возвращает токен и ошибку (уже с gRPC статусом)
func extractJWTFromMetadata(ctx context.Context) (string, error) {
	// Шаг 1: Получить metadata из контекста
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Шаг 2: Найти ключ authorization
	authValues := md[metadataKeyAuth]
	if len(authValues) == 0 {
		// Проверить альтернативные ключи (опционально)
		authValues = md["Authorization"] // тоже будет работать, т.к. приводится к нижнему
		if len(authValues) == 0 {
			return "", status.Error(codes.Unauthenticated, "authorization header is missing")
		}
	}

	// Шаг 3: Взять первое значение
	authHeader := authValues[0]

	// Шаг 4: Проверить формат Bearer
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	// Шаг 5: Извлечь токен
	token := strings.TrimPrefix(authHeader, bearerPrefix)
	if token == "" {
		return "", status.Error(codes.Unauthenticated, "token is empty")
	}

	return token, nil
}

// вспомогательная функция маппинга ошибок (ошибки jwt --- > ошибки с GRPC кодом)
func mapJWTErrorToGRPCStatus(err error) error {
	switch {
	case errors.Is(err, jwt.ErrEmptyToken):
		return status.Error(codes.Unauthenticated, "token is empty")
	case errors.Is(err, jwt.ErrMissingToken):
		return status.Error(codes.Unauthenticated, "token is missing")
	case errors.Is(err, jwt.ErrMalformedToken):
		return status.Error(codes.Unauthenticated, "token is malformed")
	case errors.Is(err, jwt.ErrInvalidSignature):
		return status.Error(codes.Unauthenticated, "invalid token signature")
	case errors.Is(err, jwt.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, "invalid token")
	case errors.Is(err, jwt.ErrParseToken):
		return status.Error(codes.Unauthenticated, "failed to parse token")
	case errors.Is(err, jwt.ErrUnexpectedSigningMethod):
		return status.Error(codes.Unauthenticated, "unexpected signing method")
	case errors.Is(err, jwt.ErrExpiredToken):
		return status.Error(codes.Unauthenticated, "token expired")
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return status.Error(codes.Unauthenticated, "token not valid yet")
	case errors.Is(err, jwt.ErrMissingUserID):
		return status.Error(codes.Unauthenticated, "invalid token: missing user_id")
	case errors.Is(err, jwt.ErrMissingRole):
		return status.Error(codes.Unauthenticated, "invalid token: missing role")
	case errors.Is(err, jwt.ErrMissingOrganizationID):
		return status.Error(codes.Unauthenticated, "invalid token: missing organization_id")
	case errors.Is(err, jwt.ErrInvalidAuthHeader):
		return status.Error(codes.Unauthenticated, "invalid authorization header format")
	case errors.Is(err, jwt.ErrNotRefreshToken):
		return status.Error(codes.Unauthenticated, "invalid refresh token")
	case errors.Is(err, jwt.ErrInvalidConfig),
		errors.Is(err, jwt.ErrEmptySecretKey),
		errors.Is(err, jwt.ErrWeakSecretKey),
		errors.Is(err, jwt.ErrInvalidAccessTTL),
		errors.Is(err, jwt.ErrInvalidRefreshTTL):
		log.Printf("JWT configuration error: %v", err) // не забудьте импортировать log
		return status.Error(codes.Internal, "internal authentication configuration error")
	default:
		return status.Error(codes.Unauthenticated, "authentication failed")
	}
}

// addClaimsToContext добавляет данные из claims в контекст
func addClaimsToContext(ctx context.Context, claims *jwt.CustomClaims) context.Context {
	ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
	ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
	ctx = context.WithValue(ctx, ContextKeyOrganizationID, claims.OrganizationID)
	ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
	ctx = context.WithValue(ctx, ContextKeyClaims, claims)
	return ctx
}
