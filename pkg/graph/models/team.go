package models

import "github.com/google/uuid"

// NadaToken contains the token of the corresponding team
type NadaToken struct {
	// name of team
	Team string `json:"team"`
	// nada token for the team
	Token uuid.UUID `json:"token"`
}
