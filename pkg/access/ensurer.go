package access

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Ensurer struct {
	repo Repo
	r    Revoker
	log  *logrus.Entry
	errs *prometheus.CounterVec
}

type Repo interface {
	RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error
	GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID) (models.BigQuery, error)
	GetUnrevokedExpiredAccess(ctx context.Context) ([]*models.Access, error)
}

type Revoker interface {
	Revoke(ctx context.Context, projectID, dataset, table, member string) error
}

func NewEnsurer(repo Repo, r Revoker, errs *prometheus.CounterVec, log *logrus.Entry) *Ensurer {
	return &Ensurer{
		repo: repo,
		r:    r,
		log:  log,
		errs: errs,
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
		ds, err := e.repo.GetBigqueryDatasource(ctx, entry.DatasetID)
		if err != nil {
			e.log.WithError(err).Error("Getting dataproduct datasource for expired access entry")
			e.errs.WithLabelValues("GetBigqueryDatasource").Inc()
			continue
		}
		if err := e.r.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, entry.Subject); err != nil {
			e.log.WithError(err).Errorf("Revoking IAM access for %v on %v.%v.%v", entry.Subject, ds.ProjectID, ds.Dataset, ds.Table)
			e.errs.WithLabelValues("Revoke").Inc()
			continue
		}
		if err := e.repo.RevokeAccessToDataset(ctx, entry.ID); err != nil {
			e.log.WithError(err).Errorf("Setting access entry with ID %v to revoked in database", entry.ID)
			e.errs.WithLabelValues("RevokeAccessToDataproduct").Inc()
			continue
		}
	}
}
