package core

import (
	"context"
	"fmt"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.ProductAreaService = &productAreaService{}

type productAreaService struct {
	productAreaStorage    service.ProductAreaStorage
	dataProductStorage    service.DataProductsStorage
	insightProductStorage service.InsightProductStorage
	storyStorage          service.StoryStorage
}

func (s *productAreaService) GetProductAreaWithAssets(ctx context.Context, id string) (*service.ProductAreaWithAssets, error) {
	rawProductArea, err := s.productAreaStorage.GetProductArea(ctx, id)
	if err != nil {
		return nil, err
	}

	dash, err := s.productAreaStorage.GetDashboard(ctx, id)
	if err != nil {
		return nil, err
	}

	productArea := &service.ProductAreaWithAssets{
		ProductArea: &service.ProductArea{
			TeamkatalogenProductArea: rawProductArea.TeamkatalogenProductArea,
			DashboardURL:             dash.Url,
		},
		Teams: make([]*service.TeamWithAssets, 0),
	}

	teamIDs := make([]string, len(rawProductArea.Teams))
	for idx, tkTeam := range rawProductArea.Teams {
		productArea.Teams = append(productArea.Teams, &service.TeamWithAssets{
			TeamkatalogenTeam: tkTeam.TeamkatalogenTeam,
			Dataproducts:      []*service.Dataproduct{},
			Stories:           []*service.Story{},
			InsightProducts:   []*service.InsightProduct{},
		})
		teamIDs[idx] = tkTeam.ID

		teamDash, err := s.productAreaStorage.GetDashboard(ctx, teamIDs[idx])
		if err != nil {
			return nil, err
		}
		productArea.Teams[idx].DashboardURL = teamDash.Url
	}

	dataproducts, apiErr := s.dataProductStorage.GetDataproductsByTeamID(ctx, teamIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	for _, dp := range dataproducts {
		for idx, team := range productArea.Teams {
			if dp.Owner.TeamID != nil && team.ID == *dp.Owner.TeamID {
				productArea.Teams[idx].Dataproducts = append(productArea.Teams[idx].Dataproducts, dp)
			}
		}
	}

	stories, apiErr := s.storyStorage.GetStoriesByTeamID(ctx, teamIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	for _, s := range stories {
		for idx, team := range productArea.Teams {
			if s.TeamID != nil && team.ID == *s.TeamID {
				productArea.Teams[idx].Stories = append(productArea.Teams[idx].Stories, s)
			}
		}
	}

	insightProducts, apiErr := s.insightProductStorage.GetInsightProductsByTeamID(ctx, teamIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	for _, ip := range insightProducts {
		for idx, team := range productArea.Teams {
			if ip.TeamID != nil && team.ID == *ip.TeamID {
				productArea.Teams[idx].InsightProducts = append(productArea.Teams[idx].InsightProducts, ip)
			}
		}
	}

	return productArea, nil
}

func (s *productAreaService) GetProductAreas(ctx context.Context) (*service.ProductAreasDto, error) {
	pa, err := s.productAreaStorage.GetProductAreas(ctx)
	if err != nil {
		return nil, fmt.Errorf("get product areas: %w", err)
	}

	for _, p := range pa {
		dash, err := s.productAreaStorage.GetDashboard(ctx, p.ID)
		if err != nil {
			return nil, err
		}

		p.DashboardURL = dash.Url

		for _, team := range p.Teams {
			numDataProducts, err := s.dataProductStorage.GetDataproductsNumberByTeam(ctx, team.ID)
			if err != nil {
				return nil, err
			}

			numStories, err := s.storyStorage.GetStoriesNumberByTeam(ctx, team.ID)
			if err != nil {
				return nil, err
			}

			numInsightProducts, err := s.insightProductStorage.GetInsightProductsNumberByTeam(ctx, team.ID)
			if err != nil {
				return nil, err
			}

			team.DataproductsNumber = int(numDataProducts)
			team.StoriesNumber = int(numStories)
			team.InsightProductsNumber = int(numInsightProducts)
		}
	}

	return &service.ProductAreasDto{
		ProductAreas: pa,
	}, nil
}

func NewProductAreaService(
	productAreaStorage service.ProductAreaStorage,
	dataProductStorage service.DataProductsStorage,
	insightProductStorage service.InsightProductStorage,
	storyStorage service.StoryStorage,
) *productAreaService {
	return &productAreaService{
		productAreaStorage:    productAreaStorage,
		dataProductStorage:    dataProductStorage,
		insightProductStorage: insightProductStorage,
		storyStorage:          storyStorage,
	}
}
