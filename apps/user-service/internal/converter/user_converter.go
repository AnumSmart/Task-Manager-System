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
		Role:      ToProtoRole(domainUser.Role),
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
		Role:      ToDomainRole(pbUser.Role),
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

// ToProtoUserWithPermissions конвертирует доменную модель в protobuf с учётом прав
func ToProtoUserWithPermissions(targetUser, currentUser *domain.User) *commonv1.User {
	if targetUser == nil {
		return nil
	}

	// Всегда конвертируем базовые поля
	pbUser := &commonv1.User{
		Id:        targetUser.ID,
		FullName:  targetUser.FullName,
		CreatedAt: timestamppb.New(targetUser.CreatedAt),
		UpdatedAt: timestamppb.New(targetUser.UpdatedAt),
	}

	// Если смотрим свои данные - показываем всё
	if currentUser != nil && currentUser.ID == targetUser.ID {
		pbUser.Email = targetUser.Email
		pbUser.Role = ToProtoRole(targetUser.Role)
		pbUser.Status = toProtoStatus(targetUser.Status)
		if targetUser.TelegramID != nil {
			pbUser.TelegramId = targetUser.TelegramID
		}
		if targetUser.TelegramUsername != nil {
			pbUser.TelegramUsername = targetUser.TelegramUsername
		}
		if targetUser.LastLoginAt != nil {
			pbUser.LastLoginAt = timestamppb.New(*targetUser.LastLoginAt)
		}
		return pbUser
	}

	// Если текущий пользователь Owner или Manager - показываем почти всё
	if currentUser != nil && (currentUser.Role == domain.RoleOwner || currentUser.Role == domain.RoleManager) {
		pbUser.Email = targetUser.Email
		pbUser.Role = ToProtoRole(targetUser.Role)
		pbUser.Status = toProtoStatus(targetUser.Status)
		// Но скрываем Telegram данные для чужих пользователей (конфиденциальность)
		// TelegramId и TelegramUsername не копируем
		return pbUser
	}

	// Employee видит только базовые поля (id, имя, даты)
	return pbUser
}

// ========== Конвертация для запросов ==========

// ToDomainUserFromCreateRequest создает доменную модель из CreateUserRequest
// Обратите внимание: CreateUserRequest из user.v1, но User внутри - из common.v1
func ToDomainUserFromCreateRequest(req *userv1.CreateUserRequest, organizationID string) *domain.User {
	if req == nil {
		return nil
	}

	user := &domain.User{
		Email:          req.GetEmail(),
		FullName:       req.GetFullName(),
		Role:           ToDomainRole(req.GetRole()),
		Status:         domain.UserStatusActive, // Новые пользователи активны по умолчанию
		OrganizationID: organizationID,          // ← приходит из JWT
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		// PasswordHash будет установлен в сервисе
	}

	if req.TelegramId != nil {
		user.TelegramID = req.TelegramId
	}

	return user
}

// ToDomainUserFromCreateRequest создает доменную модель из CreateUserRequest
// Обратите внимание: CreateUserRequest из user.v1, но User внутри - из common.v1
func ToDomainIncUserFromCreateRequest(req *userv1.CreateUserRequest, organizationID string) *domain.IncomingUser {
	if req == nil {
		return nil
	}

	user := &domain.IncomingUser{
		Email:          req.GetEmail(),
		FullName:       req.GetFullName(),
		Role:           ToDomainRole(req.GetRole()),
		Status:         domain.UserStatusActive, // Новые пользователи активны по умолчанию
		OrganizationID: organizationID,          // ← приходит из JWT
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Password:       req.Password,
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
		updates["role"] = ToDomainRole(req.GetRole())
	}

	if req.Status != nil {
		updates["status"] = toDomainStatus(req.GetStatus())
	}

	return updates
}

// ========== Конвертация для BatchGetUsersResponse ==========

// ToProtoUserMap конвертирует map доменных пользователей в map protobuf пользователей
func ToProtoUserMap(domainUsers map[string]*domain.User) map[string]*commonv1.User {
	if domainUsers == nil {
		return make(map[string]*commonv1.User)
	}

	pbUsers := make(map[string]*commonv1.User, len(domainUsers))
	for id, user := range domainUsers {
		pbUsers[id] = ToProtoUser(user)
	}
	return pbUsers
}

// ToDomainUserMap конвертирует map protobuf пользователей в map доменных пользователей
// (если понадобится обратная конвертация)
func ToDomainUserMap(pbUsers map[string]*commonv1.User) map[string]*domain.User {
	if pbUsers == nil {
		return make(map[string]*domain.User)
	}

	domainUsers := make(map[string]*domain.User, len(pbUsers))
	for id, pbUser := range pbUsers {
		domainUsers[id] = ToDomainUser(pbUser)
	}
	return domainUsers
}

// ToProtoUserSlice конвертирует slice доменных пользователей в slice protobuf
func ToProtoUserSlice(domainUsers []*domain.User) []*commonv1.User {
	if domainUsers == nil {
		return []*commonv1.User{}
	}

	pbUsers := make([]*commonv1.User, len(domainUsers))
	for i, user := range domainUsers {
		pbUsers[i] = ToProtoUser(user)
	}
	return pbUsers
}

// ToDomainUserSlice конвертирует slice protobuf пользователей в slice доменных
func ToDomainUserSlice(pbUsers []*commonv1.User) []*domain.User {
	if pbUsers == nil {
		return []*domain.User{}
	}

	domainUsers := make([]*domain.User, len(pbUsers))
	for i, pbUser := range pbUsers {
		domainUsers[i] = ToDomainUser(pbUser)
	}
	return domainUsers
}

// ========== Конвертация enum (common.v1.Role ↔ domain.Role) ==========

func ToProtoRole(role domain.Role) commonv1.Role {
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

func ToDomainRole(role commonv1.Role) domain.Role {
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

// ToDomainRoleFilter конвертирует optional protobuf Role в указатель на domain.Role
func ToDomainRoleFilter(role *commonv1.Role) *domain.Role {
	if role == nil {
		return nil
	}
	converted := ToDomainRole(*role)
	return &converted
}

// ToDomainStatusFilter конвертирует optional protobuf UserStatus в указатель на domain.UserStatus
func ToDomainStatusFilter(status *commonv1.UserStatus) *domain.UserStatus {
	if status == nil {
		return nil
	}
	converted := toDomainStatus(*status)
	return &converted
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

// ========== запросы списка пользователей ==========

// ToDomainListUsersRequest конвертирует protobuf ListUsersRequest в доменный ListUsersRequest
func ToDomainListUsersRequest(requesterID, organizationID string, req *userv1.ListUsersRequest) *domain.ListUsersRequest {
	if req == nil {
		return nil
	}

	// Нормализация параметров пагинации
	page := req.GetPage()
	pageSize := req.GetPageSize()
	offset, limit := ToPaginationParams(page, pageSize)

	// Формируем фильтры
	filters := make(map[string]string)

	// Используем существующие функции для конвертации optional полей
	if roleFilter := ToDomainRoleFilter(req.Role); roleFilter != nil {
		filters["role"] = string(*roleFilter)
	}

	if statusFilter := ToDomainStatusFilter(req.Status); statusFilter != nil {
		filters["status"] = string(*statusFilter)
	}

	// Если есть поисковый запрос (если добавите в proto позже)
	// if req.Search != nil {
	//     filters["search"] = req.GetSearch()
	// }

	return &domain.ListUsersRequest{
		OrganizationID: organizationID,
		RequesterID:    requesterID,
		Filters:        filters,
		Pagination: domain.Pagination{
			Offset: offset,
			Limit:  limit,
		},
	}
}

// ToPaginationParams - вспомогательная функция для нормализации параметров пагинации
func ToPaginationParams(page, pageSize int32) (offset, limit int) {
	normalizedPage := page
	normalizedPageSize := pageSize

	if normalizedPage < 1 {
		normalizedPage = 1
	}
	if normalizedPageSize < 1 {
		normalizedPageSize = 20
	}
	if normalizedPageSize > 100 {
		normalizedPageSize = 100
	}

	offset = int((normalizedPage - 1) * normalizedPageSize)
	limit = int(normalizedPageSize)

	return offset, limit
}

// ToProtoListUsersResponse конвертирует доменный ListUsersResponse в protobuf
func ToProtoListUsersResponse(resp *domain.ListUsersResponse, currentUser *domain.User, page, pageSize int32) *userv1.ListUsersResponse {
	if resp == nil {
		return &userv1.ListUsersResponse{
			Success:      false,
			ErrorMessage: "пустой ответ",
		}
	}

	// Конвертируем пользователей
	pbUsers := make([]*commonv1.User, 0, len(resp.Users))
	for _, user := range resp.Users {
		pbUsers = append(pbUsers, ToProtoUserWithPermissions(user, currentUser))
	}

	// Рассчитываем пагинацию
	totalPages := (resp.TotalCount + int(pageSize) - 1) / int(pageSize)

	return &userv1.ListUsersResponse{
		Success: true,
		Users:   pbUsers,
		Pagination: &commonv1.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      int32(resp.TotalCount),
			TotalPages: int32(totalPages),
		},
		ErrorMessage: "",
	}
}

// ToProtoListUsersResponseForSingleUser создает ListUsersResponse для одного пользователя
// Используется для Employee, который видит только свой профиль
func ToProtoListUsersResponseForSingleUser(user *domain.User) *userv1.ListUsersResponse {
	if user == nil {
		return &userv1.ListUsersResponse{
			Success:      false,
			ErrorMessage: "пользователь не найден",
		}
	}

	return &userv1.ListUsersResponse{
		Success: true,
		Users:   []*commonv1.User{ToProtoUser(user)},
		Pagination: &commonv1.Pagination{
			Page:       1,
			PageSize:   1,
			Total:      1,
			TotalPages: 1,
		},
		ErrorMessage: "",
	}
}
