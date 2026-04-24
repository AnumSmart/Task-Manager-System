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
