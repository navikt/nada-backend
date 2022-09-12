package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) UpsertProductArea(ctx context.Context, name, externalID string) error {
	_, err := r.querier.UpsertProductArea(ctx, gensql.UpsertProductAreaParams{
		Name:       name,
		ExternalID: externalID,
	})
	if err != nil {
		return err
	}

	return err
}

func (r *Repo) GetProductArea(ctx context.Context, id uuid.UUID) (*models.ProductArea, error) {
	pa, err := r.querier.GetProductArea(ctx, id)
	if err != nil {
		return nil, err
	}

	return productAreaFromSQL(pa), nil
}

func (r *Repo) GetProductAreaForExternalID(ctx context.Context, externalID string) (*models.ProductArea, error) {
	pa, err := r.querier.GetProductAreaForExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	return productAreaFromSQL(pa), nil
}

func (r *Repo) GetProductAreas(ctx context.Context) ([]*models.ProductArea, error) {
	pas, err := r.querier.GetAllProductAreas(ctx)
	if err != nil {
		return nil, err
	}

	pasGraph := make([]*models.ProductArea, len(pas))
	for idx, pa := range pas {
		pasGraph[idx] = productAreaFromSQL(pa)
	}

	return pasGraph, nil
}

func productAreaFromSQL(pa gensql.ProductArea) *models.ProductArea {
	return &models.ProductArea{
		ID:         pa.ID,
		ExternalID: pa.ExternalID,
		Name:       pa.Name,
	}
}
