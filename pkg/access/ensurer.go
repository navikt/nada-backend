package access

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/sirupsen/logrus"
)

type Ensurer struct {
	repo Repo
	am   AccessManager
	log  *logrus.Entry
}

type Repo interface {
	RevokeAccessToDataproduct(ctx context.Context, id uuid.UUID) error
	GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID) (models.BigQuery, error)
	GetUnrevokedExpiredAccess(ctx context.Context) ([]*models.Access, error)
}
type AccessManager interface {
	Revoke(ctx context.Context, projectID, dataset, table, member string) error
}

func NewEnsurer(repo Repo, am AccessManager, log *logrus.Entry) *Ensurer {
	return &Ensurer{
		repo: repo,
		am:   am,
		log:  log,
	}
}

func (e *Ensurer) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for {
		e.run(ctx)
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (e *Ensurer) run(ctx context.Context) {
	entries, err := e.repo.GetUnrevokedExpiredAccess(ctx)
	if err != nil {
		e.log.WithError(err).Error("Getting unrevoked expired access entries from database")
	}

	for _, entry := range entries {
		ds, err := e.repo.GetBigqueryDatasource(ctx, entry.DataproductID)
		if err != nil {
			e.log.WithError(err).Error("Getting dataproduct datasource for expired access entry")
			//TODO(jhrv): bump error metric
			continue
		}
		if err := e.am.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, entry.Subject); err != nil {
			e.log.WithError(err).Errorf("Revoking IAM access for %v on %v.%v.%v", entry.Subject, ds.ProjectID, ds.Dataset, ds.Table)
			//TODO(jhrv): bump error metric
			continue
		}
		if err := e.repo.RevokeAccessToDataproduct(ctx, entry.ID); err != nil {
			e.log.WithError(err).Errorf("Setting access entry with ID %v to revoked in database", entry.ID)
			//TODO(jhrv): bump error metric
			continue
		}
	}
}
