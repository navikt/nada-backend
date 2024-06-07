package service

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
)

type Team struct {
	teamkatalogen.Team
	DataproductsNumber    int `json:"dataproductsNumber"`
	StoriesNumber         int `json:"storiesNumber"`
	InsightProductsNumber int `json:"insightProductsNumber"`
}

type ProductAreasDto struct {
	ProductAreas []ProductArea `json:"productAreas"`
}

type ProductArea struct {
	teamkatalogen.ProductArea
	Teams        []Team `json:"teams"`
	DashboardURL string `json:"dashboardURL"`
}

type TeamWithAssets struct {
	teamkatalogen.Team
	Dataproducts    []Dataproduct    `json:"dataproducts"`
	Stories         []Story          `json:"stories"`
	InsightProducts []InsightProduct `json:"insightProducts"`
	DashboardURL    string           `json:"dashboardURL"`
}

type ProductAreaWithAssets struct {
	ProductArea
	Teams []TeamWithAssets `json:"teams"`
}

func GetProductAreas(ctx context.Context) (*ProductAreasDto, *APIError) {
	pa, err := tkClient.GetProductAreas(ctx)
	if err != nil {
		return nil, NewInternalError(err, "Failed to get product areas from Team Katalogen")
	}

	productAreas := make([]ProductArea, 0)
	for _, p := range pa {
		dash, err := queries.GetDashboard(ctx, p.ID)
		if err != nil && err != sql.ErrNoRows {
			return nil, DBErrorToAPIError(err, "GetProductAreas(): failed to get dashboard url")
		}

		teams := make([]Team, 0)
		for _, tkTeam := range p.Teams {
			dataproductsNumber, err := queries.GetDataproductsNumberByTeam(ctx, ptrToNullString(&tkTeam.ID))
			if err != nil {
				return nil, DBErrorToAPIError(err, "GetProductAreas(): failed to get dataproducts number")
			}

			storiesNumber, err := queries.GetStoriesNumberByTeam(ctx, ptrToNullString(&tkTeam.ID))
			if err != nil {
				return nil, DBErrorToAPIError(err, "GetProductAreas(): failed to get stories number")
			}

			insightProductsNumber, err := queries.GetInsightProductsNumberByTeam(ctx, ptrToNullString(&tkTeam.ID))
			if err != nil {
				return nil, DBErrorToAPIError(err, "GetProductAreas(): failed to get insight products number")
			}

			teams = append(teams, Team{
				Team:                  tkTeam,
				DataproductsNumber:    int(dataproductsNumber),
				StoriesNumber:         int(storiesNumber),
				InsightProductsNumber: int(insightProductsNumber),
			})
		}
		productAreas = append(productAreas, ProductArea{
			ProductArea:  *p,
			Teams:        teams,
			DashboardURL: dash.Url,
		})
	}
	return &ProductAreasDto{
		ProductAreas: productAreas,
	}, nil
}

func GetProductAreaWithAssets(ctx context.Context, id string) (*ProductAreaWithAssets, *APIError) {
	tkProductArea, err := tkClient.GetProductArea(ctx, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, DBErrorToAPIError(err, "GetProductAreaWithAssets(): failed to get product area")
	} else if err == sql.ErrNoRows {
		return nil, NewAPIError(http.StatusNotFound, err, "GetProductAreaWithAssets(): product area not found")
	}

	dash, err := queries.GetDashboard(ctx, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, DBErrorToAPIError(err, "GetProductAreaWithAssets(): failed to get dashboard url")
	}
	productArea := &ProductAreaWithAssets{
		ProductArea: ProductArea{
			ProductArea:  *tkProductArea,
			DashboardURL: dash.Url,
		},
		Teams: make([]TeamWithAssets, 0),
	}

	teamIDs := make([]string, len(tkProductArea.Teams))
	for idx, tkTeam := range tkProductArea.Teams {
		productArea.Teams = append(productArea.Teams, TeamWithAssets{
			Team:            tkTeam,
			Dataproducts:    []Dataproduct{},
			Stories:         []Story{},
			InsightProducts: []InsightProduct{},
		})
		teamIDs[idx] = tkTeam.ID

		teamDash, err := queries.GetDashboard(ctx, teamIDs[idx])
		if err != nil && err != sql.ErrNoRows {
			return nil, DBErrorToAPIError(err, "GetProductAreaWithAssets(): failed to get team dashboard url")
		}
		productArea.Teams[idx].DashboardURL = teamDash.Url
	}

	dataproducts, apiErr := getDataproductsByTeamID(ctx, teamIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	for _, dp := range dataproducts {
		for idx, team := range productArea.Teams {
			if dp.Owner.TeamID != nil && team.ID == *dp.Owner.TeamID {
				productArea.Teams[idx].Dataproducts = append(productArea.Teams[idx].Dataproducts, *dp)
			}
		}
	}

	stories, apiErr := getStoriesByTeamID(ctx, teamIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	for _, s := range stories {
		for idx, team := range productArea.Teams {
			if s.TeamID != nil && team.ID == *s.TeamID {
				productArea.Teams[idx].Stories = append(productArea.Teams[idx].Stories, *s)
			}
		}
	}

	insightProducts, apiErr := getInsightProductsByTeamID(ctx, teamIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	for _, ip := range insightProducts {
		for idx, team := range productArea.Teams {
			if ip.TeamID != nil && team.ID == *ip.TeamID {
				productArea.Teams[idx].InsightProducts = append(productArea.Teams[idx].InsightProducts, *ip)
			}
		}
	}

	return productArea, nil
}

func dataproductFromSQL(dp *gensql.DataproductWithTeamkatalogenView) *Dataproduct {
	return &Dataproduct{
		ID:          dp.ID,
		Name:        dp.Name,
		Description: &dp.Description.String,
		Owner: &DataproductOwner{
			Group:            dp.Group,
			TeamkatalogenURL: nullStringToPtr(dp.TeamkatalogenUrl),
			TeamContact:      nullStringToPtr(dp.TeamContact),
			TeamID:           nullStringToPtr(dp.TeamID),
			ProductAreaID:    nullUUIDToUUIDPtr(dp.PaID),
		},
		Created:         dp.Created,
		LastModified:    dp.LastModified,
		Slug:            dp.Slug,
		TeamName:        nullStringToPtr(dp.TeamName),
		ProductAreaName: nullStringToString(dp.PaName),
	}
}

func getDataproductsByTeamID(ctx context.Context, teamIDs []string) ([]*Dataproduct, *APIError) {
	sqlDP, err := queries.GetDataproductsByProductArea(ctx, teamIDs)
	if err == sql.ErrNoRows {
		return []*Dataproduct{}, nil
	}
	if err != nil {
		return nil, DBErrorToAPIError(err, "getDataproductsByTeamID(): failed to get dataproducts")
	}

	dps := make([]*Dataproduct, len(sqlDP))
	for idx, dp := range sqlDP {
		dps[idx] = dataproductFromSQL(&dp)
		keywords, err := queries.GetDataproductKeywords(ctx, dps[idx].ID)
		if err != nil && err != sql.ErrNoRows {
			return nil, DBErrorToAPIError(err, "getDataproductsByTeamID(): failed to get keywords")
		}
		if keywords == nil {
			keywords = []string{}
		}
		dps[idx].Keywords = keywords
	}

	return dps, nil
}

func getStoriesByTeamID(ctx context.Context, teamIDs []string) ([]*Story, *APIError) {
	sqlStories, err := queries.GetStoriesByProductArea(ctx, teamIDs)
	if err == sql.ErrNoRows {
		return []*Story{}, nil
	}
	if err != nil {
		return nil, DBErrorToAPIError(err, "getStoriesByTeamID(): failed to get stories")
	}

	stories := make([]*Story, len(sqlStories))
	for idx, s := range sqlStories {
		stories[idx] = storyFromSQL(&s)
	}

	return stories, nil
}

func getInsightProductsByTeamID(ctx context.Context, teamIDs []string) ([]*InsightProduct, *APIError) {
	sqlInsightProducts, err := queries.GetInsightProductsByProductArea(ctx, teamIDs)
	if err == sql.ErrNoRows {
		return []*InsightProduct{}, nil
	}
	if err != nil {
		return nil, DBErrorToAPIError(err, "getInsightProductsByTeamID(): failed to get insight products")
	}

	insightProducts := make([]*InsightProduct, len(sqlInsightProducts))
	for idx, ip := range sqlInsightProducts {
		insightProducts[idx] = insightProductFromSQL(&ip)
	}

	return insightProducts, nil
}
