package service

import "user-service/internal/server/repository"

// конструктор для части сервисного слоя, который отвечает за работу с организацией
type OrganizationLayer struct {
	OrgRepo repository.UserDBRepository
}

// констркутор для части сервисного слоя (организации)
// в конструктор передаём составной репозиторий (на будущее)
func NewOrganisationLayer(repo *repository.UserServiceRepository) *OrganizationLayer {
	return &OrganizationLayer{
		OrgRepo: repo.DBRepo,
	}
}
