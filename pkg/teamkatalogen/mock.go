package teamkatalogen

import (
	"context"
	"strings"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

type Mock struct {
	Teams []*models.TeamkatalogenResult
}

func NewMock() *Mock {
	tk := &Mock{}
	for _, t := range auth.MockUser.GoogleGroups {
		tk.Teams = append(tk.Teams, &models.TeamkatalogenResult{
			Name:          t.Name,
			URL:           "https://some.url",
			Description:   "This is a description of " + t.Name,
			ProductAreaID: "Mocked-001",
			TeamID:        t.Name + "-001",
		})
	}
	return tk
}

func (m *Mock) Search(ctx context.Context, query string) ([]*models.TeamkatalogenResult, error) {
	ret := []*models.TeamkatalogenResult{}
	for _, t := range m.Teams {
		if strings.Contains(strings.ToLower(t.Name), strings.ToLower(query)) {
			ret = append(ret, t)
		}
	}
	return ret, nil
}

func (m *Mock) GetTeamsInProductArea(ctx context.Context, paID string) ([]*models.Team, error) {
	teams := make([]*models.Team, len(auth.MockUser.GoogleGroups))
	for idx, t := range auth.MockUser.GoogleGroups {
		teams[idx] = &models.Team{
			ID:            t.Name + "-001",
			Name:          t.Name,
			ProductAreaID: "Mocked-001",
		}
	}

	return teams, nil
}

func (m *Mock) GetProductArea(ctx context.Context, paID string) (*models.ProductArea, error) {
	return &models.ProductArea{
		ID:   "Mocked-001",
		Name: "Mocked Produktområde",
	}, nil
}

func (m *Mock) GetProductAreas(ctx context.Context) ([]*models.ProductArea, error) {
	pas := make([]*models.ProductArea, 1)
	pas[0] = &models.ProductArea{
		ID:   "Mocked-001",
		Name: "Mocked Produktområde",
	}
	return pas, nil
}
