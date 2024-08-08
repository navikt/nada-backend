package serviceaccount

import (
	"context"
	"time"

	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Watcher struct {
	dataproductsService service.DataProductsStorage
	updateFrequency     time.Duration
	log                 zerolog.Logger
}

func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(w.updateFrequency)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			datasets, err := w.dataproductsService.GetDatasetsMinimal(ctx)
			if err != nil {
				return
			}

			for _, ds := range datasets {
				log.Info().Msgf("Checking service account for dataset %s", ds.Name)
			}
		}
	}
}

func NewWatcher(log zerolog.Logger) *Watcher {
	return &Watcher{
		dataproductsService: nil,
		updateFrequency:     0,
		log:                 log,
	}
}
