package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

// ProductArea is the resolver for the productArea field.
func (r *queryResolver) ProductArea(ctx context.Context, id uuid.UUID) (*models.ProductArea, error) {
	return r.repo.GetProductArea(ctx, id)
}

// ProductAreas is the resolver for the productAreas field.
func (r *queryResolver) ProductAreas(ctx context.Context) ([]*models.ProductArea, error) {
	// temporarily return only one product area
	pas, err := r.repo.GetProductAreas(ctx)
	if err != nil {
		return nil, err
	}

	return []*models.ProductArea{pas[0]}, err
}
