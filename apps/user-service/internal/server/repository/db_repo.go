package repository

import (
	"context"
	"errors"
	"fmt"
	"global_models/global_db"
	"strings"
	"time"
	"user-service/internal/domain"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
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

// метод для получения пользователя по ID (ID должен быть в формате uuid)
func (db *UserServiceDBRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	// проверка на пустой id
	if id == "" {
		return nil, domain.ErrInvalidInput
	}

	// создаём строку запроса
	query := `
					SELECT id, email, full_name, role, status, 
    						organization_id, password_hash, 
    						created_at, updated_at, last_login_at,
    						telegram_id, telegram_username 
					FROM users 
					WHERE id = $1 AND deleted_at IS NULL
					
	`

	// создаём переменную КАК СТРУКТУРУ (не nil указатель)
	var user domain.User

	// делаем запрос в БД
	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Role,
		&user.Status,
		&user.OrganizationID,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.TelegramID,
		&user.TelegramUsername,
	)

	if err != nil {
		// сравнение ошибки на ошибку отсутствия пользователя
		if errors.Is(pgx.ErrNoRows, err) {
			return nil, domain.ErrUserNotFound
		}
		// какая-то другая ошибка
		return nil, fmt.Errorf("failed to get user by id %s: %w", id, err)
	}

	return &user, nil
}

// метод для обновления пользователя по ID
func (db *UserServiceDBRepository) Update(ctx context.Context, user *domain.User) error {
	// 1. Проверка пользователя на nil
	if user == nil {
		return domain.ErrInvalidInput
	}

	// 2. Проверка ID (обязательно, чтобы знать какого пользователя обновлять)
	if user.ID == "" {
		return domain.ErrInvalidInput
	}

	// 3. Нормализация email
	email := strings.TrimSpace(user.Email)
	if email == "" {
		return domain.ErrInvalidEmail
	}
	email = strings.ToLower(email)

	user.Email = email

	// 4. Обновляем timestamp
	user.UpdatedAt = time.Now()

	query := `
						UPDATE users
						SET
									email = $1,
									full_name = $2,
									role = $3,
									status = $4,
									organization_id = $5,
									password_hash = $6,
									updated_at = $7,
									last_login_at = $8,
									telegram_id = $9,
									telegram_username = $10
						WHERE id = $11 WHERE id = $1 AND deleted_at IS NULL
	`

	//выполняем запрос в БД
	result, err := db.Pool.Exec(ctx, query,
		user.Email,
		user.FullName,
		user.Role,
		user.Status,
		user.OrganizationID,
		user.PasswordHash,
		user.UpdatedAt,
		user.LastLoginAt,
		user.TelegramID,
		user.TelegramUsername,
		user.ID,
	)

	if err != nil {
		// создаём переменную, чтобы распознать коды ошибок базы данных
		var pgxErr *pgconn.PgError
		if errors.As(err, &pgxErr) {
			switch pgxErr.Code {
			case "23503": // foreign_key_violation
				return domain.ErrOrganizationNotFound
			case "23505": // unique_violation (если email уже существует у другого пользователя)
				return domain.ErrUserAlreadyExists
			}
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	// проверяем, что одновление действительно прошло
	// result это уже реализация poolAdapter из pkg. (tag.RowsAffected())
	if result == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// метод для удаления пользователя по ID
func (db *UserServiceDBRepository) Delete(ctx context.Context, id string) error {
	// проверка на пустой id
	if id == "" {
		return domain.ErrInvalidInput
	}

	// создаём строку звпроса
	query := `
						UPDATE users SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL
	`
	// делаем запрос в БД
	result, err := db.Pool.Exec(ctx, query, time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// проверяем, что одновление действительно прошло
	// result это уже реализация poolAdapter из pkg. (tag.RowsAffected())
	if result == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// метод для получения пользователя по Email
func (db *UserServiceDBRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// нормализация email перед записью в БД
	email = strings.TrimSpace(email)

	// Защита от мусорных данных (хотя бизнес-валидация должна быть в сервисе)
	if email == "" {
		return nil, domain.ErrInvalidEmail
	}

	// приведение к нижнему регистру
	email = strings.ToLower(email)

	// создаём переменную для результата
	var user domain.User

	// создаём строку звпроса к БД (с учётом, того что у нас организовано мягкое удаление)
	query := `
						SELECT 
								id, email, full_name, role, status, 
								organization_id, password_hash, 
								created_at, updated_at, last_login_at,
								telegram_id, telegram_username
						FROM users
						WHERE email = $1 and deleted_at IS NULL
	`
	// создаём запрос к БД
	err := db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Role,
		&user.Status,
		&user.OrganizationID,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.TelegramID,
		&user.TelegramUsername,
	)

	if err != nil {
		// проверка, что ошибка, это ошибка - отсутствия пользователя
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email %s: %w", email, err)
	}

	return &user, nil
}

// метод для получения списка пользователей (задаётся оффсет и лимит)
func (db *UserServiceDBRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {

	query := `
		SELECT 
			id, 
			email, 
			telegram_id, 
			telegram_username, 
			role, 
			status, 
			full_name, 
			organization_id, 
			password_hash, 
			created_at, 
			updated_at, 
			last_login_at,
			deleted_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	// освобождаем ресурсы
	defer rows.Close()

	// созда
	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.TelegramID,
			&user.TelegramUsername,
			&user.Role,
			&user.Status,
			&user.FullName,
			&user.OrganizationID,
			&user.PasswordHash,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastLoginAt,
			&user.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	if len(users) == 0 {
		return []*domain.User{}, nil // возвращаем пустой слайс, а не nil
	}

	return users, nil
}

// метод для подсчёта количества пользователей
func (db *UserServiceDBRepository) Count(ctx context.Context) (int, error) {
	// создаём строку запроса
	query := `
						SELECT COUNT(*) FROM users
						WHERE deleted_at IS NULL
			`

	// создаём переменную, куда будем складывать ответ
	var num int

	// создаём звпрос в БД
	err := db.Pool.QueryRow(ctx, query).Scan(&num)

	if err != nil {
		// COUNT(*) всегда возвращает строку с числом (даже 0), поэтому ErrNoRows не возникнет
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return num, nil
}
