package service

import (
	"context"
)

type ProductAreaStorage interface {
	GetProductArea(ctx context.Context, paID string) (*ProductArea, error)
	GetProductAreas(ctx context.Context) ([]*ProductArea, error)
	GetDashboard(ctx context.Context, id string) (*Dashboard, error)
}

type ProductAreaService interface {
	GetProductAreas(ctx context.Context) (*ProductAreasDto, error)
	GetProductAreaWithAssets(ctx context.Context, id string) (*ProductAreaWithAssets, error)
}

type Team struct {
	*TeamkatalogenTeam
	DataproductsNumber    int `json:"dataproductsNumber"`
	StoriesNumber         int `json:"storiesNumber"`
	InsightProductsNumber int `json:"insightProductsNumber"`
}

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
	ID  string
	Url string
}

type ProductAreaWithAssets struct {
	*ProductArea
	Teams []*TeamWithAssets `json:"teams"`
}
