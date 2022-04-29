package models

import "github.com/google/uuid"

type Polly struct {
	ID uuid.UUID `json:"id"`
	NewPolly
}

type NewPolly struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}
