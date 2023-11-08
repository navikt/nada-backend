package access

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Ensurer struct {
	repo               Repo
	r                  Revoker
	googleGroups       *auth.GoogleGroupClient
	centralDataProject string
	log                *logrus.Entry
	errs               *prometheus.CounterVec
}

type Repo interface {
	RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error
	GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID, isReference bool) (models.BigQuery, error)
	GetUnrevokedExpiredAccess(ctx context.Context) ([]*models.Access, error)
	GetJoinableViewsWithReference(ctx context.Context) ([]gensql.GetJoinableViewsWithReferenceRow, error)
	ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*models.Access, error)
	GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error)
}

type Revoker interface {
	Grant(ctx context.Context, projectID, dataset, table, member string) error
	Revoke(ctx context.Context, projectID, dataset, table, member string) error
}

func NewEnsurer(repo Repo, r Revoker, googleGroups *auth.GoogleGroupClient, centralDataProject string, errs *prometheus.CounterVec, log *logrus.Entry) *Ensurer {
	return &Ensurer{
		repo:               repo,
		r:                  r,
		googleGroups:       googleGroups,
		centralDataProject: centralDataProject,
		log:                log,
		errs:               errs,
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
		ds, err := e.repo.GetBigqueryDatasource(ctx, entry.DatasetID, false)
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

	if err := e.ensureJoinableViewAccesses(ctx); err != nil {
		e.log.WithError(err).Error("ensuring joinable view accesses")
	}
}

func (e *Ensurer) ensureJoinableViewAccesses(ctx context.Context) error {
	joinableViews, err := e.repo.GetJoinableViewsWithReference(ctx)
	if err != nil {
		return err
	}

OUTER:
	for _, jv := range joinableViews {
		joinableViewName := bigquery.MakeJoinableViewName(jv.PseudoProjectID, jv.PseudoDataset, jv.PseudoTable)
		datasetOwnerGroup, err := e.repo.GetOwnerGroupOfDataset(ctx, jv.PseudoViewID)
		if err != nil {
			return err
		}
		userGroups, err := e.googleGroups.Groups(ctx, &jv.Owner)
		if err != nil {
			return err
		}

		for _, userGroup := range userGroups {
			if userGroup.Email == datasetOwnerGroup {
				if err := e.r.Grant(ctx, e.centralDataProject, jv.JoinableViewDataset, joinableViewName, fmt.Sprintf("user:%v", jv.Owner)); err != nil {
					e.log.WithError(err).Errorf("Granting IAM access for %v on %v.%v.%v", jv.Owner, e.centralDataProject, jv.JoinableViewDataset, joinableViewName)
					e.errs.WithLabelValues("Grant").Inc()
					continue
				}
				continue OUTER
			}
		}

		accesses, err := e.repo.ListActiveAccessToDataset(ctx, jv.PseudoViewID)
		if err != nil {
			return err
		}

		for _, a := range accesses {
			subjectParts := strings.Split(a.Subject, ":")
			if len(subjectParts) != 2 {
				e.log.Errorf("invalid subject format for %v, should be type:email", a.Subject)
				continue
			}
			subjectWithoutType := subjectParts[1]
			if subjectWithoutType == jv.Owner {
				if err := e.r.Grant(ctx, e.centralDataProject, jv.JoinableViewDataset, joinableViewName, fmt.Sprintf("user:%v", jv.Owner)); err != nil {
					e.log.WithError(err).Errorf("Granting IAM access for %v on %v.%v.%v", jv.Owner, e.centralDataProject, jv.JoinableViewDataset, joinableViewName)
					e.errs.WithLabelValues("Grant").Inc()
					continue
				}
				continue OUTER
			}
		}

		if err := e.r.Revoke(ctx, e.centralDataProject, jv.JoinableViewDataset, joinableViewName, fmt.Sprintf("user:%v", jv.Owner)); err != nil {
			e.log.WithError(err).Errorf("Revoking IAM access for %v on %v.%v.%v", jv.Owner, e.centralDataProject, jv.JoinableViewDataset, joinableViewName)
			e.errs.WithLabelValues("Revoke").Inc()
			continue
		}
	}

	return nil
}
