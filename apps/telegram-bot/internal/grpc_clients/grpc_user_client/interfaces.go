package grpcuserclient

import (
	pb "api/gen/go/user/v1"
	"context"
)

// Полный интерфейс (опционально)
type FullUserGrpcService interface {
	PublicUserService
	UserProfileService
	UserManagementService
	OrganizationService
}

// 1. Базовые операции (публичные методы)
type PublicUserService interface {
	SetupInitialOrganization(ctx context.Context, req *pb.SetupInitialOrganizationRequest) (*pb.SetupInitialOrganizationResponse, error)
	LinkTelegram(ctx context.Context, req *pb.LinkTelegramRequest) (*pb.LinkTelegramResponse, error)
	HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error)
	Close() error
}

// 2. Профиль пользователя
type UserProfileService interface {
	GetMyProfile(ctx context.Context, req *pb.GetMyProfileRequest) (*pb.GetUserResponse, error)
	UpdateMyProfile(ctx context.Context, req *pb.UpdateMyProfileRequest) (*pb.GetUserResponse, error)
	Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error)
}

// 3. Управление пользователями (админские функции)
type UserManagementService interface {
	CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error)
	GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error)
	UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error)
	DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error)
	ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error)
}

// 4. Организация
type OrganizationService interface {
	GetOrganization(ctx context.Context, req *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error)
}
