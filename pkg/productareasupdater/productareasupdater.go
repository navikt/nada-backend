package productareasupdater

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/navikt/nada-backend/pkg/database"
	log "github.com/sirupsen/logrus"
)

type ProductAreaUpdater struct {
	url    string
	client *http.Client
	repo   *database.Repo
}

func New(url string, repo *database.Repo) *ProductAreaUpdater {
	return &ProductAreaUpdater{
		url:    url,
		client: http.DefaultClient,
		repo:   repo,
	}
}

func (p *ProductAreaUpdater) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for {
		if err := p.UpdateProductAreas(ctx); err != nil {
			log.WithError(err).Error("updating product areas")
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (p *ProductAreaUpdater) UpdateProductAreas(ctx context.Context) error {
	var productAreas struct {
		Content []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"content"`
	}

	res, err := p.client.Get(p.url + "/productarea?status=ACTIVE")
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bodyBytes, &productAreas); err != nil {
		return err
	}

	for _, pa := range productAreas.Content {
		if err := p.repo.UpsertProductArea(ctx, pa.Name, pa.ID); err != nil {
			log.WithError(err).Error("updating product areas in db")
		}
	}

	return nil
}
