package handler

import (
	pb "api/gen/go/user/v1" // Импортируем сгенерированные protobuf - это как контракт, по которому клиент и сервер будут общаться
	"sync/atomic"
	"user-service/internal/server/service"
)

// структура слоя хэндлеров для пользователей
type UserServerHandler struct {
	pb.UnimplementedUserServiceServer
	UserServerService *service.UserService
	isShuttingDown    atomic.Bool // флаг о начале gracefull shutDown
}

// конструктор для слоя хэндлеров (пользователи)
func NewUserServerHandler(service *service.UserService) *UserServerHandler {
	return &UserServerHandler{
		UserServerService: service,
	}
}

// Установка флага о том, что начался Gracefull ShutDown
func (s *UserServerHandler) IsShuttingDown() bool {
	return s.isShuttingDown.Load()
}
