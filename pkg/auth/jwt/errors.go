package jwt

import "errors"

// JWT-specific errors
var (
	// ErrInvalidToken - токен недействителен
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken - токен истек
	ErrExpiredToken = errors.New("token has expired")

	// ErrMissingToken - токен отсутствует
	ErrMissingToken = errors.New("missing token")

	// ErrEmptyToken - пустой токен
	ErrEmptyToken = errors.New("empty token")

	// ErrInvalidAuthHeader - неверный формат заголовка авторизации
	ErrInvalidAuthHeader = errors.New("invalid authorization header format, expected 'Bearer <token>'")

	// ErrMissingUserID - отсутствует user_id в claims
	ErrMissingUserID = errors.New("user_id is missing in token claims")

	// ErrMissingRole - отсутствует role в claims
	ErrMissingRole = errors.New("role is missing in token claims")

	// ErrMissingOrganizationID - отсутствует organization_id в claims
	ErrMissingOrganizationID = errors.New("organization_id is missing in token claims")

	// ErrNotRefreshToken - токен не является refresh токеном
	ErrNotRefreshToken = errors.New("token is not a refresh token")

	// ErrInvalidConfig - неверная конфигурация
	ErrInvalidConfig = errors.New("invalid jwt config")

	// ErrEmptySecretKey - пустой секретный ключ
	ErrEmptySecretKey = errors.New("jwt secret key is empty")

	// ErrWeakSecretKey - слабый секретный ключ
	ErrWeakSecretKey = errors.New("jwt secret key must be at least 32 characters")

	// ErrInvalidAccessTTL - неверный TTL для access токена
	ErrInvalidAccessTTL = errors.New("access token TTL must be positive")

	// ErrInvalidRefreshTTL - неверный TTL для refresh токена
	ErrInvalidRefreshTTL = errors.New("refresh token TTL must be positive")

	// ErrMalformedToken - токен имеет неправильный формат (не 3 части, невалидный base64 и т.д.)
	ErrMalformedToken = errors.New("malformed token")

	// ErrInvalidSignature - подпись токена недействительна
	ErrInvalidSignature = errors.New("invalid token signature")

	// ErrTokenNotValidYet - токен еще не активен (проверка nbf claim)
	ErrTokenNotValidYet = errors.New("token is not valid yet")

	// ErrUnexpectedSigningMethod - неожиданный метод подписи (например, ожидали HMAC, а получили RSA)
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")

	// ErrParseToken - общая ошибка парсинга токена (когда не можем классифицировать)
	ErrParseToken = errors.New("failed to parse token")
)
