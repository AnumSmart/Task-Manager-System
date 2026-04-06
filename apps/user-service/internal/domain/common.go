package domain

import (
	"time"

	"github.com/google/uuid"
)

// GenerateID генерирует новый UUID для идентификаторов сущностей
func GenerateID() string {
	return uuid.New().String()
}

// ValidateID проверяет корректность UUID
func ValidateID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// MustParseID парсит ID или паникует (для использования в конструкторах)
func MustParseID(id string) string {
	parsed, err := uuid.Parse(id)
	if err != nil {
		panic("invalid UUID format: " + id)
	}
	return parsed.String()
}

// Now возвращает текущее время (удобно для тестов)
var Now = func() time.Time {
	return time.Now()
}
