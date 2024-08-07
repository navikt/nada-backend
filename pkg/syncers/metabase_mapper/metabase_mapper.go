package metabase_mapper

import (
	"context"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/errs"

	"github.com/navikt/nada-backend/pkg/leaderelection"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"
)

type Work struct {
	DatasetID uuid.UUID
	Services  []string
}

type Mapper struct {
	Queue                    chan Work
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
		Queue:                    make(chan Work, 100), //nolint: gomnd
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
			m.log.Info().Msg("Checking for new mappings")

			isLeader, err := leaderelection.IsLeader()
			if err != nil {
				m.log.Error().Err(err).Msg("checking leader status")
			}

			if !isLeader {
				m.log.Info().Msg("Not leader, skipping mapping")
				continue
			}

			var workItems []Work

			add, err := m.thirdPartyMappingStorage.GetAddMetabaseDatasetMappings(ctx)
			if err != nil {
				m.log.Error().Err(err).Msg("getting add metabase mappings")
			}

			for _, datasetID := range add {
				workItems = append(workItems, Work{
					DatasetID: datasetID,
					Services: []string{
						service.MappingServiceMetabase,
					},
				})
			}

			remove, err := m.thirdPartyMappingStorage.GetRemoveMetabaseDatasetMappings(ctx)
			if err != nil {
				m.log.Error().Err(err).Msg("getting remove metabase mappings")
			}

			for _, datasetID := range remove {
				workItems = append(workItems, Work{
					DatasetID: datasetID,
					Services:  []string{},
				})
			}

			for _, work := range workItems {
				m.MapDataset(ctx, work.DatasetID, work.Services)
			}
		case work := <-m.Queue:
			m.MapDataset(ctx, work.DatasetID, work.Services)
		case <-ctx.Done():
			m.log.Info().Msg("Shutting down metabase mapper")
			return
		}
	}
}

func (m *Mapper) MapDataset(ctx context.Context, datasetID uuid.UUID, services []string) {
	deadline := time.Duration(m.mappingDeadlineSec) * time.Second

	m.log.Info().Fields(map[string]interface{}{
		"dataset_id":       datasetID.String(),
		"services":         strings.Join(services, ","),
		"deadline_seconds": deadline.Seconds(),
	}).Msg("mapping dataset")

	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.mappingDeadlineSec)*time.Second)
	defer cancel()

	err := m.metabaseService.MapDataset(ctx, datasetID, services)
	if err != nil {
		m.log.Error().
			Err(err).
			Fields(map[string]interface{}{
				"dataset_id": datasetID.String(),
				"services":   services,
				"stack":      errs.OpStack(err),
			}).
			Msg("mapping dataset")
	}
}
