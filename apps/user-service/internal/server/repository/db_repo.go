package repository

import (
	"context"
	"errors"
	"fmt"
	"global_models/global_db"
	"strings"
	"user-service/internal/domain"

	"github.com/jackc/pgconn"
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
	// проверка пользователя на nil
	if user == nil {
		return domain.ErrInvalidInput
	}

	// нормализация email перед записью в БД
	email := strings.TrimSpace(user.Email)

	// Защита от мусорных данных (хотя бизнес-валидация должна быть в сервисе)
	if email == "" {
		return domain.ErrInvalidEmail
	}

	// приведение к нижнему регистру
	email = strings.ToLower(email)

	//переопределяем у пользователя поле email
	user.Email = email

	// пытаемся вставить нового пользователя, при конфликте  - ничего не делаем
	query := `
						INSERT INTO users (
    						id, email, full_name, role, status, 
    						organization_id, password_hash, 
    						created_at, updated_at, last_login_at,
    						telegram_id, telegram_username
								)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
						ON CONFLICT (email) DO NOTHING
				`
	// делаем запрос через pgx.pool
	result, err := db.Pool.Exec(ctx, query,
		user.ID, user.Email, user.FullName,
		user.Role, user.Status, user.OrganizationID,
		user.PasswordHash, user.CreatedAt, user.UpdatedAt,
		user.LastLoginAt, user.TelegramID, user.TelegramUsername,
	)

	// обрабатываем ошибку, если она есть (проверяем error код Postgres)
	if err != nil {
		// Приводим ошибку к типу *pgconn.PgError
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503": // foreign_key_violation
				return domain.ErrOrganizationNotFound
			}
		}
		// Неизвестная ошибка БД
		return fmt.Errorf("failed to create user: %w", err)
	}

	// result это уже реализация poolAdapter из pkg. (tag.RowsAffected())
	if result == 0 {
		return domain.ErrUserAlreadyExists
	}
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

// метод для подсчёта количества пользователей
func (db *UserServiceDBRepository) Count(ctx context.Context) (int, error) {

	// -------------------------- в разработке --------------------------

	return 0, nil
}
