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
			Name:        t.Name,
			URL:         "https://some.url",
			Description: "This is a description of " + t.Name,
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
