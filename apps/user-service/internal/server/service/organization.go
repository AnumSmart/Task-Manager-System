package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"user-service/internal/domain"
	"user-service/internal/server/repository"
)

// конструктор для части сервисного слоя, который отвечает за работу с организацией
type OrganizationLayer struct {
	OrgRepo repository.OrganizationDBRepository // использую интерфейс из repo слоя
	Cache   repository.UserCacheRepository      // использую интерфейс из repo слоя
}

// констркутор для части сервисного слоя (организации)
// в конструктор передаём составной репозиторий (на будущее)
func NewOrganisationLayer(repo *repository.UserServiceRepository) *OrganizationLayer {
	return &OrganizationLayer{
		OrgRepo: repo.DBRepo,
		Cache:   repo.CacheRepo,
	}
}

// IsInitialized - проверяет, что организация уже есть в базе
func (o *OrganizationLayer) IsInitialized(ctx context.Context) (bool, error) {
	log.Printf("📝 Checking if system is initialized")

	// Вызываем метод репозитория для проверки существования хотя бы одной организации
	exists, err := o.OrgRepo.ExistsAny(ctx)
	if err != nil {
		log.Printf("❌ Failed to check initialization: %v", err)
		return false, fmt.Errorf("failed to check system initialization: %w", err)
	}

	if exists {
		log.Printf("✅ System already initialized (organization exists)")
	} else {
		log.Printf("✅ System not initialized (no organizations found)")
	}

	return exists, nil
}

// SaveOrganization - сохраняет сформированную организацию через репо слой в базу данных
func (o *OrganizationLayer) SaveOrganization(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	// 1. Валидация
	if err := org.Validate(); err != nil {
		return nil, err
	}

	// 2. Сохранение в БД через репозиторий
	if err := o.OrgRepo.CreateOrg(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to save organization: %w", err)
	}

	// 3. Возвращаем сохранённую организацию (обычно ту же, что и передали, но с обновлёнными полями)
	return org, nil
}

// Удаляет организацию
func (o *OrganizationLayer) DeleteOrganization(ctx context.Context, orgID string) error {
	log.Printf("📝 Deleting organization: orgID=%s", orgID)

	// Вызываем метод репозитория для удаления
	if err := o.OrgRepo.DeleteOrg(ctx, orgID); err != nil {
		if errors.Is(err, domain.ErrOrganizationHasUsers) {
			return domain.ErrOrganizationHasUsers
		}
		if errors.Is(err, domain.ErrOrganizationNotFound) {
			return domain.ErrOrganizationNotFound
		}
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	// Очищаем кэш
	if err := o.Cache.Delete(ctx, "organization:"+orgID); err != nil {
		log.Printf("⚠️ Warning: failed to delete organization from cache: %v", err)
	}

	log.Printf("✅ Organization deleted successfully: orgID=%s", orgID)
	return nil
}

// Обновляет владельца организации
func (o *OrganizationLayer) UpdateOwner(ctx context.Context, orgID, ownerID string) error {
	// 1. Валидация
	if orgID == "" {
		return domain.ErrInvalidOrganizationID
	}
	if ownerID == "" {
		return domain.ErrInvalidUserID
	}

	// 2. Проверяем, существует ли организация
	org, err := o.OrgRepo.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}
	if org == nil {
		return domain.ErrOrganizationNotFound
	}

	// 3. Обновляем OwnerID в репозитории
	err = o.OrgRepo.UpdateOwner(ctx, orgID, ownerID)
	if err != nil {
		return err
	}

	// 4. Логируем (опционально)
	log.Printf("✅ Organization owner updated: org_id=%s, owner_id=%s", orgID, ownerID)

	return nil
}

// ActivateOrganization - активирует организацию
func (o *OrganizationLayer) ActivateOrganization(ctx context.Context, orgID string, owner *domain.User) error {
	// 1. Валидация
	if orgID == "" {
		return domain.ErrInvalidOrganizationID
	}
	if owner.ID == "" {
		return domain.ErrInvalidUserID
	}

	// 2. Получаем организацию
	org, err := o.OrgRepo.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}
	if org == nil {
		return domain.ErrOrganizationNotFound
	}

	// 3. Проверяем, что владелец организации установлен (не nil)
	if org.OwnerID == nil {
		return domain.ErrOrganizationNoOwner
	}

	// 4. Проверяем, что владелец совпадает с переданным пользователем
	if *org.OwnerID != owner.ID {
		return domain.ErrNotOrganizationOwner
	}

	// 5. Проверяем, что владелец имеет роль OWNER
	if owner.Role != domain.RoleOwner {
		return domain.ErrUserIsNotOwner
	}

	// 6. Активируем организацию
	err = o.OrgRepo.Activate(ctx, orgID)
	if err != nil {
		return err
	}

	log.Printf("✅ Organization activated: org_id=%s, owner_id=%s", orgID, owner.ID)
	return nil
}

// получает ID организации
// GetOrganizationByID - получение организации по ID
func (s *OrganizationLayer) GetOrganizationByID(ctx context.Context, orgID string) (*domain.Organization, error) {
	// Валидация входных данных
	if orgID == "" {
		return nil, domain.ErrInvalidOrganizationID
	}

	// Вызов метода репозитория
	org, err := s.OrgRepo.GetOrganizationByID(ctx, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizationNotFound) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to get organization by id: %w", err)
	}

	return org, nil
}
