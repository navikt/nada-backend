package database

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) KeywordsSortedByPopularity(ctx context.Context) ([]*models.Keyword, error) {
	ks, err := r.querier.GetKeywords(ctx)
	if err != nil {
		return nil, err
	}

	km := make([]*models.Keyword, len(ks))
	for i, k := range ks {
		km[i] = &models.Keyword{
			Keyword: k.Keyword,
			Count:   int(k.Count),
		}
	}

	return km, nil
}
