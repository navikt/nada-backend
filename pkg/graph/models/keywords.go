package models

type Keyword struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

type UpdateKeywords struct {
	ObsoleteKeywords []string `json:"obsoleteKeywords"`
	ReplacedKeywords []string `json:"replacedKeywords"`
	NewText          []string `json:"newText"`
}
