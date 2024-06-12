package access_ensurer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bqclient"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Service interface {
	GetUnrevokedExpiredAccess(ctx context.Context) ([]*service.Access, error)
	GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID, isReference bool) (*service.BigQuery, *service.APIError)
	RevokeAccessToDataset(ctx context.Context, id uuid.UUID) *service.APIError
	GetPseudoDatasourcesToDelete(ctx context.Context) ([]*service.BigQuery, error)
	SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error
	GetJoinableViewsWithReference(ctx context.Context) ([]gensql.GetJoinableViewsWithReferenceRow, error)
	GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error)
	ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*service.Access, error)
	SetJoinableViewDeleted(ctx context.Context, joinableViewID uuid.UUID) error
	GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]gensql.GetJoinableViewsToBeDeletedWithRefDatasourceRow, error)
}

type Ensurer struct {
	s                  Service
	r                  Revoker
	bq                 BigQuery
	repo               *database.Repo
	googleGroups       *auth.GoogleGroupClient
	centralDataProject string
	log                *logrus.Entry
	errs               *prometheus.CounterVec
}

type BigQuery interface {
	DeleteJoinableDataset(ctx context.Context, datasetID string) error
	DeleteJoinableView(ctx context.Context, joinableViewName, refProjectID, refDatasetID, refTableID string) error
	DeletePseudoView(ctx context.Context, pseudoProjectID, pseudoDatasetID, pseudoTableID string) error
}

type Revoker interface {
	Grant(ctx context.Context, projectID, dataset, table, member string) error
	Revoke(ctx context.Context, projectID, dataset, table, member string) error
}

func NewEnsurer(s Service, r Revoker, bq BigQuery, repo *database.Repo, googleGroups *auth.GoogleGroupClient, centralDataProject string, errs *prometheus.CounterVec, log *logrus.Entry) *Ensurer {
	if s == nil {
		s = ServiceWrapper{}
	}
	return &Ensurer{
		s:                  s,
		r:                  r,
		bq:                 bq,
		repo:               repo,
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
	entries, err := e.s.GetUnrevokedExpiredAccess(ctx)
	if err != nil {
		e.log.WithError(err).Error("Getting unrevoked expired access entries from database")
	}

	for _, entry := range entries {
		ds, apiErr := e.s.GetBigqueryDatasource(ctx, entry.DatasetID, false)
		if apiErr != nil {
			e.log.WithError(apiErr.Err).Error("Getting dataproduct datasource for expired access entry")
			e.errs.WithLabelValues("GetBigqueryDatasource").Inc()
			continue
		}
		if err := e.r.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, entry.Subject); err != nil {
			e.log.WithError(err).Errorf("Revoking IAM access for %v on %v.%v.%v", entry.Subject, ds.ProjectID, ds.Dataset, ds.Table)
			e.errs.WithLabelValues("Revoke").Inc()
			continue
		}
		if err := e.repo.Querier.RevokeAccessToDataset(ctx, entry.ID); err != nil {
			e.log.WithError(err).Errorf("Setting access entry with ID %v to revoked in database", entry.ID)
			e.errs.WithLabelValues("RevokeAccessToDataproduct").Inc()
			continue
		}
	}

	// TODO: enable pseudo feature
	if true {
		return
	}

	if err := e.ensureDeleteJoinableViewBQForDeletedDataset(ctx); err != nil {
		e.log.WithError(err).Error("ensuring delete bq datasource for deleted dataset")
	}

	if err := e.ensureJoinableViewAccesses(ctx); err != nil {
		e.log.WithError(err).Error("ensuring joinable view accesses")
	}

	if err := e.ensureDeletePseudoViewBQForDeletedDataset(ctx); err != nil {
		e.log.WithError(err).Error("ensuring delete pseudo view for deleted dataset")
	}
}

func (e *Ensurer) ensureDeletePseudoViewBQForDeletedDataset(ctx context.Context) error {
	pseudoDatasources, err := e.s.GetPseudoDatasourcesToDelete(ctx)
	if err != nil {
		return err
	}

	if len(pseudoDatasources) == 0 {
		return nil
	}

	e.log.Infof("Delete pseudo views without a dataset: %v", pseudoDatasources)

	for _, pds := range pseudoDatasources {
		if len(pds.PseudoColumns) == 0 {
			e.log.Errorf("deleting pseudo view without pseudo columns, ignored")
			continue
		}

		if err := e.bq.DeletePseudoView(ctx, pds.ProjectID, pds.Dataset, pds.Table); err != nil {
			e.log.WithError(err).Errorf("deleting pseudo view with deleted dataset %v", pds.Dataset)
			continue
		}

		if err := e.s.SetDatasourceDeleted(ctx, pds.ID); err != nil {
			e.log.WithError(err).Errorf("setting pseudo view deleted in db, view id: %v", pds.ID)
		} else {
			e.log.Infof("pseudo view without dataset deleted: %v", pds.ID)
		}
	}
	return nil
}

func (e *Ensurer) ensureDeleteJoinableViewBQForDeletedDataset(ctx context.Context) error {
	jvdatasources, err := e.s.GetJoinableViewsToBeDeletedWithRefDatasource(ctx)
	if err != nil {
		return err
	}

	for _, jvds := range jvdatasources {
		err := e.bq.DeleteJoinableView(ctx, jvds.JoinableViewName, jvds.BqProjectID, jvds.BqDatasetID, jvds.BqTableID)
		if err != nil {
			e.log.WithError(err).Errorf("deleting joinable view with deleted pseudo-datasource %v %v.%v.%v", jvds.JoinableViewName, jvds.BqProjectID, jvds.BqDatasetID, jvds.BqTableID)
			continue
		}
	}

	return nil
}

func (e *Ensurer) ensureJoinableViewAccesses(ctx context.Context) error {
	joinableViews, err := e.s.GetJoinableViewsWithReference(ctx)
	if err != nil {
		e.log.WithError(err).Error("getting joinable views with reference")
		return err
	}

OUTER:
	for _, jv := range joinableViews {
		if hasExpired(jv) {
			if err := e.bq.DeleteJoinableDataset(ctx, jv.JoinableViewDataset); err != nil {
				e.log.WithError(err).Errorf("deleting expired joinable view dataset %v", jv.JoinableViewDataset)
				e.errs.WithLabelValues("DeleteExpiredDataset").Inc()
				continue
			}
			if err := e.s.SetJoinableViewDeleted(ctx, jv.JoinableViewID); err != nil {
				e.log.WithError(err).Errorf("setting joinable view deleted in db, view id: %v", jv.JoinableViewID)
				e.errs.WithLabelValues("SetJoinableViewDeleted").Inc()
				continue
			}
			continue
		}

		joinableViewName := bqclient.MakeJoinableViewName(jv.PseudoProjectID, jv.PseudoDataset, jv.PseudoTable)
		datasetOwnerGroup, err := e.s.GetOwnerGroupOfDataset(ctx, jv.PseudoViewID)
		if err != nil {
			e.log.WithError(err).Errorf("getting owner group of dataset: %v", jv.PseudoViewID)
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

		accesses, err := e.s.ListActiveAccessToDataset(ctx, jv.PseudoViewID)
		if err != nil {
			e.log.WithError(err).Errorf("listing active access to dataset: %v", jv.PseudoViewID)
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

func hasExpired(jv gensql.GetJoinableViewsWithReferenceRow) bool {
	if jv.Expires.Valid {
		return jv.Expires.Time.Before(time.Now())
	}

	return false
}
