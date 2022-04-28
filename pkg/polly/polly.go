package polly

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

type Polly struct {
	client *http.Client
	apiURL string
	url    string
}

type PollyResponse struct {
	Content []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Purpose struct {
			Code string `json:"code"`
		}
	}
}

func New(apiURL string) *Polly {
	url := "https://behandlingskatalog.dev.adeo.no/process/purpose"
	if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
		url = "https://behandlingskatalog.adeo.no/process/purpose"
	}

	return &Polly{
		client: &http.Client{},
		apiURL: apiURL,
		url:    url,
	}
}

func (p *Polly) SearchPolly(ctx context.Context, q string) ([]*models.Polly, error) {
	var pr PollyResponse
	res, err := p.client.Get(p.apiURL + "/search/" + q + "?includePurpose=true")
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

	var ret []*models.Polly
	for _, r := range pr.Content {
		ret = append(ret, &models.Polly{
			ExternalID: r.ID,
			Name:       r.Name,
			URL:        p.url + "/" + r.Purpose.Code + "/" + r.ID,
		})
	}

	return ret, nil
}
