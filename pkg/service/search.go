package service

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

func (DataproductWithDataset) IsSearchResult() {}

func (Dataset) IsSearchResult() {}

func (Story) IsSearchResult() {}

type ResultItem interface {
	IsSearchResult()
}

type SearchResult struct {
	Results []*SearchResultRow `json:"results"`
}

type SearchResultRow struct {
	Excerpt string     `json:"excerpt"`
	Result  ResultItem `json:"result"`
	Rank    float64    `json:"rank"`
}

func Search(ctx context.Context, // Freetext search
	text string,
	// Filter on keyword
	keywords []string,
	// Filter on group
	groups []string,
	//Filter on team_id
	teamIDs []string,
	// Filter on enabled services
	services []string,
	// Filter on types
	types []string,
	limit int,
	offset int) (*SearchResult, *APIError) {
	if limit == 0 {
		limit = 24
	}
	res, err := queries.Search(ctx, gensql.SearchParams{
		Query:   text,
		Keyword: keywords,
		Grp:     groups,
		TeamID:  teamIDs,
		Service: services,
		Types:   types,
		Lim:     int32(limit),
		Offs:    int32(offset),
	})

	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to search")
	}

	order := map[string]int{}
	var dataproducts []uuid.UUID
	var stories []uuid.UUID
	excerpts := map[uuid.UUID]string{}
	for i, sr := range res {
		switch sr.ElementType {
		case "dataproduct":
			dataproducts = append(dataproducts, sr.ElementID)
		case "story":
			stories = append(stories, sr.ElementID)
		default:
			log.Error("unknown search result type", sr.ElementType)
			continue
		}
		order[sr.ElementType+sr.ElementID.String()] = i
		excerpts[sr.ElementID] = sr.Excerpt
	}

	dps, apierr := GetDataproducts(ctx, dataproducts)
	if apierr != nil {
		return nil, apierr
	}

	ss, err := queries.GetStoriesWithTeamkatalogenByIDs(ctx, stories)
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to get stories by IDs")
	}

	ret := []*SearchResultRow{}
	for _, d := range dps {
		ret = append(ret, &SearchResultRow{
			Excerpt: excerpts[d.ID],
			Result:  d,
		})
	}

	for _, s := range ss {
		ret = append(ret, &SearchResultRow{
			Excerpt: excerpts[s.ID],
			Result:  storyFromSQL(&s),
		})
	}

	sortSearch(ret, order)

	return &SearchResult{
		Results: ret,
	}, nil
}

func sortSearch(ret []*SearchResultRow, order map[string]int) {
	getRank := func(m ResultItem) int {
		switch m := m.(type) {
		case *DataproductWithDataset:
			return order["dataproduct"+m.ID.String()]
		case *Dataset:
			return order["dataset"+m.ID.String()]
		default:
			return -1
		}
	}

	getCreatedAt := func(m ResultItem) time.Time {
		switch m := m.(type) {
		case *DataproductWithDataset:
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
