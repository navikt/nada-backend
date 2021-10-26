package models

type SearchResult interface {
	IsSearchResult()
}

type SearchQuery struct {
	// Freetext search
	Text *string `json:"text"`
	// Filter on keyword
	Keyword *string `json:"keyword"`
	Limit   *int    `json:"limit"`
	Offset  *int    `json:"offset"`
}
