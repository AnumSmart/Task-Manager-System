package interceptors

import "fmt"

//классификатор методов, в зависимости от прав доступа

// MethodType определяет тип авторизации для метода
type MethodType int

const (
	MethodTypePublic MethodType = iota // Публичный метод (без авторизации)
	MethodTypeJWT                      // JWT авторизация (пользователи)
	MethodTypeAPIKey                   // API Key авторизация (сервисы)
)

// MethodClassifier хранит классификацию всех gRPC методов
type MethodClassifier struct {
	methodMap map[string]MethodType // мапа в которой ключ - это название метода, а занчение - тип метода из константы
}

// NewMethodClassifier создает классификатор с предопределенными правилами
func NewMethodClassifier() *MethodClassifier {
	return &MethodClassifier{
		methodMap: buildMethodMap(),
	}
}

// buildMethodMap строит мапу методов на основе proto-спецификации
func buildMethodMap() map[string]MethodType {
	return map[string]MethodType{
		// ========== Публичные методы ==========
		"/user.v1.UserService/SetupInitialOrganization": MethodTypePublic,
		"/user.v1.UserService/LinkTelegram":             MethodTypePublic,
		"/user.v1.UserService/HealthCheck":              MethodTypePublic,

		// ========== JWT методы (пользователи) ==========
		"/user.v1.UserService/GetMyProfile":    MethodTypeJWT,
		"/user.v1.UserService/UpdateMyProfile": MethodTypeJWT,
		"/user.v1.UserService/GetOrganization": MethodTypeJWT,
		"/user.v1.UserService/CreateUser":      MethodTypeJWT,
		"/user.v1.UserService/GetUser":         MethodTypeJWT,
		"/user.v1.UserService/UpdateUser":      MethodTypeJWT,
		"/user.v1.UserService/DeleteUser":      MethodTypeJWT,
		"/user.v1.UserService/ListUsers":       MethodTypeJWT,
		"/user.v1.UserService/Logout":          MethodTypeJWT,

		// ========== API Key методы (межсервисные) ==========
		"/user.v1.UserService/ValidateToken":     MethodTypeAPIKey,
		"/user.v1.UserService/GetUserByID":       MethodTypeAPIKey,
		"/user.v1.UserService/GetUserByTelegram": MethodTypeAPIKey,
		"/user.v1.UserService/BatchGetUsers":     MethodTypeAPIKey,
		"/user.v1.UserService/ValidateUser":      MethodTypeAPIKey,
		"/user.v1.UserService/CheckUserExists":   MethodTypeAPIKey,
		"/user.v1.UserService/GetUsersByRole":    MethodTypeAPIKey,
		"/user.v1.UserService/GetUserRole":       MethodTypeAPIKey,
		"/user.v1.UserService/GetAllUsers":       MethodTypeAPIKey,
	}
}

// Classify определяет тип авторизации для метода
func (c *MethodClassifier) Classify(fullMethod string) MethodType {
	if methodType, exists := c.methodMap[fullMethod]; exists {
		return methodType
	}
	// По умолчанию - требуем JWT (безопасное поведение)
	return MethodTypeJWT
}

// IsPublic проверяет, является ли метод публичным
func (c *MethodClassifier) IsPublic(fullMethod string) bool {
	return c.Classify(fullMethod) == MethodTypePublic
}

// IsJWT проверяет, требует ли метод JWT авторизации
func (c *MethodClassifier) IsJWT(fullMethod string) bool {
	return c.Classify(fullMethod) == MethodTypeJWT
}

// IsAPIKey проверяет, требует ли метод API Key авторизации
func (c *MethodClassifier) IsAPIKey(fullMethod string) bool {
	return c.Classify(fullMethod) == MethodTypeAPIKey
}

// Validate проверяет, что все методы из proto покрыты классификацией
func (c *MethodClassifier) Validate(allMethodsFromProto []string) error {
	for _, method := range allMethodsFromProto {
		if _, exists := c.methodMap[method]; !exists {
			return fmt.Errorf("method %s not classified", method)
		}
	}
	return nil
}
