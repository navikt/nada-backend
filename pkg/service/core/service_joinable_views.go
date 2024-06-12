package core

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
)

type joinableViewsService struct {
	joinableViewsStorage service.JoinableViewsStorage
	accessStorage        service.AccessStorage
	dataProductStorage   service.DataProductsStorage
	bigQueryAPI          service.BigQueryAPI
	bigQueryStorage      service.BigQueryStorage
}

var _ service.JoinableViewsService = &joinableViewsService{}

func (s *joinableViewsService) GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]service.JoinableViewToBeDeletedWithRefDatasource, error) {
	return s.joinableViewsStorage.GetJoinableViewsToBeDeletedWithRefDatasource(ctx)
}

func (s *joinableViewsService) GetJoinableViewsWithReference(ctx context.Context) ([]service.JoinableViewWithReference, error) {
	return s.joinableViewsStorage.GetJoinableViewsWithReference(ctx)
}

func (s *joinableViewsService) SetJoinableViewDeleted(ctx context.Context, id uuid.UUID) error {
	return s.joinableViewsStorage.SetJoinableViewDeleted(ctx, id)
}

// FIXME: We should pass in the user
func (s *joinableViewsService) GetJoinableViewsForUser(ctx context.Context) ([]service.JoinableView, error) {
	joinableViewsDB, err := s.joinableViewsStorage.GetJoinableViewsForOwner(ctx)
	if err != nil {
		return nil, err
	}

	joinableViewsDBMerged := make(map[uuid.UUID][]service.JoinableViewForOwner)
	for _, vdb := range joinableViewsDB {
		_, _exist := joinableViewsDBMerged[vdb.ID]
		if !_exist {
			joinableViewsDBMerged[vdb.ID] = []service.JoinableViewForOwner{}
		}
		joinableViewsDBMerged[vdb.ID] = append(joinableViewsDBMerged[vdb.ID], vdb)
	}

	var joinableViews []service.JoinableView

	for k, v := range joinableViewsDBMerged {
		newJoinableView := service.JoinableView{
			ID:      k,
			Name:    v[0].Name,
			Created: v[0].Created,
			Expires: v[0].Expires,
		}

		joinableViews = append(joinableViews, newJoinableView)
	}

	return joinableViews, nil
}

func (s *joinableViewsService) GetJoinableView(ctx context.Context, id string) (*service.JoinableViewWithDatasource, error) {
	joinableViewDatasets, err := s.joinableViewsStorage.GetJoinableViewWithDataset(ctx, id)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)

	jv := service.JoinableViewWithDatasource{
		JoinableView: service.JoinableView{
			ID:      joinableViewDatasets[0].JoinableViewID,
			Name:    joinableViewDatasets[0].JoinableViewName,
			Created: joinableViewDatasets[0].JoinableViewCreated,
			Expires: joinableViewDatasets[0].JoinableViewExpires,
		},
	}

	for _, ds := range joinableViewDatasets {
		jvbq := &service.PseudoDatasource{
			BigQueryUrl: s.bigQueryAPI.MakeBigQueryUrlForJoinableViews(jv.Name, ds.BqTable, ds.BqProject, ds.BqDataset),
			Deleted:     ds.Deleted != nil,
		}

		if user.GoogleGroups.Contains(ds.Group) {
			jvbq.Accessible = true
		} else {
			activeAccessList, apierr := s.accessStorage.ListActiveAccessToDataset(ctx, ds.DatasetID.UUID)

			if apierr != nil {
				return nil, apierr
			}

			jvbq.Accessible = false
			for _, access := range activeAccessList {
				if access.Subject == "user:"+user.Email {
					jvbq.Accessible = true
					break
				}
			}
		}

		jv.PseudoDatasources = append(jv.PseudoDatasources, *jvbq)
	}

	return &jv, nil
}

func (s *joinableViewsService) CreateJoinableViews(ctx context.Context, input service.NewJoinableViews) (string, error) {
	user := auth.GetUser(ctx)

	var datasets []*service.Dataset
	for _, dsid := range input.DatasetIDs {
		dataset, apierr := s.dataProductStorage.GetDataset(ctx, dsid.String())
		if apierr != nil {
			return "", apierr
		}
		dataproduct, apierr := s.dataProductStorage.GetDataproduct(ctx, dataset.DataproductID.String())
		if apierr != nil {
			return "", apierr
		}
		if !user.GoogleGroups.Contains(dataproduct.Owner.Group) {
			access, apierr := s.accessStorage.ListActiveAccessToDataset(ctx, dataset.ID)
			if apierr != nil {
				return "", apierr
			}
			accessSet := make(map[string]int)
			for _, da := range access {
				accessSet[da.Subject] = 1
			}
			for _, ugg := range user.GoogleGroups {
				accessSet["group:"+ugg.Email] = 1
			}
			accessSet["user:"+user.Email] = 1
			if len(accessSet) == len(user.GoogleGroups.Emails())+1+len(access) {
				return "", service.ErrUnauthorized
			}
		}
		datasets = append(datasets, dataset)
	}

	var datasources []service.JoinableViewDatasource
	var pseudoDatasourceIDs []uuid.UUID

	for _, ds := range datasets {
		refDatasource, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, ds.ID, true)
		if err != nil {
			return "", fmt.Errorf("get bigquery datasource: %w", err)
		}

		pseudoDatasource, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, ds.ID, false)
		if err != nil {
			return "", fmt.Errorf("get bigquery datasource: %w", err)
		}

		datasources = append(datasources, service.JoinableViewDatasource{
			RefDatasource: &service.DatasourceForJoinableView{
				Project:       refDatasource.ProjectID,
				Dataset:       refDatasource.Dataset,
				Table:         refDatasource.Table,
				PseudoColumns: refDatasource.PseudoColumns,
			},
			PseudoDatasource: &service.DatasourceForJoinableView{
				Project:       pseudoDatasource.ProjectID,
				Dataset:       pseudoDatasource.Dataset,
				Table:         pseudoDatasource.Table,
				PseudoColumns: pseudoDatasource.PseudoColumns,
			},
		})
		pseudoDatasourceIDs = append(pseudoDatasourceIDs, pseudoDatasource.ID)
	}

	projectID, joinableDatasetID, joinableViewsMap, err := s.bigQueryAPI.CreateJoinableViewsForUser(ctx, input.Name, datasources)
	if err != nil {
		return "", fmt.Errorf("create joinable views for user: %w", err)
	}

	for _, d := range datasources {
		dstbl := d.RefDatasource
		if err := s.bigQueryAPI.AddToAuthorizedViews(ctx, dstbl.Project, dstbl.Dataset, projectID, joinableDatasetID, joinableViewsMap[dstbl.Dataset]); err != nil {
			return "", fmt.Errorf("add to authorized views: %w", err)
		}
		if err := s.bigQueryAPI.AddToAuthorizedViews(ctx, projectID, "secrets_vault", projectID, joinableDatasetID, joinableViewsMap[dstbl.Dataset]); err != nil {
			return "", fmt.Errorf("add to secrets' authorized views: %w", err)
		}
	}

	subj := user.Email
	subjType := service.SubjectTypeUser
	subjWithType := subjType + ":" + subj

	for _, v := range joinableViewsMap {
		if err := s.bigQueryAPI.Grant(ctx, projectID, joinableDatasetID, v, subjWithType); err != nil {
			return "", fmt.Errorf("grant: %w", err)
		}
	}

	if _, err := s.joinableViewsStorage.CreateJoinableViewsDB(ctx, joinableDatasetID, user.Email, input.Expires, pseudoDatasourceIDs); err != nil {
		return "", fmt.Errorf("create joinable views db: %w", err)
	}

	return joinableDatasetID, nil
}

func NewJoinableViewsService(joinableViewsStorage service.JoinableViewsStorage) *joinableViewsService {
	return &joinableViewsService{
		joinableViewsStorage: joinableViewsStorage,
	}
}
