package models

import "github.com/google/uuid"

type Polly struct {
	ID uuid.UUID `json:"id"`
	QueryPolly
}

type PollyInput struct {
	ID *uuid.UUID `json:"id"`
	QueryPolly
}

type QueryPolly struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}
