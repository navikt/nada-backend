package service

import (
	"context"

	"github.com/google/uuid"
)

type TeamKatalogenAPI interface {
	GetTeam(ctx context.Context, teamID uuid.UUID) (*TeamkatalogenTeam, error)
	GetTeamCatalogURL(teamID uuid.UUID) string
	GetTeamsInProductArea(ctx context.Context, paID uuid.UUID) ([]*TeamkatalogenTeam, error)
	GetProductAreas(ctx context.Context) ([]*TeamkatalogenProductArea, error)
	Search(ctx context.Context, gcpGroups []string) ([]TeamkatalogenResult, error)
}

type TeamKatalogenService interface {
	SearchTeamKatalogen(ctx context.Context, gcpGroups []string) ([]TeamkatalogenResult, error)
}

type TeamkatalogenResult struct {
	TeamID        string `json:"teamID"`
	URL           string `json:"url"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ProductAreaID string `json:"productAreaID"`
}

type TeamkatalogenProductArea struct {
	// id is the id of the product area.
	ID uuid.UUID `json:"id"`
	// name is the name of the product area.
	Name string `json:"name"`
	// areaType is the type of the product area.
	AreaType string `json:"areaType"`

	// FIXME: Can probably get rid of this
	Teams []Team `json:"teams"`
}

type TeamkatalogenTeam struct {
	// id is the team external id in teamkatalogen.
	ID uuid.UUID `json:"id"`
	// name is the name of the team.
	Name string `json:"name"`
	// productAreaID is the id of the product area.
	ProductAreaID uuid.UUID `json:"productAreaID"`
}
