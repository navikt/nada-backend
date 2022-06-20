package models

import (
	"fmt"
	"io"
	"strconv"
)

type SearchResult interface {
	IsSearchResult()
}

type SearchQueryOld struct {
	// Freetext search
	Text *string `json:"text"`
	// Filter on keyword
	Keyword *string `json:"keyword"`
	// Filter on group
	Group  *string `json:"group"`
	Limit  *int    `json:"limit"`
	Offset *int    `json:"offset"`
}

type SearchQuery struct {
	// Freetext search
	Text *string `json:"text"`
	// Filter on keyword
	Keywords []string `json:"keywords"`
	// Filter on group
	Groups []string `json:"groups"`
	// Filter on enabled services
	Services []MappingService `json:"services"`
	// Filter on types
	Types []SearchType `json:"types"`

	Limit  *int `json:"limit"`
	Offset *int `json:"offset"`
}

type SimpleSearchQuery struct {
	// Use OR as a keyword for the OR operator. Example "night OR day"
	Text *string `json:"text"`
	// keyword filters results on the keyword.
	Keyword *string `json:"keyword"`
	// group filters results on the group.
	Group *string `json:"group"`
	// limit the number of returned search results.
	Limit *int `json:"limit"`
	// offset the list of returned search results. Used as pagination with PAGE-INDEX * limit.
	Offset *int `json:"offset"`
}

type SearchResultRow struct {
	Excerpt string       `json:"excerpt"`
	Result  SearchResult `json:"result"`
	Rank    float64      `json:"rank"`
}

type SearchType string

const (
	SearchTypeDataproduct SearchType = "dataproduct"
	SearchTypeStory       SearchType = "story"
)

var AllSearchType = []SearchType{
	SearchTypeDataproduct,
	SearchTypeStory,
}

func (e SearchType) IsValid() bool {
	switch e {
	case SearchTypeDataproduct:
		return true
	case SearchTypeStory:
		return true
	}
	return false
}

func (e SearchType) String() string {
	return string(e)
}

func (e *SearchType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = SearchType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid SearchType", str)
	}
	return nil
}

func (e SearchType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
