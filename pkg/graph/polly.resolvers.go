package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *queryResolver) Polly(ctx context.Context, q string) ([]*models.QueryPolly, error) {
	return r.pollyAPI.SearchPolly(ctx, q)
}
