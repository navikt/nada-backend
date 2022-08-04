package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

// Quartos is the resolver for the quartos field.
func (r *queryResolver) Quartos(ctx context.Context) ([]*models.Quarto, error) {
	return r.repo.GetQuartos(ctx)
}

// Quarto is the resolver for the quarto field.
func (r *queryResolver) Quarto(ctx context.Context, id uuid.UUID) (*models.Quarto, error) {
	data, err := r.repo.GetQuarto(ctx, id)
	if err != nil {
		return nil, err
	}

	fmt.Println(data)
	return data, err
}
