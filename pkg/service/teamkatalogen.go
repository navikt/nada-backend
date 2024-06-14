package service

import (
	"context"
)

type TeamKatalogenAPI interface {
	GetTeam(ctx context.Context, teamID string) (*TeamkatalogenTeam, error)
	GetTeamCatalogURL(teamID string) string
	GetTeamsInProductArea(ctx context.Context, paID string) ([]*TeamkatalogenTeam, error)
	GetProductArea(ctx context.Context, paID string) (*TeamkatalogenProductArea, error)
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
	ID string `json:"id"`
	// name is the name of the product area.
	Name string `json:"name"`
	// areaType is the type of the product area.
	AreaType string `json:"areaType"`

	// FIXME: Can probably get rid of this
	Teams []Team `json:"teams"`
}

type TeamkatalogenTeam struct {
	// id is the team external id in teamkatalogen.
	ID string `json:"id"`
	// name is the name of the team.
	Name string `json:"name"`
	// productAreaID is the id of the product area.
	ProductAreaID string `json:"productAreaID"`
}
