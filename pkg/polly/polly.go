package polly

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

type Polly struct {
	client *http.Client
	url    string
}

type PollyResponse struct {
	Content []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
}

func New(url string) *Polly {
	return &Polly{
		client: &http.Client{},
		url:    url,
	}
}

func (p *Polly) SearchPolly(ctx context.Context, q string) ([]*models.PollyResult, error) {
	var pr PollyResponse
	res, err := p.client.Get(p.url + "/" + q + "?includePurpose=true")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bodyBytes, &pr); err != nil {
		return nil, err
	}

	ret := []*models.PollyResult{}
	for _, r := range pr.Content {
		ret = append(ret, &models.PollyResult{
			ID:   uuid.MustParse(r.ID),
			Name: r.Name,
		})
	}

	return ret, nil
}
