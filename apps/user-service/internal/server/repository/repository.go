package repository

import "fmt"

// описание структуры слоя репозитория
type UserServiceRepository struct {
	DBRepo    *UserServiceDBRepository
	CacheRepo *UserServiceCacheRepository
}

// конструктор для слоя репозиторий
func NewUserServiceRepository(dbRepo *UserServiceDBRepository, cacheRepo *UserServiceCacheRepository) (*UserServiceRepository, error) {
	// Проверяем обязательные зависимости
	if dbRepo == nil {
		return nil, fmt.Errorf("dbRepo is required")
	}
	if cacheRepo == nil {
		return nil, fmt.Errorf("blackListRepo is required")
	}
	return &UserServiceRepository{
		DBRepo:    dbRepo,
		CacheRepo: cacheRepo,
	}, nil
}

// метод для теста
func (r *UserServiceRepository) Echo() string {
	return fmt.Sprintln("Hello from repo layer!")
}
