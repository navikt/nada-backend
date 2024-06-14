package http

import (
	"context"
	"encoding/json"
	"github.com/navikt/nada-backend/pkg/service"
	"io"
	"net/http"
	"os"
	"time"
)

var _ service.PollyAPI = &pollyAPI{}

type pollyAPI struct {
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

func (p *pollyAPI) SearchPolly(_ context.Context, q string) ([]*service.QueryPolly, error) {
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

	var ret []*service.QueryPolly
	for _, r := range pr.Content[:numRes] {
		ret = append(ret, &service.QueryPolly{
			ExternalID: r.ID,
			Name:       r.Name,
			URL:        p.url + "/" + r.Purpose.Code + "/" + r.ID,
		})
	}

	return ret, nil
}

func NewPollyAPI(url, apiURL string) *pollyAPI {
	// FIXME: make these configurable
	url = "https://behandlingskatalog.intern.dev.nav.no/process/purpose"
	if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
		url = "https://behandlingskatalog.intern.nav.no/process/purpose"
	}

	return &pollyAPI{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		apiURL: apiURL,
		url:    url,
	}
}
