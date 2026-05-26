package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"global_models/global_db"
	"strings"
	"time"
	"user-service/internal/domain"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/lib/pq"
)

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

// CreateWithTx - создание пользователя в рамках переданной транзакции
func (db *UserServiceDBRepository) CreateWithTx(ctx context.Context, tx global_db.Tx, user *domain.User) error {
	if user == nil {
		return domain.ErrInvalidInput
	}

	email := strings.TrimSpace(user.Email)
	if email == "" {
		return domain.ErrInvalidEmail
	}
	email = strings.ToLower(email)
	user.Email = email

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

	result, err := tx.Exec(ctx, query,
		user.ID, user.Email, user.FullName,
		user.Role, user.Status, user.OrganizationID,
		user.PasswordHash, user.CreatedAt, user.UpdatedAt,
		user.LastLoginAt, user.TelegramID, user.TelegramUsername,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				return domain.ErrOrganizationNotFound
			}
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

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

// GetByIDWithTx - получение пользователя в рамках переданной транзакции
func (db *UserServiceDBRepository) GetByIDWithTx(ctx context.Context, tx global_db.Tx, id string) (*domain.User, error) {
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

	// создаём переменную как структуру (не nil указатель)
	var user domain.User

	// делаем запрос в БД через транзакцию
	err := tx.QueryRow(ctx, query, id).Scan(
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

// метод получения пользователя по телеграмм ID
func (db *UserServiceDBRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	// 1. Проверка контекста
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	// 2. Валидация входных данных
	if telegramID == 0 {
		return nil, domain.ErrInvalidInput
	}

	// 3. SQL запрос (без пароля, так как он не нужен для поиска по Telegram)
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
			created_at, 
			updated_at, 
			last_login_at, 
			deleted_at
		FROM users 
		WHERE telegram_id = $1
		LIMIT 1
	`

	user := &domain.User{}

	// Для nullable полей
	var telegramIDNull sql.NullInt64
	var telegramUsernameNull sql.NullString
	var lastLoginAtNull sql.NullTime
	var deletedAtNull sql.NullTime

	// 4. Выполнение запроса
	err := db.Pool.QueryRow(ctx, query, telegramID).Scan(
		&user.ID,
		&user.Email,
		&telegramIDNull,
		&telegramUsernameNull,
		&user.Role,
		&user.Status,
		&user.FullName,
		&user.OrganizationID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAtNull,
		&deletedAtNull,
	)

	if err != nil {
		// 5. Обработка ошибок
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by telegram_id %d: %w", telegramID, err)
	}

	// 6. Заполнение nullable полей
	if telegramIDNull.Valid {
		telegramID := telegramIDNull.Int64
		user.TelegramID = &telegramID
	}

	if telegramUsernameNull.Valid {
		user.TelegramUsername = &telegramUsernameNull.String
	}

	if lastLoginAtNull.Valid {
		user.LastLoginAt = &lastLoginAtNull.Time
	}

	if deletedAtNull.Valid {
		user.DeletedAt = &deletedAtNull.Time
	}

	// 7. Убедимся, что Telegram ID совпадает (дополнительная проверка)
	if !user.IsTelegramLinked() {
		return nil, fmt.Errorf("inconsistent state: telegram_id in DB but not set in user object")
	}

	return user, nil
}

// метод получения списка пользователей по списку ID
func (db *UserServiceDBRepository) BatchGetByIDs(ctx context.Context, ids []string) ([]*domain.User, error) {
	if len(ids) == 0 {
		return []*domain.User{}, nil
	}

	// Используем параметризованный запрос с ANY
	query := `
        SELECT id, email, telegram_id, telegram_username, role, status, 
               full_name, organization_id, created_at, updated_at, 
               last_login_at, deleted_at, password_hash
        FROM users 
        WHERE id = ANY($1) AND deleted_at IS NULL
    `
	rows, err := db.Pool.Query(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to batch get users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0, len(ids))
	for rows.Next() {
		user := &domain.User{}

		var telegramID sql.NullInt64
		var telegramUsername sql.NullString
		var lastLoginAt, deletedAt sql.NullTime

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&telegramID,
			&telegramUsername,
			&user.Role,
			&user.Status,
			&user.FullName,
			&user.OrganizationID,
			&user.CreatedAt,
			&user.UpdatedAt,
			&lastLoginAt,
			&deletedAt,
			&user.PasswordHash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Обработка nullable полей
		if telegramID.Valid {
			id := telegramID.Int64
			user.TelegramID = &id
		}
		if telegramUsername.Valid {
			user.TelegramUsername = &telegramUsername.String
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}
		if deletedAt.Valid {
			user.DeletedAt = &deletedAt.Time
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
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
									telegram_username = $10,
									deleted_at = $11
						WHERE id = $12 AND deleted_at IS NULL
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
		user.DeletedAt,
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

// UpdateWithTx - обновление пользователя в рамках переданной транзакции
func (db *UserServiceDBRepository) UpdateWithTx(ctx context.Context, tx global_db.Tx, user *domain.User) error {
	if user == nil || user.ID == "" {
		return domain.ErrInvalidInput
	}

	email := strings.TrimSpace(user.Email)
	if email == "" {
		return domain.ErrInvalidEmail
	}
	email = strings.ToLower(email)
	user.Email = email
	user.UpdatedAt = time.Now()

	query := `
        UPDATE users
        SET email = $1, full_name = $2, role = $3, status = $4,
            organization_id = $5, password_hash = $6, updated_at = $7,
            last_login_at = $8, telegram_id = $9, telegram_username = $10
        WHERE id = $11 AND deleted_at IS NULL
    `

	result, err := tx.Exec(ctx, query,
		user.Email, user.FullName, user.Role, user.Status,
		user.OrganizationID, user.PasswordHash, user.UpdatedAt,
		user.LastLoginAt, user.TelegramID, user.TelegramUsername,
		user.ID,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				return domain.ErrOrganizationNotFound
			case "23505":
				return domain.ErrUserAlreadyExists
			}
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

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

// метод для HealthCheck
func (db *UserServiceDBRepository) PingDB(ctx context.Context) error {
	// Проверка PostgreSQL
	if _, err := db.Pool.Exec(ctx, "SELECT 1"); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}
	return nil
}

// ListWithFilters - получение списка пользователей с фильтрацией и поиском
func (db *UserServiceDBRepository) ListWithFilters(ctx context.Context, organizationID string, roleFilter *domain.Role, statusFilter *domain.UserStatus, searchQuery string, offset, limit int) ([]*domain.User, error) {
	query := `
		SELECT id, organization_id, email, full_name, role, status, 
		       telegram_id, telegram_username, created_at, updated_at, last_login_at
		FROM users
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	// Фильтр по организации (обязательный)
	if organizationID != "" {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argIndex))
		args = append(args, organizationID)
		argIndex++
	}

	// Фильтр по роли
	if roleFilter != nil {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, string(*roleFilter))
		argIndex++
	}

	// Фильтр по статусу
	if statusFilter != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, string(*statusFilter))
		argIndex++
	}

	// Поиск по имени или email
	if searchQuery != "" {
		conditions = append(conditions, fmt.Sprintf("(full_name ILIKE $%d OR email ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + searchQuery + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	// Собираем WHERE часть
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Добавляем ORDER BY, LIMIT, OFFSET
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user := &domain.User{}
		var telegramID sql.NullInt64
		var telegramUsername sql.NullString
		var lastLoginAt sql.NullTime

		err := rows.Scan(
			&user.ID, &user.OrganizationID, &user.Email, &user.FullName,
			&user.Role, &user.Status, &telegramID, &telegramUsername,
			&user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if telegramID.Valid {
			user.TelegramID = &telegramID.Int64
		}
		if telegramUsername.Valid {
			user.TelegramUsername = &telegramUsername.String
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// CountWithFilters - подсчёт пользователей с фильтрацией
// CountWithFilters - подсчёт пользователей с фильтрацией и поиском
func (db *UserServiceDBRepository) CountWithFilters(ctx context.Context, organizationID string, roleFilter *domain.Role, statusFilter *domain.UserStatus, searchQuery string) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	args := []interface{}{}
	argIndex := 1

	if organizationID != "" {
		query += fmt.Sprintf(" AND organization_id = $%d", argIndex)
		args = append(args, organizationID)
		argIndex++
	}

	if roleFilter != nil {
		query += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, string(*roleFilter))
		argIndex++
	}

	if statusFilter != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*statusFilter))
		argIndex++
	}

	if searchQuery != "" {
		query += fmt.Sprintf(" AND (full_name ILIKE $%d OR email ILIKE $%d)", argIndex, argIndex+1)
		searchPattern := "%" + searchQuery + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	var count int
	err := db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}
