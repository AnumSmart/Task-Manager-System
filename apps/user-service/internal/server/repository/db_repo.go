package repository

import (
	"context"
	"fmt"
	"global_models/global_db"
	"user-service/internal/domain"
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

// метод для создания юзера в базе данных
func (db *UserServiceDBRepository) Create(ctx context.Context, user *domain.User) error {

	// -------------------------- в разработке --------------------------

	return nil
}

// метод для получения пользователя по ID
func (db *UserServiceDBRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {

	// -------------------------- в разработке --------------------------

	return nil, nil
}

// метод для обновления пользователя по ID
func (db *UserServiceDBRepository) Update(ctx context.Context, user *domain.User) error {

	// -------------------------- в разработке --------------------------

	return nil
}

// метод для удаления пользователя по ID
func (db *UserServiceDBRepository) Delete(ctx context.Context, id string) error {

	// -------------------------- в разработке --------------------------

	return nil
}

// метод для получения пользователя по Email
func (db *UserServiceDBRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {

	// -------------------------- в разработке --------------------------

	return nil, nil
}

// метод для получения списка пользователей (задаётся оффсет и лимит)
func (db *UserServiceDBRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {

	// -------------------------- в разработке --------------------------

	return nil, nil
}
