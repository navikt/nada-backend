package api

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
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

type SearchOptions struct {
	// Freetext search
	Text string `json:"text"`
	// Filter on keyword
	Keywords []string `json:"keywords"`
	// Filter on group
	Groups []string `json:"groups"`
	//Filter on team_id
	TeamIDs []string `json:"teamIDs"`
	// Filter on enabled services
	Services []string `json:"services"`
	// Filter on types
	Types []string `json:"types"`

	Limit  *int `json:"limit"`
	Offset *int `json:"offset"`
}

func Search(ctx context.Context, query *SearchOptions) (*SearchResult, *APIError) {
	res, err := queries.Search(ctx, gensql.SearchParams{
		Query:   query.Text,
		Keyword: query.Keywords,
		Grp:     query.Groups,
		TeamID:  query.TeamIDs,
		Service: query.Services,
		Types:   query.Types,
		Lim:     int32(ptrToIntDefault(query.Limit, 24)),
		Offs:    int32(ptrToIntDefault(query.Offset, 0)),
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

	dps, apierr := getDataproducts(ctx, dataproducts)
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
