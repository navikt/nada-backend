package polly

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

type Polly interface {
	SearchPolly(ctx context.Context, q string) ([]*QueryPolly, error)
}

type PollyClient struct {
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

func New(apiURL string) *PollyClient {
	url := "https://behandlingskatalog.intern.dev.nav.no/process/purpose"
	if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
		url = "https://behandlingskatalog.intern.nav.no/process/purpose"
	}

	return &PollyClient{
		client: &http.Client{},
		apiURL: apiURL,
		url:    url,
	}
}

type QueryPolly struct {
	ExternalID string `json:"externalID"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}

func (p *PollyClient) SearchPolly(ctx context.Context, q string) ([]*QueryPolly, error) {
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

	numRes := 10
	if len(pr.Content) < 10 {
		numRes = len(pr.Content)
	}

	var ret []*QueryPolly
	for _, r := range pr.Content[:numRes] {
		ret = append(ret, &QueryPolly{
			ExternalID: r.ID,
			Name:       r.Name,
			URL:        p.url + "/" + r.Purpose.Code + "/" + r.ID,
		})
	}

	return ret, nil
}
