package service

import (
	"context"

	"github.com/google/uuid"
)

type ProductAreaStorage interface {
	GetProductArea(ctx context.Context, paID uuid.UUID) (*ProductArea, error)
	GetProductAreas(ctx context.Context) ([]*ProductArea, error)
	GetDashboard(ctx context.Context, id uuid.UUID) (*Dashboard, error)
	UpsertProductAreaAndTeam(ctx context.Context, pa []*UpsertProductAreaRequest, t []*UpsertTeamRequest) error
}

type ProductAreaService interface {
	GetProductAreas(ctx context.Context) (*ProductAreasDto, error)
	GetProductAreaWithAssets(ctx context.Context, id uuid.UUID) (*ProductAreaWithAssets, error)
}

type UpsertProductAreaRequest struct {
	ID   uuid.UUID
	Name string
}

// FIXCME: does this belong here?
type UpsertTeamRequest struct {
	ID            uuid.UUID
	ProductAreaID uuid.UUID
	Name          string
}

type Team struct {
	*TeamkatalogenTeam
	DataproductsNumber    int `json:"dataproductsNumber"`
	StoriesNumber         int `json:"storiesNumber"`
	InsightProductsNumber int `json:"insightProductsNumber"`
}

// FIXME: we need to simplify these structs, there is too much duplication
type ProductAreasDto struct {
	ProductAreas []*ProductArea `json:"productAreas"`
}

type ProductArea struct {
	*TeamkatalogenProductArea
	Teams        []*Team `json:"teams"`
	DashboardURL string  `json:"dashboardURL"`
}

type TeamWithAssets struct {
	*TeamkatalogenTeam
	Dataproducts    []*Dataproduct    `json:"dataproducts"`
	Stories         []*Story          `json:"stories"`
	InsightProducts []*InsightProduct `json:"insightProducts"`
	DashboardURL    string            `json:"dashboardURL"`
}

type Dashboard struct {
	ID  uuid.UUID
	Url string
}

type ProductAreaWithAssets struct {
	*ProductArea
	Teams []*TeamWithAssets `json:"teams"`
}
