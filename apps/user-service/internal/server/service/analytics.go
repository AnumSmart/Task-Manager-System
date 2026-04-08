package service

import "user-service/internal/server/repository"

// конструктор для части сервисного слоя, который отвечает за работу с аналитикой
type AnalyticsLayer struct {
	AnRepo repository.UserDBRepository
}

// констркутор для части сервисного слоя (аналитика)
// в конструктор передаём составной репозиторий (на будущее)
func NewAnalyticsLayer(repo *repository.UserServiceRepository) *AnalyticsLayer {
	return &AnalyticsLayer{
		AnRepo: repo.DBRepo,
	}
}
