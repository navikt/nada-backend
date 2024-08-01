package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.JoinableViewsStorage = &joinableViewStorage{}

type joinableViewStorage struct {
	db *database.Repo
}

func (s *joinableViewStorage) GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]service.JoinableViewToBeDeletedWithRefDatasource, error) {
	const op errs.Op = "joinableViewStorage.GetJoinableViewsToBeDeletedWithRefDatasource"

	rows, err := s.db.Querier.GetJoinableViewsToBeDeletedWithRefDatasource(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	joinableViews := make([]service.JoinableViewToBeDeletedWithRefDatasource, 0, len(rows))
	for i, row := range rows {
		joinableViews[i] = service.JoinableViewToBeDeletedWithRefDatasource{
			JoinableViewID:   row.JoinableViewID,
			JoinableViewName: row.JoinableViewName,
			BqProjectID:      row.BqProjectID,
			BqDatasetID:      row.BqDatasetID,
			BqTableID:        row.BqTableID,
		}
	}

	return joinableViews, nil
}

func (s *joinableViewStorage) GetJoinableViewsWithReference(ctx context.Context) ([]service.JoinableViewWithReference, error) {
	const op errs.Op = "joinableViewStorage.GetJoinableViewsWithReference"

	rows, err := s.db.Querier.GetJoinableViewsWithReference(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	joinableViews := make([]service.JoinableViewWithReference, 0, len(rows))
	for i, row := range rows {
		joinableViews[i] = service.JoinableViewWithReference{
			Owner:               row.Owner,
			JoinableViewID:      row.JoinableViewID,
			JoinableViewDataset: row.JoinableViewDataset,
			PseudoViewID:        row.PseudoViewID,
			PseudoProjectID:     row.PseudoProjectID,
			PseudoDataset:       row.PseudoDataset,
			PseudoTable:         row.PseudoTable,
			Expires:             row.Expires,
		}
	}

	return joinableViews, nil
}

func (s *joinableViewStorage) SetJoinableViewDeleted(ctx context.Context, id uuid.UUID) error {
	const op errs.Op = "joinableViewStorage.SetJoinableViewDeleted"

	err := s.db.Querier.SetJoinableViewDeleted(ctx, id)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *joinableViewStorage) GetJoinableViewsForOwner(ctx context.Context, user *service.User) ([]service.JoinableViewForOwner, error) {
	const op errs.Op = "joinableViewStorage.GetJoinableViewsForOwner"

	joinableViewsDB, err := s.db.Querier.GetJoinableViewsForOwner(ctx, user.Email)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	views := make([]service.JoinableViewForOwner, len(joinableViewsDB))
	for i, view := range joinableViewsDB {
		views[i] = service.JoinableViewForOwner{
			ID:        view.ID,
			Name:      view.Name,
			Owner:     view.Owner,
			Created:   view.Created,
			Expires:   nullTimeToPtr(view.Expires),
			ProjectID: view.ProjectID,
			DatasetID: view.DatasetID,
			TableID:   view.TableID,
		}
	}

	return views, nil
}

func (s *joinableViewStorage) GetJoinableViewsForReferenceAndUser(ctx context.Context, user string, pseudoDatasetID uuid.UUID) ([]service.JoinableViewForReferenceAndUser, error) {
	const op errs.Op = "joinableViewStorage.GetJoinableViewsForReferenceAndUser"

	joinableViews, err := s.db.Querier.GetJoinableViewsForReferenceAndUser(ctx, gensql.GetJoinableViewsForReferenceAndUserParams{
		Owner:           user,
		PseudoDatasetID: pseudoDatasetID,
	})
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	views := make([]service.JoinableViewForReferenceAndUser, len(joinableViews))

	for i, view := range joinableViews {
		views[i] = service.JoinableViewForReferenceAndUser{
			ID:      view.ID,
			Dataset: view.Dataset,
		}
	}

	return views, nil
}

func (s *joinableViewStorage) GetJoinableViewWithDataset(ctx context.Context, id uuid.UUID) ([]service.JoinableViewWithDataset, error) {
	const op errs.Op = "joinableViewStorage.GetJoinableViewWithDataset"

	joinableViewDatasets, err := s.db.Querier.GetJoinableViewWithDataset(ctx, id)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	views := make([]service.JoinableViewWithDataset, len(joinableViewDatasets))
	for i, view := range joinableViewDatasets {
		views[i] = service.JoinableViewWithDataset{
			BqProject:           view.BqProject,
			BqDataset:           view.BqDataset,
			BqTable:             view.BqTable,
			Deleted:             nullTimeToPtr(view.Deleted),
			DatasetID:           view.DatasetID,
			JoinableViewID:      view.JoinableViewID,
			Group:               view.Group.String,
			JoinableViewName:    view.JoinableViewName,
			JoinableViewCreated: view.JoinableViewCreated,
			JoinableViewExpires: nullTimeToPtr(view.JoinableViewExpires),
		}
	}

	return views, nil
}

func (s *joinableViewStorage) CreateJoinableViewsDB(ctx context.Context, name, owner string, expires *time.Time, datasourceIDs []uuid.UUID) (string, error) {
	const op errs.Op = "joinableViewStorage.CreateJoinableViewsDB"

	tx, err := s.db.GetDB().Begin()
	if err != nil {
		return "", errs.E(errs.Database, op, err)
	}
	defer tx.Rollback()

	q := s.db.Querier.WithTx(tx)

	jv, err := q.CreateJoinableViews(ctx, gensql.CreateJoinableViewsParams{
		Name:    name,
		Owner:   owner,
		Created: time.Now(),
		Expires: ptrToNullTime(expires),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errs.E(errs.NotExist, op, err)
		}

		return "", errs.E(errs.Database, op, err)
	}

	for _, bqid := range datasourceIDs {
		_, err = q.CreateJoinableViewsDatasource(ctx, gensql.CreateJoinableViewsDatasourceParams{
			JoinableViewID: jv.ID,
			DatasourceID:   bqid,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", errs.E(errs.NotExist, op, err)
			}

			return "", errs.E(errs.Database, op, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return "", errs.E(errs.Database, op, err)
	}

	return jv.ID.String(), nil
}

func NewJoinableViewStorage(db *database.Repo) *joinableViewStorage {
	return &joinableViewStorage{
		db: db,
	}
}
