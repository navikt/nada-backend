package tk

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Static struct {
	ProductAreas map[uuid.UUID]*ProductArea
	Teams        map[uuid.UUID]*Team
	apiURL       string
}

func (s *Static) GetProductAreas(_ context.Context) (*ProductAreas, error) {
	var productAreas ProductAreas

	for _, pa := range s.ProductAreas {
		productAreas.Content = append(productAreas.Content, *pa)
	}

	return &productAreas, nil
}

func (s *Static) GetTeams(_ context.Context) (*Teams, error) {
	var teams Teams

	for _, team := range s.Teams {
		teams.Content = append(teams.Content, *team)
	}

	return &teams, nil
}

func (s *Static) GetTeam(_ context.Context, teamID uuid.UUID) (*Team, error) {
	team, ok := s.Teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found")
	}

	return team, nil
}

func (s *Static) GetTeamsInProductArea(_ context.Context, productAreaID uuid.UUID) (*Teams, error) {
	if _, ok := s.ProductAreas[productAreaID]; !ok {
		return nil, fmt.Errorf("product area not found")
	}

	var teams Teams

	for _, team := range s.Teams {
		if team.ProductAreaID == productAreaID {
			teams.Content = append(teams.Content, *team)
		}
	}

	return &teams, nil
}

func (s *Static) GetTeamCatalogURL(teamID uuid.UUID) string {
	return fmt.Sprintf("%s/team/%s", s.apiURL, teamID.String())
}

func NewStatic(apiURL string, productAreas []*ProductArea, teams []*Team) *Static {
	productAreaMap := make(map[uuid.UUID]*ProductArea)
	for _, pa := range productAreas {
		productAreaMap[pa.ID] = pa
	}

	teamMap := make(map[uuid.UUID]*Team)
	for _, team := range teams {
		teamMap[team.ID] = team
	}

	return &Static{
		ProductAreas: productAreaMap,
		Teams:        teamMap,
		apiURL:       apiURL,
	}
}
