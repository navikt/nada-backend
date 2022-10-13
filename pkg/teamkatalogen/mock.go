package teamkatalogen

import (
	"context"
	"errors"
	"strconv"
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
	teams := GetMockedTeams()
	teamsInPA := make([]*models.Team, 0)
	for _, t := range teams {
		if t.ProductAreaID == paID {
			teamsInPA = append(teamsInPA, t)
		}
	}
	return teamsInPA, nil
}

func (m *Mock) GetProductArea(ctx context.Context, paID string) (*models.ProductArea, error) {
	pas, _ := m.GetProductAreas(ctx)
	for _, pa := range pas {
		if pa.ID == paID {
			return pa, nil
		}
	}
	return nil, errors.New("Invalid product area ID")
}

var mockedProductAreas = []*models.ProductArea{
	{
		ID:   "MOCKED-PO-ID-001",
		Name: "PO-FriMatHverdag",
	},
	{
		ID:   "MOCKED-PO-ID-002",
		Name: "PO-FriAlkoholForVoksen",
	},
}

var mockedTeams []*models.Team

func GetMockedTeams() []*models.Team {
	if mockedTeams != nil {
		return mockedTeams
	}

	mockedTeams = make([]*models.Team, len(auth.MockUser.GoogleGroups))
	for idx, t := range auth.MockUser.GoogleGroups {
		mockedTeams[idx] = &models.Team{
			ID:            strconv.Itoa(idx),
			Name:          t.Name,
			ProductAreaID: "",
		}
		if idx < 3 {
			mockedTeams[idx].ProductAreaID = mockedProductAreas[0].ID
		} else {
			mockedTeams[idx].ProductAreaID = mockedProductAreas[1].ID
		}
	}
	return mockedTeams
}

func (m *Mock) GetProductAreas(ctx context.Context) ([]*models.ProductArea, error) {
	return mockedProductAreas, nil
}
