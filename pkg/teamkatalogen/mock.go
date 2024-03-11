package teamkatalogen

import (
	"context"

	"github.com/navikt/nada-backend/pkg/auth"
)

type Mock struct {
	Teams []TeamkatalogenResult
}

func NewMock() *Mock {
	tk := &Mock{}
	for _, t := range auth.MockUser.GoogleGroups {
		tk.Teams = append(tk.Teams, TeamkatalogenResult{
			Name:          t.Name,
			URL:           "https://some.url",
			Description:   "This is a description of " + t.Name,
			ProductAreaID: "Mocked-001",
			TeamID:        t.Name + "-001",
		})
	}
	return tk
}

func (m *Mock) Search(ctx context.Context, gcpGroups []string) ([]TeamkatalogenResult, error) {
	ret := []TeamkatalogenResult{}
	for _, t := range m.Teams {
		if ContainsAnyCaseInsensitive(t.Name, gcpGroups) {
			ret = append(ret, t)
		}
	}
	return ret, nil
}

func (m *Mock) GetTeamsInProductArea(ctx context.Context, paID string) ([]*Team, error) {
	mockedTeams := CreateMockedTeams()
	teams := make([]*Team, 0)
	for _, t := range mockedTeams {
		if t.ProductAreaID == paID {
			teams = append(teams, t)
		}
	}
	return teams, nil
}

func (m *Mock) GetProductArea(ctx context.Context, paID string) (*ProductArea, error) {
	for _, pa := range mockedProductAreas {
		if pa.ID == paID {
			return pa, nil
		}
	}
	return nil, nil
}

func (m *Mock) GetProductAreas(ctx context.Context) ([]*ProductArea, error) {
	return mockedProductAreas, nil
}

func (m *Mock) GetTeam(ctx context.Context, teamID string) (*Team, error) {
	return &Team{
		ID:            "Team-Frifrokost-001",
		Name:          "Team Frifrokost",
		ProductAreaID: "Mocked-002",
	}, nil
}

func (m *Mock) GetTeamCatalogURL(teamID string) string {
	return "https://teamkatalog.nav.no/team/" + teamID
}

var mockedProductAreas = []*ProductArea{
	{
		ID:       "Mocked-001",
		Name:     "Mocked Produktområde",
		AreaType: "PRODUCT_AREA",
	},
	{
		ID:       "Mocked-002",
		Name:     "PO Fri mat hverdag",
		AreaType: "PRODUCT_AREA",
	},
	{
		ID:       "Mocked-003",
		Name:     "PO Fri alkohol til voksen",
		AreaType: "PROJECT",
	},
}

var mockedTeams []*Team

func CreateMockedTeams() []*Team {
	if mockedTeams != nil {
		return mockedTeams
	}

	mockedTeams = make([]*Team, 0)
	for _, t := range auth.MockUser.GoogleGroups {
		mockedTeams = append(mockedTeams, &Team{
			ID:            t.Name + "-001",
			Name:          t.Name,
			ProductAreaID: "Mocked-001",
		})
	}

	staticMockedTeams := []*Team{
		{
			ID:            "Team-Frifrokost-001",
			Name:          "Team Frifrokost",
			ProductAreaID: "Mocked-002",
		},
		{
			ID:            "Team-Frilunsj-001",
			Name:          "Team Frilunsj",
			ProductAreaID: "Mocked-002",
		},
		{
			ID:            "Team-Frimiddag-001",
			Name:          "Team Frimiddag",
			ProductAreaID: "Mocked-002",
		},
		{
			ID:            "Team-Frivin-001",
			Name:          "Team Frivin",
			ProductAreaID: "Mocked-003",
		},
		{
			ID:            "Team-FriølTilPersonMedVeldigLangNavnSomDenne-001",
			Name:          "Team Friøl hver andre dag til person med veldig lang navn som denne",
			ProductAreaID: "Mocked-003",
		},
	}
	mockedTeams = append(mockedTeams, staticMockedTeams...)
	return mockedTeams
}
