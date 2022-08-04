package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

// Teamkatalogen is the resolver for the teamkatalogen field.
func (r *queryResolver) Teamkatalogen(ctx context.Context, q string) ([]*models.TeamkatalogenResult, error) {
	return r.teamkatalogen.Search(ctx, q)
}
