package converter

import (
	commonv1 "api/gen/go/common/v1"
	userv1 "api/gen/go/user/v1"
	"time"
	"user-service/internal/domain"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ========== Domain → Protobuf ==========

// ToProtoUser конвертирует доменную модель в protobuf (common.v1.User)
func ToProtoUser(domainUser *domain.User) *commonv1.User {
	if domainUser == nil {
		return nil
	}

	pbUser := &commonv1.User{
		Id:        domainUser.ID,
		Email:     domainUser.Email,
		FullName:  domainUser.FullName,
		Role:      toProtoRole(domainUser.Role),
		Status:    toProtoStatus(domainUser.Status),
		CreatedAt: timestamppb.New(domainUser.CreatedAt),
		UpdatedAt: timestamppb.New(domainUser.UpdatedAt),
	}

	// Опциональные поля
	if domainUser.TelegramID != nil {
		pbUser.TelegramId = domainUser.TelegramID
	}

	if domainUser.TelegramUsername != nil {
		pbUser.TelegramUsername = domainUser.TelegramUsername
	}

	if domainUser.LastLoginAt != nil {
		pbUser.LastLoginAt = timestamppb.New(*domainUser.LastLoginAt)
	}

	return pbUser
}

// ToProtoUsers конвертирует список доменных моделей
func ToProtoUsers(domainUsers []*domain.User) []*commonv1.User {
	if domainUsers == nil {
		return []*commonv1.User{}
	}

	pbUsers := make([]*commonv1.User, len(domainUsers))
	for i, user := range domainUsers {
		pbUsers[i] = ToProtoUser(user)
	}
	return pbUsers
}

// ToDomainUser конвертирует protobuf (common.v1.User) в доменную модель
func ToDomainUser(pbUser *commonv1.User) *domain.User {
	if pbUser == nil {
		return nil
	}

	domainUser := &domain.User{
		ID:        pbUser.Id,
		Email:     pbUser.Email,
		FullName:  pbUser.FullName,
		Role:      toDomainRole(pbUser.Role),
		Status:    toDomainStatus(pbUser.Status),
		CreatedAt: pbUser.CreatedAt.AsTime(),
		UpdatedAt: pbUser.UpdatedAt.AsTime(),
	}

	// Опциональные поля
	if pbUser.TelegramId != nil {
		domainUser.TelegramID = pbUser.TelegramId
	}

	if pbUser.TelegramUsername != nil {
		domainUser.TelegramUsername = pbUser.TelegramUsername
	}

	if pbUser.LastLoginAt != nil {
		lastLogin := pbUser.LastLoginAt.AsTime()
		domainUser.LastLoginAt = &lastLogin
	}

	return domainUser
}

// ========== Конвертация для запросов ==========

// ToDomainUserFromCreateRequest создает доменную модель из CreateUserRequest
// Обратите внимание: CreateUserRequest из user.v1, но User внутри - из common.v1
func ToDomainUserFromCreateRequest(req *userv1.CreateUserRequest) *domain.User {
	if req == nil {
		return nil
	}

	user := &domain.User{
		Email:     req.GetEmail(),
		FullName:  req.GetFullName(),
		Role:      toDomainRole(req.GetRole()),
		Status:    domain.UserStatusActive, // Новые пользователи активны по умолчанию
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.TelegramId != nil {
		user.TelegramID = req.TelegramId
	}

	return user
}

// ToDomainUserFromUpdateRequest создает map для обновления из UpdateUserRequest
func ToDomainUserUpdates(req *userv1.UpdateUserRequest) map[string]interface{} {
	updates := make(map[string]interface{})

	if req.Email != nil {
		updates["email"] = req.GetEmail()
	}

	if req.FullName != nil {
		updates["full_name"] = req.GetFullName()
	}

	if req.Role != nil {
		updates["role"] = toDomainRole(req.GetRole())
	}

	if req.Status != nil {
		updates["status"] = toDomainStatus(req.GetStatus())
	}

	updates["updated_at"] = time.Now()

	return updates
}

// ========== Конвертация для BatchGetUsersResponse ==========

// ToProtoUserMap конвертирует map domain пользователей в map protobuf
func ToProtoUserMap(domainUsers map[string]*domain.User) map[string]*commonv1.User {
	if domainUsers == nil {
		return make(map[string]*commonv1.User)
	}

	pbUsers := make(map[string]*commonv1.User)
	for id, user := range domainUsers {
		pbUsers[id] = ToProtoUser(user)
	}
	return pbUsers
}

// ========== Конвертация enum (common.v1.Role ↔ domain.Role) ==========

func toProtoRole(role domain.Role) commonv1.Role {
	switch role {
	case domain.RoleOwner:
		return commonv1.Role_ROLE_OWNER
	case domain.RoleManager:
		return commonv1.Role_ROLE_MANAGER
	case domain.RoleEmployee:
		return commonv1.Role_ROLE_EMPLOYEE
	default:
		return commonv1.Role_ROLE_UNSPECIFIED
	}
}

func toDomainRole(role commonv1.Role) domain.Role {
	switch role {
	case commonv1.Role_ROLE_OWNER:
		return domain.RoleOwner
	case commonv1.Role_ROLE_MANAGER:
		return domain.RoleManager
	case commonv1.Role_ROLE_EMPLOYEE:
		return domain.RoleEmployee
	default:
		return domain.RoleEmployee
	}
}

// ========== Конвертация enum (common.v1.UserStatus ↔ domain.UserStatus) ==========

func toProtoStatus(status domain.UserStatus) commonv1.UserStatus {
	switch status {
	case domain.UserStatusActive:
		return commonv1.UserStatus_USER_STATUS_ACTIVE
	case domain.UserStatusInactive:
		return commonv1.UserStatus_USER_STATUS_INACTIVE
	case domain.UserStatusSuspended:
		return commonv1.UserStatus_USER_STATUS_SUSPENDED
	default:
		return commonv1.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

func toDomainStatus(status commonv1.UserStatus) domain.UserStatus {
	switch status {
	case commonv1.UserStatus_USER_STATUS_ACTIVE:
		return domain.UserStatusActive
	case commonv1.UserStatus_USER_STATUS_INACTIVE:
		return domain.UserStatusInactive
	case commonv1.UserStatus_USER_STATUS_SUSPENDED:
		return domain.UserStatusSuspended
	default:
		return domain.UserStatusActive
	}
}
