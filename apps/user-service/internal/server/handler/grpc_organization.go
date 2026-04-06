package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"log"
)

// SetupInitialOrganization - вызывается при первом запуске для создания организации и владельца
func (s *UserServerHandler) SetupInitialOrganization(ctx context.Context, req *pb.SetupInitialOrganizationRequest) (*pb.SetupInitialOrganizationResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	return &pb.SetupInitialOrganizationResponse{Success: true}, nil
}

// GetOrganization - получение информации об организации
func (s *UserServerHandler) GetOrganization(ctx context.Context, req *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	return &pb.GetOrganizationResponse{}, nil
}
