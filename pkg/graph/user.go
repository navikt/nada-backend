package graph

import (
	"strings"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

func teamNamesFromGroups(groups []*models.Group) []string {
	teams := []string{}
	for _, g := range groups {
		teams = append(teams, strings.Split(g.Email, "@")[0])
	}

	return teams
}
