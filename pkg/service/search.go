package service

import (
	"context"

	"github.com/google/uuid"
)

type SearchStorage interface {
	Search(ctx context.Context, query *SearchOptions) ([]*SearchResultRaw, error)
}

type SearchService interface {
	Search(ctx context.Context, query *SearchOptions) (*SearchResult, error)
}

func (DataproductWithDataset) IsSearchResult() {}

func (Dataset) IsSearchResult() {}

func (Story) IsSearchResult() {}

type ResultItem interface {
	IsSearchResult()
}

type SearchResult struct {
	Results []*SearchResultRow `json:"results"`
}

type SearchResultRaw struct {
	ElementID   uuid.UUID
	ElementType string
	Rank        float32
	Excerpt     string
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
	// Filter on team_id
	TeamIDs []uuid.UUID `json:"teamIDs"`
	// Filter on enabled services
	Services []string `json:"services"`
	// Filter on types
	Types []string `json:"types"`

	Limit  *int `json:"limit"`
	Offset *int `json:"offset"`
}
