package core

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.InsightProductService = &insightProductService{}

type insightProductService struct {
	insightProductStorage service.InsightProductStorage
}

func (s *insightProductService) DeleteInsightProduct(ctx context.Context, user *service.User, id uuid.UUID) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductService.DeleteInsightProduct"

	product, err := s.GetInsightProduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if !user.GoogleGroups.Contains(product.Group) {
		return nil, errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user not authorized to delete product"))
	}

	err = s.insightProductStorage.DeleteInsightProduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return product, nil
}

func (s *insightProductService) UpdateInsightProduct(ctx context.Context, user *service.User, id uuid.UUID, input service.UpdateInsightProductDto) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductService.UpdateInsightProduct"

	existing, err := s.GetInsightProduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if !user.GoogleGroups.Contains(existing.Group) {
		return nil, errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user not authorized to update product"))
	}

	productSQL, err := s.insightProductStorage.UpdateInsightProduct(ctx, id, input)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return productSQL, nil
}

func (s *insightProductService) CreateInsightProduct(ctx context.Context, user *service.User, input service.NewInsightProduct) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductService.CreateInsightProduct"

	ip, err := s.insightProductStorage.CreateInsightProduct(ctx, user.Email, input)
	if err != nil {
		return nil, errs.E(op, errs.UserName(user.Email), err)
	}

	return ip, nil
}

func (s *insightProductService) GetInsightProduct(ctx context.Context, id uuid.UUID) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductService.GetInsightProduct"

	product, err := s.insightProductStorage.GetInsightProductWithTeamkatalogen(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return product, nil
}

func NewInsightProductService(storage service.InsightProductStorage) *insightProductService {
	return &insightProductService{insightProductStorage: storage}
}
