package models

type SearchResult interface {
	IsSearchResult()
}

type SearchQuery struct {
	// Freetext search
	Text *string `json:"text"`
	// Filter on keyword
	Keyword *string `json:"keyword"`
	// Filter on group
	Group  *string `json:"group"`
	Limit  *int    `json:"limit"`
	Offset *int    `json:"offset"`
}

type SearchResultRow struct {
	Excerpt string       `json:"excerpt"`
	Result  SearchResult `json:"result"`
	Rank    float64      `json:"rank"`
}
