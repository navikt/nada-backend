package core

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
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
	const op = "joinableViewsService.GetJoinableViewsToBeDeletedWithRefDatasource"

	views, err := s.joinableViewsStorage.GetJoinableViewsToBeDeletedWithRefDatasource(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return views, nil
}

func (s *joinableViewsService) GetJoinableViewsWithReference(ctx context.Context) ([]service.JoinableViewWithReference, error) {
	const op = "joinableViewsService.GetJoinableViewsWithReference"

	views, err := s.joinableViewsStorage.GetJoinableViewsWithReference(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return views, nil
}

func (s *joinableViewsService) SetJoinableViewDeleted(ctx context.Context, id uuid.UUID) error {
	const op = "joinableViewsService.SetJoinableViewDeleted"

	err := s.joinableViewsStorage.SetJoinableViewDeleted(ctx, id)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *joinableViewsService) GetJoinableViewsForUser(ctx context.Context, user *service.User) ([]service.JoinableView, error) {
	const op = "joinableViewsService.GetJoinableViewsForUser"

	joinableViewsDB, err := s.joinableViewsStorage.GetJoinableViewsForOwner(ctx, user)
	if err != nil {
		return nil, errs.E(op, err)
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

func (s *joinableViewsService) GetJoinableView(ctx context.Context, user *service.User, id uuid.UUID) (*service.JoinableViewWithDatasource, error) {
	const op = "joinableViewsService.GetJoinableView"

	joinableViewDatasets, err := s.joinableViewsStorage.GetJoinableViewWithDataset(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

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
			activeAccessList, err := s.accessStorage.ListActiveAccessToDataset(ctx, ds.DatasetID.UUID)
			if err != nil {
				return nil, errs.E(op, err)
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

func (s *joinableViewsService) CreateJoinableViews(ctx context.Context, user *service.User, input service.NewJoinableViews) (string, error) {
	const op = "joinableViewsService.CreateJoinableViews"

	var datasets []*service.Dataset
	for _, dsid := range input.DatasetIDs {
		dataset, err := s.dataProductStorage.GetDataset(ctx, dsid)
		if err != nil {
			return "", errs.E(op, err)
		}

		dataproduct, err := s.dataProductStorage.GetDataproduct(ctx, dataset.DataproductID)
		if err != nil {
			return "", errs.E(op, err)
		}

		if !user.GoogleGroups.Contains(dataproduct.Owner.Group) {
			access, err := s.accessStorage.ListActiveAccessToDataset(ctx, dataset.ID)
			if err != nil {
				return "", errs.E(op, err)
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
				return "", errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user not authorized to create joinable views"))
			}
		}

		datasets = append(datasets, dataset)
	}

	var datasources []service.JoinableViewDatasource
	var pseudoDatasourceIDs []uuid.UUID

	for _, ds := range datasets {
		refDatasource, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, ds.ID, true)
		if err != nil {
			return "", errs.E(op, err)
		}

		pseudoDatasource, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, ds.ID, false)
		if err != nil {
			return "", errs.E(op, err)
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
		return "", errs.E(op, err)
	}

	for _, d := range datasources {
		dstbl := d.RefDatasource
		if err := s.bigQueryAPI.AddToAuthorizedViews(ctx, dstbl.Project, dstbl.Dataset, projectID, joinableDatasetID, joinableViewsMap[dstbl.Dataset]); err != nil {
			return "", errs.E(op, err)
		}

		if err := s.bigQueryAPI.AddToAuthorizedViews(ctx, projectID, "secrets_vault", projectID, joinableDatasetID, joinableViewsMap[dstbl.Dataset]); err != nil {
			return "", errs.E(op, err)
		}
	}

	subj := user.Email
	subjType := service.SubjectTypeUser
	subjWithType := subjType + ":" + subj

	for _, v := range joinableViewsMap {
		if err := s.bigQueryAPI.Grant(ctx, projectID, joinableDatasetID, v, subjWithType); err != nil {
			return "", errs.E(op, err)
		}
	}

	if _, err := s.joinableViewsStorage.CreateJoinableViewsDB(ctx, joinableDatasetID, user.Email, input.Expires, pseudoDatasourceIDs); err != nil {
		return "", errs.E(op, err)
	}

	return joinableDatasetID, nil
}

func NewJoinableViewsService(
	joinableViewsStorage service.JoinableViewsStorage,
	accessStorage service.AccessStorage,
	dataProductStorage service.DataProductsStorage,
	bigQueryAPI service.BigQueryAPI,
	bigQueryStorage service.BigQueryStorage,
) *joinableViewsService {
	return &joinableViewsService{
		joinableViewsStorage: joinableViewsStorage,
		accessStorage:        accessStorage,
		dataProductStorage:   dataProductStorage,
		bigQueryAPI:          bigQueryAPI,
		bigQueryStorage:      bigQueryStorage,
	}
}
