package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *databasePollyResolver) ID(ctx context.Context, obj *models.DatabasePolly) (string, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Polly(ctx context.Context, q string) ([]*models.Polly, error) {
	return r.pollyAPI.SearchPolly(ctx, q)
}

// DatabasePolly returns generated.DatabasePollyResolver implementation.
func (r *Resolver) DatabasePolly() generated.DatabasePollyResolver { return &databasePollyResolver{r} }

type databasePollyResolver struct{ *Resolver }
