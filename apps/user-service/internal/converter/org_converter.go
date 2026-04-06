package converter

import (
	commonv1 "api/gen/go/common/v1"
	"user-service/internal/domain"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ========== Конвертация для Organization ==========

func ToProtoOrganization(domainOrg *domain.Organization) *commonv1.Organization {
	if domainOrg == nil {
		return nil
	}

	return &commonv1.Organization{
		Id:        domainOrg.ID,
		Name:      domainOrg.Name,
		CreatedAt: timestamppb.New(domainOrg.CreatedAt),
		UpdatedAt: timestamppb.New(domainOrg.UpdatedAt),
	}
}

func ToDomainOrganization(pbOrg *commonv1.Organization) *domain.Organization {
	if pbOrg == nil {
		return nil
	}

	return &domain.Organization{
		ID:        pbOrg.Id,
		Name:      pbOrg.Name,
		CreatedAt: pbOrg.CreatedAt.AsTime(),
		UpdatedAt: pbOrg.UpdatedAt.AsTime(),
	}
}
