package models

import "github.com/google/uuid"

type TableColumn struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	Type        string `json:"type"`
}

type TableMetadata struct {
	ID     uuid.UUID     `json:"id"`
	Schema []TableColumn `json:"schema"`
}
