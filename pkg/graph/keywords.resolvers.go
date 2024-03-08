package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.44

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

// UpdateKeywords is the resolver for the updateKeywords field.
func (r *mutationResolver) UpdateKeywords(ctx context.Context, input models.UpdateKeywords) (bool, error) {
	err := ensureUserInGroup(ctx, "nada@nav.no")
	if err != nil {
		return false, err
	}

	err = r.repo.UpdateKeywords(ctx, input)
	if err != nil {
		return false, err
	}

	return true, nil
}

// Keywords is the resolver for the keywords field.
func (r *queryResolver) Keywords(ctx context.Context) ([]*models.Keyword, error) {
	return r.repo.KeywordsSortedByPopularity(ctx)
}
