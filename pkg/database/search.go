package database

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) Search(ctx context.Context, query *models.SearchQuery) ([]*models.SearchResultRow, error) {
	services := []string{}
	for _, s := range query.Services {
		services = append(services, string(s))
	}

	types := []string{}
	for _, s := range query.Types {
		types = append(types, string(s))
		if strings.ToLower(string(s)) == "story" {
			types = append(types, "story")
		}
	}

	res, err := r.querier.Search(ctx, gensql.SearchParams{
		Query:   ptrToString(query.Text),
		Keyword: query.Keywords,
		Grp:     query.Groups,
		TeamID:  query.TeamIDs,
		Service: services,
		Types:   types,
		Lim:     int32(ptrToIntDefault(query.Limit, 24)),
		Offs:    int32(ptrToIntDefault(query.Offset, 0)),
	})
	if err != nil {
		return nil, err
	}

	order := map[string]int{}
	var dataproducts []uuid.UUID
	var stories []uuid.UUID
	var datasets []uuid.UUID
	excerpts := map[uuid.UUID]string{}
	for i, sr := range res {
		switch sr.ElementType {
		case "dataproduct":
			dataproducts = append(dataproducts, sr.ElementID)
		case "story":
			stories = append(stories, sr.ElementID)
		case "dataset":
			datasets = append(datasets, sr.ElementID)
		default:
			r.log.Error("unknown search result type", sr.ElementType)
			continue
		}
		order[sr.ElementType+sr.ElementID.String()] = i
		excerpts[sr.ElementID] = sr.Excerpt
	}

	dps, err := r.querier.GetDataproductsByIDs(ctx, dataproducts)
	if err != nil {
		return nil, err
	}

	ss, err := r.querier.GetStoriesByIDs(ctx, stories)
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

	for _, s := range ss {
		ret = append(ret, &models.SearchResultRow{
			Excerpt: excerpts[s.ID],
			Result:  storySQLToGraphql(&s),
		})
	}

	sortSearch(ret, order)

	return ret, nil
}

func sortSearch(ret []*models.SearchResultRow, order map[string]int) {
	getRank := func(m models.SearchResult) int {
		switch m := m.(type) {
		case *models.Dataproduct:
			return order["dataproduct"+m.ID.String()]
		case *models.Dataset:
			return order["dataset"+m.ID.String()]
		default:
			return -1
		}
	}

	getCreatedAt := func(m models.SearchResult) time.Time {
		switch m := m.(type) {
		case *models.Dataproduct:
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

func ptrToIntDefault(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}
