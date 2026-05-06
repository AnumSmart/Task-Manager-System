package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"user-service/internal/domain"
)

// метод получения всех юзеров из базы, согласно параметрам
func (db *UserServiceDBRepository) GetAllUsers(ctx context.Context, includeInactive bool, limit, offset int) ([]*domain.User, int, error) {
	log.Printf("📊 [Repository] GetAllUsers: includeInactive=%v, limit=%d, offset=%d", includeInactive, limit, offset)

	// 1. Строим базовый запрос (без фильтра по организации!)
	baseQuery := `FROM users`
	args := []interface{}{}
	argIndex := 1

	if !includeInactive {
		baseQuery += ` WHERE status = $1`
		args = append(args, domain.UserStatusActive)
		argIndex++
	}

	// 2. Получаем общее количество
	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// 3. Получаем пользователей с пагинацией
	query := `
        SELECT id, email, password_hash, full_name, role, status, 
               telegram_chat_id, telegram_username, 
               organization_id, created_at, updated_at, last_login_at
    ` + baseQuery + `
        ORDER BY created_at DESC
        LIMIT $` + fmt.Sprint(argIndex) + ` OFFSET $` + fmt.Sprint(argIndex+1)

	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	// 4. Парсим результат
	users := make([]*domain.User, 0, limit)
	for rows.Next() {
		user := &domain.User{}
		var telegramChatID sql.NullInt64
		var telegramUsername sql.NullString
		var lastLoginAt sql.NullTime

		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
			&user.Role, &user.Status, &telegramChatID, &telegramUsername,
			&user.OrganizationID, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		// Обработка nullable полей
		if telegramChatID.Valid {
			user.TelegramID = &telegramChatID.Int64
		}
		if telegramUsername.Valid {
			user.TelegramUsername = &telegramUsername.String
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	log.Printf("✅ [Repository] GetAllUsers: получено %d пользователей (всего: %d)",
		len(users), total)

	return users, total, nil
}

// метод получения пользователей, согласно переданной роли
func (db *UserServiceDBRepository) GetUsersByRole(ctx context.Context, role domain.Role, includeInactive bool) ([]*domain.User, error) {
	log.Printf("📦 [Repository] GetUsersByRole: role=%s, includeInactive=%v", role, includeInactive)

	// 1. Строим запрос
	query := `
        SELECT id, email, password_hash, full_name, role, status, 
               telegram_chat_id, telegram_username, 
               organization_id, created_at, updated_at, last_login_at
        FROM users
        WHERE role = $1
    `
	args := []interface{}{string(role)}

	// 2. Сортировка (по умолчанию - по дате создания)
	query += ` ORDER BY created_at ASC`

	// 3. Выполняем запрос
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by role: %w", err)
	}
	defer rows.Close()

	// 5. Парсим результат
	users := make([]*domain.User, 0)
	for rows.Next() {
		user := &domain.User{}
		var telegramChatID sql.NullInt64
		var telegramUsername sql.NullString
		var lastLoginAt sql.NullTime

		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
			&user.Role, &user.Status, &telegramChatID, &telegramUsername,
			&user.OrganizationID, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Обработка nullable полей
		if telegramChatID.Valid {
			user.TelegramID = &telegramChatID.Int64
		}
		if telegramUsername.Valid {
			user.TelegramUsername = &telegramUsername.String
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	log.Printf("✅ [Repository] GetUsersByRole: получено %d пользователей с ролью %s",
		len(users), role)

	return users, nil
}

// метод получаения роли пользователя
func (db *UserServiceDBRepository) GetUserRole(ctx context.Context, userID string) (domain.Role, error) {
	log.Printf("📦 [Repository] GetUserRole: user_id=%s", userID)

	// 1. Проверка контекста
	select {
	case <-ctx.Done():
		return domain.RoleEmployee, ctx.Err()
	default:
	}

	// 2. Оптимизированный запрос - получаем только роль
	query := `SELECT role FROM users WHERE id = $1 AND deleted_at IS NULL`

	var role string
	err := db.Pool.QueryRow(ctx, query, userID).Scan(&role)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ [Repository] GetUserRole: пользователь %s не найден", userID)
			return domain.RoleEmployee, domain.ErrUserNotFound
		}
		log.Printf("❌ [Repository] GetUserRole: ошибка запроса: %v", err)
		return domain.RoleEmployee, fmt.Errorf("failed to get user role: %w", err)
	}

	log.Printf("✅ [Repository] GetUserRole: пользователь %s имеет роль %s", userID, role)

	return domain.Role(role), nil
}
