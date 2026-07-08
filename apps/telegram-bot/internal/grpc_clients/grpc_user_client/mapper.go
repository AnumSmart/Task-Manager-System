package grpcuserclient

import (
	commonpb "api/gen/go/common/v1"
	userpb "api/gen/go/user/v1"
	"telegram-bot/internal/domain"
	"time"
)

type UserMapper struct{}

func NewUserMapper() *UserMapper {
	return &UserMapper{}
}

// ===== Конвертация User =====

// ToBotUser - конвертирует common.v1.User в доменную модель бота.
func (m *UserMapper) ToBotUser(pbUser *commonpb.User) *domain.BotUser {
	if pbUser == nil {
		return nil
	}

	var telegramID *int64
	if pbUser.TelegramId != nil {
		telegramID = pbUser.TelegramId
	}

	var telegramUsername *string
	if pbUser.TelegramUsername != nil {
		telegramUsername = pbUser.TelegramUsername
	}

	var lastLoginAt *time.Time

	if pbUser.GetLastLoginAt() != nil {
		t := pbUser.GetLastLoginAt().AsTime()
		lastLoginAt = &t
	}

	return &domain.BotUser{
		ID:               pbUser.GetId(),
		Email:            pbUser.GetEmail(),
		TelegramID:       telegramID,
		TelegramUsername: telegramUsername,
		Role:             m.toDomainRole(pbUser.GetRole()),
		Status:           m.toDomainStatus(pbUser.GetStatus()),
		FullName:         pbUser.GetFullName(),
		CreatedAt:        pbUser.GetCreatedAt().AsTime(),
		UpdatedAt:        pbUser.GetUpdatedAt().AsTime(),
		LastLoginAt:      lastLoginAt,
	}
}

// ToBotUsers - конвертирует список пользователей.
func (m *UserMapper) ToBotUsers(pbUsers []*commonpb.User) []*domain.BotUser {
	if len(pbUsers) == 0 {
		return []*domain.BotUser{}
	}

	result := make([]*domain.BotUser, 0, len(pbUsers))

	for _, pbUser := range pbUsers {
		if user := m.ToBotUser(pbUser); user != nil {
			result = append(result, user)
		}
	}

	return result
}

// ===== Конвертация Organization =====

// ToBotOrganization - конвертирует common.v1.Organization в доменную модель.
func (m *UserMapper) ToBotOrganization(pbOrg *commonpb.Organization) *domain.BotOrganization {
	if pbOrg == nil {
		return nil
	}

	return &domain.BotOrganization{
		ID:        pbOrg.GetId(),
		Name:      pbOrg.GetName(),
		IsActive:  pbOrg.GetIsActive(),
		CreatedAt: pbOrg.GetCreatedAt().AsTime(),
		UpdatedAt: pbOrg.GetUpdatedAt().AsTime(),
		OwnerID:   pbOrg.GetOwnerId(),
	}
}

// ===== Конвертация JWT =====

// ToJWTInfo - конвертирует ответ привязки Telegram.
func (m *UserMapper) ToJWTInfo(resp *userpb.LinkTelegramResponse) *domain.JWTInfo {
	if resp == nil || !resp.GetSuccess() {
		return nil
	}

	return &domain.JWTInfo{
		Token:        resp.GetJwtToken(),
		RefreshToken: resp.GetRefreshToken(),
		ExpiresIn:    resp.GetExpiresIn(),
		Claims:       m.ToJWTClaims(resp.GetClaims()),
	}
}

// ToJWTClaims - конвертирует common.v1.JWTClaims в доменную модель.
func (m *UserMapper) ToJWTClaims(pbClaims *commonpb.JWTClaims) *domain.JWTClaims {
	if pbClaims == nil {
		return nil
	}

	var telegramID *int64
	if pbClaims.TelegramId != nil {
		telegramID = pbClaims.TelegramId
	}

	return &domain.JWTClaims{
		UserID:         pbClaims.GetUserId(),
		Email:          pbClaims.GetEmail(),
		Role:           m.toDomainRole(pbClaims.GetRole()),
		OrganizationID: pbClaims.GetOrganizationId(),
		TelegramID:     telegramID,
		TokenID:        pbClaims.GetTokenId(),
		IssuedAt:       pbClaims.GetIssuedAt(),
		ExpiresAt:      pbClaims.GetExpiresAt(),
	}
}

// ===== Конвертация Pagination =====

// ToPagination - конвертирует common.v1.Pagination.
func (m *UserMapper) ToPagination(pbPagination *commonpb.Pagination) *Pagination {
	if pbPagination == nil {
		return nil
	}

	return &Pagination{
		Page:       pbPagination.GetPage(),
		PageSize:   pbPagination.GetPageSize(),
		Total:      pbPagination.GetTotal(),
		TotalPages: pbPagination.GetTotalPages(),
	}
}

// ===== Вспомогательные методы конвертации enum'ов =====

func (m *UserMapper) toDomainRole(role commonpb.Role) domain.UserRole {
	switch role {
	case commonpb.Role_ROLE_OWNER:
		return domain.RoleOwner
	case commonpb.Role_ROLE_MANAGER:
		return domain.RoleManager
	case commonpb.Role_ROLE_EMPLOYEE:
		return domain.RoleEmployee
	default:
		return domain.RoleUnspecified
	}
}

func (m *UserMapper) toDomainStatus(status commonpb.UserStatus) domain.UserStatus {
	switch status {
	case commonpb.UserStatus_USER_STATUS_ACTIVE:
		return domain.UserStatusActive
	case commonpb.UserStatus_USER_STATUS_INACTIVE:
		return domain.UserStatusInactive
	case commonpb.UserStatus_USER_STATUS_SUSPENDED:
		return domain.UserStatusSuspended
	default:
		return domain.UserStatusUnspecified
	}
}

// Обратная конвертация (Domain → gRPC) - если нужно отправлять запросы.
func (m *UserMapper) ToGRPCRole(role domain.UserRole) commonpb.Role {
	switch role {
	case domain.RoleOwner:
		return commonpb.Role_ROLE_OWNER
	case domain.RoleManager:
		return commonpb.Role_ROLE_MANAGER
	case domain.RoleEmployee:
		return commonpb.Role_ROLE_EMPLOYEE
	default:
		return commonpb.Role_ROLE_UNSPECIFIED
	}
}

func (m *UserMapper) ToGRPCStatus(status domain.UserStatus) commonpb.UserStatus {
	switch status {
	case domain.UserStatusActive:
		return commonpb.UserStatus_USER_STATUS_ACTIVE
	case domain.UserStatusInactive:
		return commonpb.UserStatus_USER_STATUS_INACTIVE
	case domain.UserStatusSuspended:
		return commonpb.UserStatus_USER_STATUS_SUSPENDED
	default:
		return commonpb.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

// ===== Дополнительные структуры =====

// Pagination - модель пагинации для бота.
type Pagination struct {
	Page       int32
	PageSize   int32
	Total      int32
	TotalPages int32
}
