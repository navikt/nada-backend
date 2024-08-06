package core

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.SearchService = &searchService{}

type searchService struct {
	searchStorage       service.SearchStorage
	storyStorage        service.StoryStorage
	dataProductsStorage service.DataProductsStorage
}

func (s *searchService) Search(ctx context.Context, query *service.SearchOptions) (*service.SearchResult, error) {
	const op errs.Op = "searchService.Search"

	res, err := s.searchStorage.Search(ctx, query)
	if err != nil {
		return nil, errs.E(op, err)
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
			continue
		}
		order[sr.ElementType+sr.ElementID.String()] = i
		excerpts[sr.ElementID] = sr.Excerpt
	}

	dps, err := s.dataProductsStorage.GetDataproducts(ctx, dataproducts)
	if err != nil {
		return nil, errs.E(op, err)
	}

	ss, err := s.storyStorage.GetStoriesWithTeamkatalogenByIDs(ctx, stories)
	if err != nil {
		return nil, errs.E(op, err)
	}

	ret := []*service.SearchResultRow{}
	for _, d := range dps {
		ret = append(ret, &service.SearchResultRow{
			Excerpt: excerpts[d.ID],
			Result:  d,
		})
	}

	for _, s := range ss {
		ret = append(ret, &service.SearchResultRow{
			Excerpt: excerpts[s.ID],
			Result:  s,
		})
	}

	sortSearch(ret, order)

	return &service.SearchResult{
		Results: ret,
	}, nil
}

func sortSearch(ret []*service.SearchResultRow, order map[string]int) {
	getRank := func(m service.ResultItem) int {
		switch m := m.(type) {
		case *service.DataproductWithDataset:
			return order["dataproduct"+m.ID.String()]
		case *service.Dataset:
			return order["dataset"+m.ID.String()]
		default:
			return -1
		}
	}

	getCreatedAt := func(m service.ResultItem) time.Time {
		switch m := m.(type) {
		case *service.DataproductWithDataset:
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

func NewSearchService(
	searchStorage service.SearchStorage,
	storyStorage service.StoryStorage,
	dataProductsStorage service.DataProductsStorage,
) *searchService {
	return &searchService{
		searchStorage:       searchStorage,
		storyStorage:        storyStorage,
		dataProductsStorage: dataProductsStorage,
	}
}
