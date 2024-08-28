package metabase_collections

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/leaderelection"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"
)

type Syncer struct {
	api          service.MetabaseAPI
	storage      service.MetabaseStorage
	log          zerolog.Logger
	syncInterval time.Duration
}

func (s *Syncer) Run(ctx context.Context, initialDelaySec int) {
	isLeader, err := leaderelection.IsLeader()
	if err != nil {
		s.log.Error().Err(err).Msg("checking leader status")
		return
	}

	if isLeader {
		// Delay a little before starting
		time.Sleep(time.Duration(initialDelaySec) * time.Second)

		// Do an initial sync
		s.log.Info().Msg("running initial metabase collections syncer")

		err = s.AddRestrictedTagToCollections(ctx)
		if err != nil {
			s.log.Error().Fields(map[string]interface{}{"stack": errs.OpStack(err)}).Err(err).Msg("adding restricted tag to collections")
		}
	}

	if !isLeader {
		s.log.Info().Msg("not leader, skipping metabase collections sync")
	}

	ticker := time.NewTicker(s.syncInterval)

	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.log.Info().Msg("context done, stopping metabase collections syncer")
			return
		case <-ticker.C:
			s.log.Info().Msg("running metabase collections syncer")

			isLeader, err := leaderelection.IsLeader()
			if err != nil {
				s.log.Error().Err(err).Msg("checking leader status")
				continue
			}

			if !isLeader {
				s.log.Info().Msg("not leader, skipping metabase collections sync")
				continue
			}

			err = s.AddRestrictedTagToCollections(ctx)
			if err != nil {
				s.log.Error().Fields(map[string]interface{}{"stack": errs.OpStack(err)}).Err(err).Msg("adding restricted tag to collections")
			}

			report, err := s.CollectionsReport(ctx)
			if err != nil {
				s.log.Error().Fields(map[string]interface{}{"stack": errs.OpStack(err)}).Err(err).Msg("reporting missing collections")
				continue
			}

			for _, missing := range report.Missing {
				s.log.Warn().Fields(map[string]interface{}{
					"dataset_id":    missing.DatasetID,
					"collection_id": missing.CollectionID,
					"database_id":   missing.DatabaseID,
				}).Msg("collection_not_in_metabase")
			}

			for _, dangling := range report.Dangling {
				s.log.Info().Fields(map[string]interface{}{
					"collection_id":   dangling.ID,
					"collection_name": dangling.Name,
				}).Msg("collection_not_in_database")
			}
		}
	}
}

// Dangling means that a collection has been created in metabase but not stored
// in our database
type Dangling struct {
	ID   int
	Name string
}

// Missing means that a collection has been stored in our database but no
// longer exists in metabase
type Missing struct {
	DatasetID    string
	CollectionID int
	DatabaseID   int
}

type CollectionsReport struct {
	Dangling []Dangling
	Missing  []Missing
}

func (s *Syncer) CollectionsReport(ctx context.Context) (*CollectionsReport, error) {
	const op errs.Op = "metabase_collections.Syncer.CollectionsReport"

	metas, err := s.storage.GetAllMetadata(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	collections, err := s.api.GetCollections(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	collectionByID := make(map[int]*service.MetabaseCollection)
	for _, collection := range collections {
		collectionByID[collection.ID] = collection
	}

	report := &CollectionsReport{}

	for _, meta := range metas {
		if meta.SyncCompleted != nil && *meta.CollectionID != 0 {
			_, ok := collectionByID[*meta.CollectionID]
			if !ok {
				report.Missing = append(report.Missing, Missing{
					DatasetID:    meta.DatasetID.String(),
					CollectionID: *meta.CollectionID,
					DatabaseID:   *meta.DatabaseID,
				})
			}

			delete(collectionByID, *meta.CollectionID)
		}
	}

	for id, collection := range collectionByID {
		report.Dangling = append(report.Dangling, Dangling{
			ID:   id,
			Name: collection.Name,
		})
	}

	return report, nil
}

func (s *Syncer) AddRestrictedTagToCollections(ctx context.Context) error {
	const op errs.Op = "metabase_collections.Syncer.AddRestrictedTagToCollections"

	metas, err := s.storage.GetAllMetadata(ctx)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	collections, err := s.api.GetCollections(ctx)
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	collectionByID := make(map[int]*service.MetabaseCollection)
	for _, collection := range collections {
		collectionByID[collection.ID] = collection
	}

	s.log.Info().Msgf("collections: %v", collections)

	for _, meta := range metas {
		s.log.Debug().Msgf("meta: %v", meta)

		if meta.SyncCompleted != nil && meta.CollectionID != nil && *meta.CollectionID != 0 {
			collection, ok := collectionByID[*meta.CollectionID]
			if !ok {
				continue
			}

			if !strings.Contains(collection.Name, service.MetabaseRestrictedCollectionTag) {
				newName := fmt.Sprintf("%s %s", collection.Name, service.MetabaseRestrictedCollectionTag)

				s.log.Info().Fields(map[string]interface{}{
					"collection_id": collection.ID,
					"existing_name": collection.Name,
					"new_name":      newName,
				}).Msg("adding_restricted_tag")

				err := s.api.UpdateCollection(ctx, &service.MetabaseCollection{
					ID:   collection.ID,
					Name: newName,
				})
				if err != nil {
					return errs.E(op, err)
				}
			}
		}
	}

	return nil
}

func New(api service.MetabaseAPI, storage service.MetabaseStorage, syncIntervalSec int, log zerolog.Logger) *Syncer {
	return &Syncer{
		api:          api,
		storage:      storage,
		log:          log,
		syncInterval: time.Duration(syncIntervalSec) * time.Second,
	}
}
