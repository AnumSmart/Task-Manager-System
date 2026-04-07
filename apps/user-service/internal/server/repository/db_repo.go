package repository

import (
	"fmt"
	"global_models/global_db"
)

// создаём репозиторий базы данных для сервиса авторизации на базе адаптера к pgxpool

// Реализуем ТОЛЬКО то, что нужно auth_service
type UserServiceDBRepository struct {
	Pool global_db.Pool // строится на базе глобального интерфейса
}

// создаём конструктор для слоя базы данных
func NewUserServiceDBRepository(pool global_db.Pool) (*UserServiceDBRepository, error) {
	// Проверяем обязательные зависимости
	if pool == nil {
		return nil, fmt.Errorf("pool cannot be nil")
	}
	return &UserServiceDBRepository{Pool: pool}, nil
}
