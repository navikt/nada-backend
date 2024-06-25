package core

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.InsightProductService = &insightProductService{}

type insightProductService struct {
	insightProductStorage service.InsightProductStorage
}

func (s *insightProductService) DeleteInsightProduct(ctx context.Context, id uuid.UUID) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductService.DeleteInsightProduct"

	product, err := s.GetInsightProduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(product.Group) {
		return nil, errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user not authorized to delete product"))
	}

	err = s.insightProductStorage.DeleteInsightProduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return product, nil
}

func (s *insightProductService) UpdateInsightProduct(ctx context.Context, id uuid.UUID, input service.UpdateInsightProductDto) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductService.UpdateInsightProduct"

	existing, err := s.GetInsightProduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	// FIXME: move up the call chain
	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(existing.Group) {
		return nil, errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user not authorized to update product"))
	}

	productSQL, err := s.insightProductStorage.UpdateInsightProduct(ctx, id, input)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return productSQL, nil
}

func (s *insightProductService) CreateInsightProduct(ctx context.Context, input service.NewInsightProduct) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductService.CreateInsightProduct"

	creator := auth.GetUser(ctx).Email

	productSQL, err := s.insightProductStorage.CreateInsightProduct(ctx, creator, input)
	if err != nil {
		return nil, errs.E(op, errs.UserName(creator), err)
	}

	return productSQL, nil
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
