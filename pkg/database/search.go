package database

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) Search(ctx context.Context, query *models.SearchQuery) ([]models.SearchResult, error) {
	res, err := r.querier.Search(ctx, gensql.SearchParams{
		Query:   ptrToString(query.Text),
		Keyword: ptrToString(query.Keyword),
	})
	if err != nil {
		return nil, err
	}
	ranks := map[string]float32{}
	var dataproducts []uuid.UUID
	var collections []uuid.UUID
	for _, sr := range res {
		switch sr.ElementType {
		case "dataproduct":
			dataproducts = append(dataproducts, sr.ElementID)
		case "collection":
			collections = append(collections, sr.ElementID)
		}
		ranks[sr.ElementType+sr.ElementID.String()] = sr.TsRankCd
	}

	dps, err := r.querier.GetDataproductsByIDs(ctx, dataproducts)
	if err != nil {
		return nil, err
	}
	cols, err := r.querier.GetCollectionsByIDs(ctx, collections)
	if err != nil {
		return nil, err
	}

	ret := []models.SearchResult{}
	for _, d := range dps {
		ret = append(ret, dataproductFromSQL(d))
	}
	for _, c := range cols {
		ret = append(ret, collectionFromSQL(c))
	}

	getRank := func(m models.SearchResult) float32 {
		switch m := m.(type) {
		case *models.Dataproduct:
			return ranks["dataproduct"+m.ID.String()]
		case *models.Collection:
			return ranks["collection"+m.ID.String()]
		default:
			return -1
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return getRank(ret[i]) < getRank(ret[j])
	})

	return ret, nil
}
