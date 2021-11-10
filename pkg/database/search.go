package database

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) Search(ctx context.Context, query *models.SearchQuery) ([]*models.SearchResultRow, error) {
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
	excerpts := map[uuid.UUID]string{}
	for _, sr := range res {
		switch sr.ElementType {
		case "dataproduct":
			dataproducts = append(dataproducts, sr.ElementID)
		case "collection":
			collections = append(collections, sr.ElementID)
		}
		ranks[sr.ElementType+sr.ElementID.String()] = sr.Rank
		excerpts[sr.ElementID] = sr.Excerpt
	}

	dps, err := r.querier.GetDataproductsByIDs(ctx, dataproducts)
	if err != nil {
		return nil, err
	}
	cols, err := r.querier.GetCollectionsByIDs(ctx, collections)
	if err != nil {
		return nil, err
	}

	ret := []*models.SearchResultRow{}
	for _, d := range dps {
		ret = append(ret, &models.SearchResultRow{
			Excerpt: excerpts[d.ID],
			Result:  dataproductFromSQL(d),
		})
	}
	for _, c := range cols {
		ret = append(ret, &models.SearchResultRow{
			Excerpt: excerpts[c.ID],
			Result:  collectionFromSQL(c),
		})
	}
	sortSearch(ret, ranks)

	return ret, nil
}

func sortSearch(ret []*models.SearchResultRow, ranks map[string]float32) {
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

	getCreatedAt := func(m models.SearchResult) time.Time {
		switch m := m.(type) {
		case *models.Dataproduct:
			return m.Created
		case *models.Collection:
			return m.Created
		default:
			return time.Time{}
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		ri, rj := getRank(ret[i].Result), getRank(ret[j].Result)
		if ri != rj {
			return ri > rj
		}

		return getCreatedAt(ret[i].Result).After(getCreatedAt(ret[j].Result))
	})
}