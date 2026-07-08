package grpcuserclient

import (
	pb "api/gen/go/user/v1" // Импортируем сгенерированные protobuf - это как контракт, по которому клиент и сервер будут общаться
	"context"
	"fmt"
	"telegram-bot/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// структура grpc клиета, который обращается к сервису user-service по grpc.
type UserGRPCClient struct {
	grpcClient pb.UserServiceClient
	conn       *grpc.ClientConn
}

func NewClient(cfg *config.GrpcClientConfig) (*UserGRPCClient, error) {
	// Создаем соединение с интерцепторами для авторизации
	conn, err := grpc.NewClient(
		cfg.GetAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	// Инициируем соединение
	conn.Connect()

	// проверяем состояние
	client := pb.NewUserServiceClient(conn)

	return &UserGRPCClient{
		grpcClient: client,
		conn:       conn,
	}, nil
}

// всегда закрываем соединение.
func (u *UserGRPCClient) Close() error {
	return u.conn.Close()
}

// SetupInitialOrganization - первичная настройка системы
// Вызывается при первом запуске для создания организации и владельца
// 🔓 ПУБЛИЧНЫЙ МЕТОД (без авторизации).
func (u *UserGRPCClient) SetupInitialOrganization(ctx context.Context, req *pb.SetupInitialOrganizationRequest) (*pb.SetupInitialOrganizationResponse, error) {
	return u.grpcClient.SetupInitialOrganization(ctx, req)
}

// LinkTelegram - привязка Telegram аккаунта к существующему пользователю
// Пользователь вводит свой email в боте, после чего происходит привязка
// 🔓 ПУБЛИЧНЫЙ МЕТОД (без авторизации)
// 🔥 ВОЗВРАЩАЕТ: JWT токен для дальнейшей работы.
func (u *UserGRPCClient) LinkTelegram(ctx context.Context, req *pb.LinkTelegramRequest) (*pb.LinkTelegramResponse, error) {
	return u.LinkTelegram(ctx, req)
}

// HealthCheck - проверка здоровья сервиса
// Используется для мониторинга и orchestration
// 🔓 ПУБЛИЧНЫЙ МЕТОД (без авторизации).
func (u *UserGRPCClient) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return u.HealthCheck(ctx, req)
}

// GetMyProfile - получение своего профиля
// Удобный метод для бота, чтобы не хранить user_id на клиенте
// 🔒 JWT авторизация (токен в metadata)
// 👤 ДОСТУП: Любой авторизованный пользователь.
func (u *UserGRPCClient) GetMyProfile(ctx context.Context, req *pb.GetMyProfileRequest) (*pb.GetUserResponse, error) {
	return u.GetMyProfile(ctx, req)
}

// UpdateMyProfile - обновление своего профиля
// Пользователь может обновить только свои данные (full_name)
// 🔒 JWT авторизация (токен в metadata)
// 👤 ДОСТУП: Любой авторизованный пользователь.
func (u *UserGRPCClient) UpdateMyProfile(ctx context.Context, req *pb.UpdateMyProfileRequest) (*pb.GetUserResponse, error) {
	return u.UpdateMyProfile(ctx, req)
}

// GetOrganization - получение информации об организации
// Всегда возвращает организацию текущего пользователя (из JWT)
// 🔒 JWT авторизация (токен в metadata)
// 👤 ДОСТУП: Любой авторизованный пользователь.
func (u *UserGRPCClient) GetOrganization(ctx context.Context, req *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error) {
	return u.GetOrganization(ctx, req)
}

// CreateUser - создание нового пользователя в организации
// 🔒 JWT авторизация + проверка роли
// 👤 ДОСТУП: Только MANAGER или OWNER.
func (u *UserGRPCClient) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	return u.CreateUser(ctx, req)
}

// GetUser - получение информации о пользователе по ID
// 🔒 JWT авторизация + проверка прав
// 👤 ДОСТУП: OWNER/MANAGER могут видеть всех, EMPLOYEE только себя.
func (u *UserGRPCClient) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	return u.GetUser(ctx, req)
}

// UpdateUser - обновление данных пользователя
// 🔒 JWT авторизация + проверка прав
// 👤 ДОСТУП: OWNER может менять любые поля, MANAGER - только некоторые, EMPLOYEE - только свой профиль.
func (u *UserGRPCClient) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	return u.UpdateUser(ctx, req)
}

// DeleteUser - удаление или деактивация пользователя
// 🔒 JWT авторизация + проверка роли
// 👤 ДОСТУП: Только OWNER
// ⚠️ Нельзя удалить самого себя.
func (u *UserGRPCClient) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	return u.DeleteUser(ctx, req)
}

// ListUsers - получение списка пользователей с фильтрацией и пагинацией
// 🔒 JWT авторизация + проверка прав
// 👤 ДОСТУП: OWNER и MANAGER (EMPLOYEE видит только себя).
func (u *UserGRPCClient) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	return u.ListUsers(ctx, req)
}

// Logout - выход из системы (отзыв JWT токена)
// 🔒 JWT авторизация (токен в metadata)
// 👤 ДОСТУП: Любой авторизованный пользователь.
func (u *UserGRPCClient) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return u.Logout(ctx, req)
}

// Проверяем, что клиент реализует все интерфейсы.
var _ PublicUserService = (*UserGRPCClient)(nil)
var _ UserProfileService = (*UserGRPCClient)(nil)
var _ UserManagementService = (*UserGRPCClient)(nil)
var _ OrganizationService = (*UserGRPCClient)(nil)
var _ FullUserGrpcService = (*UserGRPCClient)(nil)
