package metabase_mapper

import (
	"context"
	"time"

	"github.com/navikt/nada-backend/pkg/leaderelection"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"
)

type Mapper struct {
	Queue                    chan uuid.UUID
	ticker                   *time.Ticker
	mappingDeadlineSec       int
	metabaseService          service.MetabaseService
	thirdPartyMappingStorage service.ThirdPartyMappingStorage
	log                      zerolog.Logger
}

func New(
	metabaseService service.MetabaseService,
	thirdPartyMappingStorage service.ThirdPartyMappingStorage,
	mappingDeadlineSec, mappingFrequencySec int,
	log zerolog.Logger,
) *Mapper {
	return &Mapper{
		Queue:                    make(chan uuid.UUID, 100),
		ticker:                   time.NewTicker(time.Duration(mappingFrequencySec) * time.Second),
		mappingDeadlineSec:       mappingDeadlineSec,
		metabaseService:          metabaseService,
		thirdPartyMappingStorage: thirdPartyMappingStorage,
		log:                      log,
	}
}

func (m *Mapper) Run(ctx context.Context) {
	m.log.Info().Msg("Starting metabase mapper")

	for {
		select {
		case <-m.ticker.C:
			isLeader, err := leaderelection.IsLeader()
			if err != nil {
				m.log.Error().Err(err).Msg("checking leader status")
			}

			if !isLeader {
				m.log.Info().Msg("Not leader, skipping mapping")
				continue
			}

			mappings, err := m.thirdPartyMappingStorage.GetUnprocessedMetabaseDatasetMappings(ctx)
			if err != nil {
				m.log.Error().Err(err).Msg("getting unprocessed metabase mappings")
			}

			for _, datasetID := range mappings {
				m.MapDataset(ctx, datasetID)
			}
		case datasetID := <-m.Queue:
			m.MapDataset(ctx, datasetID)
		case <-ctx.Done():
			m.log.Info().Msg("Shutting down metabase mapper")
			return
		}
	}
}

func (m *Mapper) MapDataset(ctx context.Context, datasetID uuid.UUID) {
	deadline := time.Duration(m.mappingDeadlineSec) * time.Second

	m.log.Info().Str("dataset_id", datasetID.String()).Float64("deadline_seconds", deadline.Seconds()).Msg("mapping dataset")

	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.mappingDeadlineSec)*time.Second)
	defer cancel()

	err := m.metabaseService.MapDataset(ctx, datasetID, []string{service.MappingServiceMetabase})
	if err != nil {
		m.log.Error().Err(err).Msg("mapping dataset")
	}
}