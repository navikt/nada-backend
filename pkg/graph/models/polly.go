package models

import "github.com/google/uuid"

type PollyResult struct {
	// purpose id from polly
	ID uuid.UUID `json:"id"`
	// purpose name from polly
	Name string `json:"name"`
}
