package polly

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

type PollyMock struct {
	apiURL string
	url    string
}

func NewMock(apiURL string) *PollyMock {
	url := "https://some.url"

	return &PollyMock{
		apiURL: apiURL,
		url:    url,
	}
}

func (m *PollyMock) SearchPolly(ctx context.Context, q string) ([]*models.Polly, error) {
	var ret []*models.Polly

	ret = append(ret, &models.Polly{
		ExternalID: "28570031-e2b3-4110-8864-41ab279e2e0c",
		Name:       "Behandling",
		URL:        m.url + "/28570031-e2b3-4110-8864-41ab279e2e0c",
	})
	return ret, nil
}
