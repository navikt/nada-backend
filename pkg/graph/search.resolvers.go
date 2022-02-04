package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *queryResolver) Search(ctx context.Context, q *models.SearchQueryOld, options *models.SearchQuery) ([]*models.SearchResultRow, error) {
	if q == nil {
		q = &models.SearchQueryOld{}
	}
	if options == nil {
		options = &models.SearchQuery{
			Text:   q.Text,
			Limit:  q.Limit,
			Offset: q.Offset,
			Types: []models.SearchType{
				models.SearchTypeDataproduct,
			},
		}

		if q.Keyword != nil {
			options.Keywords = []string{*q.Keyword}
		}
		if q.Group != nil {
			options.Groups = []string{*q.Group}
		}
	}
	return r.repo.Search(ctx, options)
}
