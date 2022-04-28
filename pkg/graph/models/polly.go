package models

import "github.com/google/uuid"

type DatabasePolly struct {
	ID uuid.UUID `json:"id"`
	Polly
}

type Polly struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}
