package access_ensurer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/rs/zerolog"

	"github.com/navikt/nada-backend/pkg/service"
	"github.com/prometheus/client_golang/prometheus"
)

type Ensurer struct {
	accessStorage       service.AccessStorage
	metabaseService     service.MetabaseService
	dataProductsStorage service.DataProductsStorage
	bigQueryStorage     service.BigQueryStorage
	bigQueryAPI         service.BigQueryAPI
	bigQueryService     service.BigQueryService
	joinableViewService service.JoinableViewsService

	googleGroups       *auth.GoogleGroupClient
	centralDataProject string
	log                zerolog.Logger
	errs               *prometheus.CounterVec
}

func NewEnsurer(
	googleGroups *auth.GoogleGroupClient,
	centralDataProject string,
	errs *prometheus.CounterVec,
	accessStorage service.AccessStorage,
	metabaseService service.MetabaseService,
	dataProductsStorage service.DataProductsStorage,
	bigQueryStorage service.BigQueryStorage,
	bigQueryAPI service.BigQueryAPI,
	bigQueryService service.BigQueryService,
	joinableViewService service.JoinableViewsService,
	log zerolog.Logger,
) *Ensurer {
	return &Ensurer{
		accessStorage:       accessStorage,
		metabaseService:     metabaseService,
		dataProductsStorage: dataProductsStorage,
		bigQueryStorage:     bigQueryStorage,
		bigQueryAPI:         bigQueryAPI,
		bigQueryService:     bigQueryService,
		joinableViewService: joinableViewService,
		googleGroups:        googleGroups,
		centralDataProject:  centralDataProject,
		log:                 log,
		errs:                errs,
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
	entries, err := e.accessStorage.GetUnrevokedExpiredAccess(ctx)
	if err != nil {
		e.log.Error().Err(err).Msg("getting unrevoked expired access entries from database")
	}

	for _, entry := range entries {
		ds, err := e.bigQueryStorage.GetBigqueryDatasource(ctx, entry.DatasetID, false)
		if err != nil {
			e.log.Error().Err(err).Msg("getting dataproduct datasource for expired access entry")
			e.errs.WithLabelValues("GetBigqueryDatasource").Inc()
			continue
		}
		if err := e.bigQueryAPI.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, entry.Subject); err != nil {
			e.log.Error().Err(err).Msgf("revoking IAM access for %v on %v.%v.%v", entry.Subject, ds.ProjectID, ds.Dataset, ds.Table)
			e.errs.WithLabelValues("Revoke").Inc()
			continue
		}

		if err := e.accessStorage.RevokeAccessToDataset(ctx, entry.ID); err != nil {
			e.log.Error().Err(err).Msgf("setting access entry with ID %v to revoked in database", entry.ID)
			e.errs.WithLabelValues("RevokeAccessToDataproduct").Inc()
			continue
		}

		if err := e.metabaseService.RevokeMetabaseAccess(ctx, entry.DatasetID, entry.Subject); err != nil {
			e.log.Error().Err(err).Msgf("revoking access to Metabase for access ID %v", entry.ID)
			e.errs.WithLabelValues("RevokeAccessToMetabase").Inc()
			continue
		}
	}

	// TODO: enable pseudo feature
	if true {
		return
	}

	if err := e.ensureDeleteJoinableViewBQForDeletedDataset(ctx); err != nil {
		e.log.Error().Err(err).Msg("ensuring delete joinable view for deleted dataset")
	}

	if err := e.ensureJoinableViewAccesses(ctx); err != nil {
		e.log.Error().Err(err).Msg("ensuring joinable view accesses")
	}

	if err := e.ensureDeletePseudoViewBQForDeletedDataset(ctx); err != nil {
		e.log.Error().Err(err).Msg("ensuring delete pseudo view for deleted dataset")
	}
}

func (e *Ensurer) ensureDeletePseudoViewBQForDeletedDataset(ctx context.Context) error {
	pseudoDatasources, err := e.bigQueryStorage.GetPseudoDatasourcesToDelete(ctx)
	if err != nil {
		return err
	}

	if len(pseudoDatasources) == 0 {
		return nil
	}

	e.log.Info().Msgf("delete pseudo views without a dataset: %v", pseudoDatasources)

	for _, pds := range pseudoDatasources {
		if len(pds.PseudoColumns) == 0 {
			e.log.Error().Msgf("deleting pseudo view without pseudo columns, ignored")
			continue
		}

		if err := e.bigQueryAPI.DeletePseudoView(ctx, pds.ProjectID, pds.Dataset, pds.Table); err != nil {
			e.log.Error().Err(err).Msgf("deleting pseudo view with deleted dataset %v", pds.Dataset)
			continue
		}

		if err := e.dataProductsStorage.SetDatasourceDeleted(ctx, pds.ID); err != nil {
			e.log.Error().Err(err).Msgf("setting pseudo view deleted in db, view id: %v", pds.ID)
		} else {
			e.log.Info().Msgf("pseudo view without dataset deleted: %v", pds.ID)
		}
	}
	return nil
}

func (e *Ensurer) ensureDeleteJoinableViewBQForDeletedDataset(ctx context.Context) error {
	jvdatasources, err := e.joinableViewService.GetJoinableViewsToBeDeletedWithRefDatasource(ctx)
	if err != nil {
		return err
	}

	for _, jvds := range jvdatasources {
		err := e.bigQueryAPI.DeleteJoinableView(ctx, jvds.JoinableViewName, jvds.BqProjectID, jvds.BqDatasetID, jvds.BqTableID)
		if err != nil {
			e.log.Error().Err(err).Msgf("deleting joinable view with deleted pseudo-datasource %v %v.%v.%v", jvds.JoinableViewName, jvds.BqProjectID, jvds.BqDatasetID, jvds.BqTableID)
			continue
		}
	}

	return nil
}

// FIXME: duplicated
func makeJoinableViewName(projectID, datasetID, tableID string) string {
	// datasetID will always be same markedsplassen dataset id
	return fmt.Sprintf("%v_%v", projectID, tableID)
}

func (e *Ensurer) ensureJoinableViewAccesses(ctx context.Context) error {
	joinableViews, err := e.joinableViewService.GetJoinableViewsWithReference(ctx)
	if err != nil {
		e.log.Error().Err(err).Msg("getting joinable views with reference")
		return err
	}

OUTER:
	for _, jv := range joinableViews {
		if hasExpired(jv) {
			if err := e.bigQueryAPI.DeleteJoinableDataset(ctx, jv.JoinableViewDataset); err != nil {
				e.log.Error().Err(err).Msgf("deleting expired joinable view dataset %v", jv.JoinableViewDataset)
				e.errs.WithLabelValues("DeleteExpiredDataset").Inc()
				continue
			}
			if err := e.joinableViewService.SetJoinableViewDeleted(ctx, jv.JoinableViewID); err != nil {
				e.log.Error().Err(err).Msgf("setting joinable view deleted in db, view id: %v", jv.JoinableViewID)
				e.errs.WithLabelValues("SetJoinableViewDeleted").Inc()
				continue
			}
			continue
		}

		joinableViewName := makeJoinableViewName(jv.PseudoProjectID, jv.PseudoDataset, jv.PseudoTable)
		datasetOwnerGroup, err := e.dataProductsStorage.GetOwnerGroupOfDataset(ctx, jv.PseudoViewID)
		if err != nil {
			e.log.Error().Err(err).Msgf("getting owner group of dataset: %v", jv.PseudoViewID)
			return err
		}
		userGroups, err := e.googleGroups.Groups(ctx, &jv.Owner)
		if err != nil {
			return err
		}

		for _, userGroup := range userGroups {
			if userGroup.Email == datasetOwnerGroup {
				if err := e.bigQueryAPI.Grant(ctx, e.centralDataProject, jv.JoinableViewDataset, joinableViewName, fmt.Sprintf("user:%v", jv.Owner)); err != nil {
					e.log.Error().Err(err).Msgf("Granting IAM access for %v on %v.%v.%v", jv.Owner, e.centralDataProject, jv.JoinableViewDataset, joinableViewName)
					e.errs.WithLabelValues("Grant").Inc()
					continue
				}
				continue OUTER
			}
		}

		accesses, err := e.accessStorage.ListActiveAccessToDataset(ctx, jv.PseudoViewID)
		if err != nil {
			e.log.Error().Err(err).Msgf("listing active access to dataset: %v", jv.PseudoViewID)
			return err
		}

		for _, a := range accesses {
			subjectParts := strings.Split(a.Subject, ":")
			if len(subjectParts) != 2 {
				e.log.Error().Msgf("invalid subject format for %v, should be type:email", a.Subject)
				continue
			}
			subjectWithoutType := subjectParts[1]
			if subjectWithoutType == jv.Owner {
				if err := e.bigQueryAPI.Grant(ctx, e.centralDataProject, jv.JoinableViewDataset, joinableViewName, fmt.Sprintf("user:%v", jv.Owner)); err != nil {
					e.log.Error().Err(err).Msgf("Granting IAM access for %v on %v.%v.%v", jv.Owner, e.centralDataProject, jv.JoinableViewDataset, joinableViewName)
					e.errs.WithLabelValues("Grant").Inc()
					continue
				}
				continue OUTER
			}
		}

		if err := e.bigQueryAPI.Revoke(ctx, e.centralDataProject, jv.JoinableViewDataset, joinableViewName, fmt.Sprintf("user:%v", jv.Owner)); err != nil {
			e.log.Error().Err(err).Msgf("Revoking IAM access for %v on %v.%v.%v", jv.Owner, e.centralDataProject, jv.JoinableViewDataset, joinableViewName)
			e.errs.WithLabelValues("Revoke").Inc()
			continue
		}
	}

	return nil
}

func hasExpired(jv service.JoinableViewWithReference) bool {
	if jv.Expires.Valid {
		return jv.Expires.Time.Before(time.Now())
	}

	return false
}
