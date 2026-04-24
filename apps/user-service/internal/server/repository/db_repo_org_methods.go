package repository

import (
	"context"
	"fmt"
	"user-service/internal/domain"
)

// ExistsAny - проверяет наличие хотя-бы 1 записи в таблице organizations
func (db *UserServiceDBRepository) ExistsAny(ctx context.Context) (bool, error) {
	sql := `SELECT EXISTS(SELECT 1 FROM organizations LIMIT 1)`
	var exists bool

	row := db.Pool.QueryRow(ctx, sql)
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check organizations existence: %w", err)
	}

	return exists, nil
}

// CreateOrg - создаёт запись об организации в БД
func (db *UserServiceDBRepository) CreateOrg(ctx context.Context, org *domain.Organization) error {
	sql := `
        INSERT INTO organizations (
            id, 
            name, 
            owner_id, 
            is_active, 
            created_at, 
            updated_at,
            deleted_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	// При создании deleted_at всегда nil (не удалена)
	_, err := db.Pool.Exec(ctx, sql,
		org.ID,
		org.Name,
		org.OwnerID,
		org.IsActive,
		org.CreatedAt,
		org.UpdatedAt,
		nil, // deleted_at = NULL
	)

	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	return nil
}

// DeleteOrganization - удаляет организацию из базы по ID
// DeleteOrganization - удаляет организацию, проверяя что нет зависимостей
func (db *UserServiceDBRepository) DeleteOrg(ctx context.Context, orgID string) error {
	// Начинаем транзакцию
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// 1. Проверяем, есть ли пользователи в организации
	checkUsersSQL := `SELECT COUNT(*) FROM users WHERE organization_id = $1 AND deleted_at IS NULL`
	var userCount int64
	row := tx.QueryRow(ctx, checkUsersSQL, orgID)
	if err := row.Scan(&userCount); err != nil {
		return fmt.Errorf("failed to check users: %w", err)
	}

	if userCount > 0 {
		return domain.ErrOrganizationHasUsers
	}

	// 2. Мягкое удаление организации
	deleteSQL := `
        UPDATE organizations 
        SET deleted_at = $1, updated_at = $1, is_active = false
        WHERE id = $2 AND deleted_at IS NULL
    `
	now := domain.Now()
	rowsAffected, err := tx.Exec(ctx, deleteSQL, now, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrOrganizationNotFound
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
