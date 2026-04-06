package handler

import (
	pb "api/gen/go/user/v1" // Импортируем сгенерированные protobuf - это как контракт, по которому клиент и сервер будут общаться
	"user-service/internal/server/service"
)

// структура слоя хэндлеров для пользователей
type UserServerHandler struct {
	pb.UnimplementedUserServiceServer
	UserServerService *service.UserService
}

// конструктор для слоя хэндлеров (пользователи)
func NewUserServerHandler(service *service.UserService) *UserServerHandler {
	return &UserServerHandler{
		UserServerService: service,
	}
}
