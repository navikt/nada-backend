package models

type TeamkatalogenResult struct {
	TeamID        string `json:"teamID"`
	URL           string `json:"url"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ProductAreaID string `json:"productAreaID"`
}
