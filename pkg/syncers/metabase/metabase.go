package metabase

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/navikt/nada-backend/pkg/service"
)

type Synchronizer struct {
	service service.MetabaseService
}

func New(service service.MetabaseService) *Synchronizer {
	return &Synchronizer{
		service: service,
	}
}

func (s *Synchronizer) Run(ctx context.Context, frequency time.Duration, log zerolog.Logger) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Info().Msg("running metabase synchronizer")

			err := s.service.SyncAllTablesVisibility(ctx)
			if err != nil {
				log.Error().Err(err).Msg("syncing all tables visibility")
			}
		}
	}
}
