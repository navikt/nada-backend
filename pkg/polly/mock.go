package polly

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

type PollyMock struct {
	url string
}

func NewMock(url string) *PollyMock {
	return &PollyMock{
		url: url,
	}
}

func (m *PollyMock) SearchPolly(ctx context.Context, q string) ([]*models.Polly, error) {
	ret := []*models.Polly{}

	ret = append(ret, &models.Polly{
		ID:   "28570031-e2b3-4110-8864-41ab279e2e0c",
		Name: "Behandling",
		URL:  m.url + "/28570031-e2b3-4110-8864-41ab279e2e0c",
	})
	return ret, nil
}
