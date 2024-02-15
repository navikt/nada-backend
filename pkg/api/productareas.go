package api

import (
	"context"
	"database/sql"

	"github.com/navikt/nada-backend/pkg/teamkatalogen"
)

type TeamDto struct {
	teamkatalogen.Team
	DataproductsNumber    int `json:"dataproductsNumber"`
	StoriesNumber         int `json:"storiesNumber"`
	InsightProductsNumber int `json:"insightProductsNumber"`
}

type ProductAreaDto struct {
	teamkatalogen.ProductArea
	Teams        []*TeamDto `json:"teams"`
	DashboardURL string     `json:"dashboardURL"`
}

func GetProductAreas(ctx context.Context) ([]*ProductAreaDto, *APIError) {
	pa, err := tkClient.GetProductAreas(ctx)
	if err != nil {
		return nil, NewInternalError(err, "Failed to get product areas from Team Katalogen")
	}

	productAreas := make([]*ProductAreaDto, 0)
	for _, p := range pa {
		dash, err := querier.GetDashboard(ctx, p.ID)
		if err != nil && err != sql.ErrNoRows {
			return nil, DBErrorToAPIError(err, "GetProductAreas(): failed to get dashboard url")
		}

		tkTeams, err := tkClient.GetTeamsInProductArea(ctx, p.ID)
		if err != nil {
			return nil, NewInternalError(err, "Failed to get teams in product area from Team Katalogen")
		}
		teams := make([]*TeamDto, 0)
		for _, tkTeam := range tkTeams {
			dataproductsNumber, err := querier.GetDataproductsNumberByTeam(ctx, ptrToNullString(&tkTeam.ID))
			if err != nil {
				return nil, DBErrorToAPIError(err, "GetProductAreas(): failed to get dataproducts number")
			}

			storiesNumber, err := querier.GetStoriesNumberByTeam(ctx, ptrToNullString(&tkTeam.ID))
			if err != nil {
				return nil, DBErrorToAPIError(err, "GetProductAreas(): failed to get stories number")
			}

			insightProductsNumber, err := querier.GetInsightProductsNumberByTeam(ctx, ptrToNullString(&tkTeam.ID))
			if err != nil {
				return nil, DBErrorToAPIError(err, "GetProductAreas(): failed to get insight products number")
			}
			teams = append(teams, &TeamDto{
				Team:                  *tkTeam,
				DataproductsNumber:    int(dataproductsNumber),
				StoriesNumber:         int(storiesNumber),
				InsightProductsNumber: int(insightProductsNumber),
			})
		}
		productAreas = append(productAreas, &ProductAreaDto{
			ProductArea:  *p,
			Teams:        teams,
			DashboardURL: dash.Url,
		})
	}
	return productAreas, nil
}
