package metabase

import (
	"context"
	"time"

	"github.com/navikt/nada-backend/pkg/service"
	"github.com/sirupsen/logrus"
)

type Synchronizer struct {
	service service.MetabaseService
}

func New(service service.MetabaseService) *Synchronizer {
	return &Synchronizer{
		service: service,
	}
}

func (s *Synchronizer) Run(ctx context.Context, frequency time.Duration, log *logrus.Entry) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Info("running metabase synchronizer")

			err := s.service.SyncAllTablesVisibility(ctx)
			if err != nil {
				log.WithError(err).Error("metabase synchronizer")
			}
		}
	}
}
