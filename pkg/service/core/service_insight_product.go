package core

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.InsightProductService = &insightProductService{}

type insightProductService struct {
	insightProductStorage service.InsightProductStorage
}

func (s *insightProductService) DeleteInsightProduct(ctx context.Context, id string) (*service.InsightProduct, error) {
	productUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	product, apiErr := s.GetInsightProduct(ctx, id)
	if apiErr != nil {
		return nil, apiErr
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(product.Group) {
		return nil, err
	}

	err = s.insightProductStorage.DeleteInsightProduct(ctx, productUUID)
	if err != nil {
		return nil, err
	}

	return product, nil
}

func (s *insightProductService) UpdateInsightProduct(ctx context.Context, id string, input service.UpdateInsightProductDto) (*service.InsightProduct, error) {
	productUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	existing, apierr := s.GetInsightProduct(ctx, id)
	if apierr != nil {
		return nil, apierr
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(existing.Group) {
		return nil, err
	}

	productSQL, err := s.insightProductStorage.UpdateInsightProduct(ctx, productUUID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to update insight product: %w", err)
	}

	return productSQL, nil
}

func (s *insightProductService) CreateInsightProduct(ctx context.Context, input service.NewInsightProduct) (*service.InsightProduct, error) {
	creator := auth.GetUser(ctx).Email

	productSQL, err := s.insightProductStorage.CreateInsightProduct(ctx, creator, input)
	if err != nil {
		return nil, err
	}

	return productSQL, nil
}

func (s *insightProductService) GetInsightProduct(ctx context.Context, id string) (*service.InsightProduct, error) {
	productUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse product id: %v", err)
	}

	productSQL, err := s.insightProductStorage.GetInsightProductWithTeamkatalogen(ctx, productUUID)
	if err != nil {
		return nil, err
	}

	return productSQL, nil
}

func NewInsightProductService(storage service.InsightProductStorage) *insightProductService {
	return &insightProductService{insightProductStorage: storage}
}
