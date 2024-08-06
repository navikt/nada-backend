package core

import (
	"context"
	"fmt"
	"html"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.DataProductsService = &dataProductsService{}

type dataProductsService struct {
	dataProductStorage service.DataProductsStorage
	bigQueryStorage    service.BigQueryStorage
	bigQueryAPI        service.BigQueryAPI
	naisConsoleStorage service.NaisConsoleStorage
	allUsersGroup      string
}

func (s *dataProductsService) GetDataset(ctx context.Context, id uuid.UUID) (*service.Dataset, error) {
	const op errs.Op = "dataProductsService.GetDataset"

	ds, err := s.dataProductStorage.GetDataset(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return ds, nil
}

func (s *dataProductsService) GetDataproduct(ctx context.Context, id uuid.UUID) (*service.DataproductWithDataset, error) {
	const op errs.Op = "dataProductsService.GetDataproduct"

	dp, err := s.dataProductStorage.GetDataproduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return dp, nil
}

func (s *dataProductsService) GetAccessiblePseudoDatasetsForUser(ctx context.Context, user *service.User) ([]*service.PseudoDataset, error) {
	const op errs.Op = "dataProductsService.GetAccessiblePseudoDatasetsForUser"

	subjectsAsOwner := []string{user.Email}
	subjectsAsOwner = append(subjectsAsOwner, user.GoogleGroups.Emails()...)
	subjectsAsAccesser := []string{"user:" + user.Email}

	for _, geml := range user.GoogleGroups.Emails() {
		subjectsAsAccesser = append(subjectsAsAccesser, "group:"+geml)
	}

	pseudoDatasets, err := s.dataProductStorage.GetAccessiblePseudoDatasourcesByUser(ctx, subjectsAsOwner, subjectsAsAccesser)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return pseudoDatasets, nil
}

func (s *dataProductsService) GetDatasetsMinimal(ctx context.Context) ([]*service.DatasetMinimal, error) {
	const op errs.Op = "dataProductsService.GetDatasetsMinimal"

	datasets, err := s.dataProductStorage.GetDatasetsMinimal(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return datasets, nil
}

func (s *dataProductsService) CreateDataproduct(ctx context.Context, user *service.User, input service.NewDataproduct) (*service.DataproductMinimal, error) {
	const op errs.Op = "dataProductsService.CreateDataproduct"

	if err := ensureUserInGroup(user, input.Group); err != nil {
		return nil, errs.E(op, err)
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	dataproduct, err := s.dataProductStorage.CreateDataproduct(ctx, input)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return dataproduct, nil
}

func (s *dataProductsService) UpdateDataproduct(ctx context.Context, user *service.User, id uuid.UUID, input service.UpdateDataproductDto) (*service.DataproductMinimal, error) {
	const op errs.Op = "dataProductsService.UpdateDataproduct"

	dp, err := s.dataProductStorage.GetDataproduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if err := ensureUserInGroup(user, dp.Owner.Group); err != nil {
		return nil, errs.E(op, err)
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	dataproduct, err := s.dataProductStorage.UpdateDataproduct(ctx, id, input)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return dataproduct, nil
}

func (s *dataProductsService) DeleteDataproduct(ctx context.Context, user *service.User, id uuid.UUID) (*service.DataproductWithDataset, error) {
	const op errs.Op = "dataProductsService.DeleteDataproduct"

	dp, err := s.dataProductStorage.GetDataproduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if err := ensureUserInGroup(user, dp.Owner.Group); err != nil {
		return nil, errs.E(op, err)
	}

	err = s.dataProductStorage.DeleteDataproduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return dp, nil
}

func (s *dataProductsService) CreateDataset(ctx context.Context, user *service.User, input service.NewDataset) (*service.Dataset, error) {
	const op errs.Op = "dataProductsService.CreateDataset"

	dp, err := s.dataProductStorage.GetDataproduct(ctx, input.DataproductID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if err := ensureUserInGroup(user, dp.Owner.Group); err != nil {
		return nil, errs.E(op, err)
	}

	var referenceDatasource *service.NewBigQuery
	var pseudoBigQuery *service.NewBigQuery
	if len(input.PseudoColumns) > 0 {
		projectID, datasetID, tableID, err := s.bigQueryAPI.CreatePseudonymisedView(ctx, input.BigQuery.ProjectID,
			input.BigQuery.Dataset, input.BigQuery.Table, input.PseudoColumns)
		if err != nil {
			return nil, errs.E(op, err)
		}

		referenceDatasource = &input.BigQuery

		pseudoBigQuery = &service.NewBigQuery{
			ProjectID: projectID,
			Dataset:   datasetID,
			Table:     tableID,
			PiiTags:   input.BigQuery.PiiTags,
		}
	}

	updatedInput, err := s.prepareBigQueryHandlePseudoView(ctx, input, pseudoBigQuery, dp.Owner.Group)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if updatedInput.Description != nil && *updatedInput.Description != "" {
		*updatedInput.Description = html.EscapeString(*updatedInput.Description)
	}

	ds, err := s.dataProductStorage.CreateDataset(ctx, updatedInput, referenceDatasource, user)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if pseudoBigQuery == nil && updatedInput.GrantAllUsers != nil && *updatedInput.GrantAllUsers {
		if err := s.bigQueryAPI.Grant(ctx, updatedInput.BigQuery.ProjectID, updatedInput.BigQuery.Dataset, updatedInput.BigQuery.Table, s.allUsersGroup); err != nil {
			return nil, errs.E(op, err)
		}
	}

	return ds, nil
}

func (s *dataProductsService) prepareBigQueryHandlePseudoView(ctx context.Context, ds service.NewDataset, viewBQ *service.NewBigQuery, group string) (service.NewDataset, error) {
	const op errs.Op = "dataProductsService.prepareBigQueryHandlePseudoView"

	if err := s.ensureGroupOwnsGCPProject(ctx, group, ds.BigQuery.ProjectID); err != nil {
		return service.NewDataset{}, errs.E(op, err)
	}

	if viewBQ != nil {
		metadata, err := s.prepareBigQuery(ctx, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, viewBQ.ProjectID, viewBQ.Dataset, viewBQ.Table)
		if err != nil {
			return service.NewDataset{}, errs.E(op, err)
		}
		ds.BigQuery = *viewBQ
		ds.Metadata = *metadata
		return ds, nil
	}

	metadata, err := s.prepareBigQuery(ctx, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, ds.BigQuery.Table)
	if err != nil {
		return service.NewDataset{}, errs.E(op, err)
	}
	ds.Metadata = *metadata

	return ds, nil
}

func (s *dataProductsService) ensureGroupOwnsGCPProject(ctx context.Context, group, projectID string) error {
	const op errs.Op = "dataProductsService.ensureGroupOwnsGCPProject"

	groupProject, err := s.naisConsoleStorage.GetTeamProject(ctx, auth.TrimNaisTeamPrefix(group))
	if err != nil {
		return errs.E(errs.Unauthorized, op, fmt.Errorf("group %s does not own the GCP project %s", group, projectID))
	}

	if groupProject == projectID {
		return nil
	}

	return errs.E(errs.Unauthorized, op, fmt.Errorf("group %s does not own the GCP project %s", group, projectID))
}

func (s *dataProductsService) prepareBigQuery(ctx context.Context, srcProject, srcDataset, sinkProject, sinkDataset, sinkTable string) (*service.BigqueryMetadata, error) {
	const op errs.Op = "dataProductsService.prepareBigQuery"

	metadata, err := s.bigQueryAPI.TableMetadata(ctx, sinkProject, sinkDataset, sinkTable)
	if err != nil {
		return nil, errs.E(op, err)
	}

	switch metadata.TableType {
	case service.RegularTable:
	case service.ViewTable:
		fallthrough
	case service.MaterializedView:
		if err := s.bigQueryAPI.AddToAuthorizedViews(ctx, srcProject, srcDataset, sinkProject, sinkDataset, sinkTable); err != nil {
			return nil, errs.E(op, err)
		}
	default:
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("prepareBigQuery: unsupported table type %v", metadata.TableType))
	}

	return &metadata, nil
}

func (s *dataProductsService) DeleteDataset(ctx context.Context, user *service.User, id uuid.UUID) (string, error) {
	const op errs.Op = "dataProductsService.DeleteDataset"

	ds, err := s.dataProductStorage.GetDataset(ctx, id)
	if err != nil {
		return "", errs.E(op, err)
	}

	dp, err := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return "", errs.E(op, err)
	}

	if err := ensureUserInGroup(user, dp.Owner.Group); err != nil {
		return "", errs.E(op, err)
	}

	err = s.dataProductStorage.DeleteDataset(ctx, id)
	if err != nil {
		return "", errs.E(op, err)
	}

	return dp.ID.String(), nil
}

func (s *dataProductsService) UpdateDataset(ctx context.Context, user *service.User, id uuid.UUID, input service.UpdateDatasetDto) (string, error) {
	const op errs.Op = "dataProductsService.UpdateDataset"

	ds, err := s.dataProductStorage.GetDataset(ctx, id)
	if err != nil {
		return "", errs.E(op, err)
	}

	if input.DataproductID == nil {
		input.DataproductID = &ds.DataproductID
	}

	dp, err := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return "", errs.E(op, err)
	}

	if err := ensureUserInGroup(user, dp.Owner.Group); err != nil {
		return "", errs.E(op, err)
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	if input.Keywords == nil {
		input.Keywords = []string{}
	}

	if *input.DataproductID != ds.DataproductID {
		dp2, err := s.dataProductStorage.GetDataproduct(ctx, *input.DataproductID)
		if err != nil {
			return "", errs.E(op, err)
		}

		if err := ensureUserInGroup(user, dp2.Owner.Group); err != nil {
			return "", errs.E(op, err)
		}

		if dp.Owner.Group != dp2.Owner.Group {
			return "", errs.E(errs.InvalidRequest, op, fmt.Errorf("updateDataset: cannot move dataset between dataproducts owned by different groups"))
		}
	}

	if len(input.PseudoColumns) > 0 {
		referenceDatasource, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, id, true)
		if err != nil {
			return "", errs.E(op, err)
		}

		_, _, _, err = s.bigQueryAPI.CreatePseudonymisedView(ctx, referenceDatasource.ProjectID,
			referenceDatasource.Dataset, referenceDatasource.Table, input.PseudoColumns)
		if err != nil {
			return "", errs.E(op, err)
		}
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	updatedID, err := s.dataProductStorage.UpdateDataset(ctx, id, input)
	if err != nil {
		return "", errs.E(op, err)
	}

	err = s.bigQueryStorage.UpdateBigqueryDatasource(ctx, service.BigQueryDataSourceUpdate{
		PiiTags:       input.PiiTags,
		PseudoColumns: input.PseudoColumns,
		DatasetID:     id,
	})
	if err != nil {
		return "", errs.E(op, err)
	}

	return updatedID, nil
}

func NewDataProductsService(
	dataProductStorage service.DataProductsStorage,
	bigQueryStorage service.BigQueryStorage,
	bigQueryAPI service.BigQueryAPI,
	naisConsoleStorage service.NaisConsoleStorage,
	allUsersGroup string,
) *dataProductsService {
	return &dataProductsService{
		dataProductStorage: dataProductStorage,
		bigQueryStorage:    bigQueryStorage,
		bigQueryAPI:        bigQueryAPI,
		naisConsoleStorage: naisConsoleStorage,
		allUsersGroup:      allUsersGroup,
	}
}
