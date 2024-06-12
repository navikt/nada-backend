package core

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"html"
)

var _ service.DataProductsService = &dataProductsService{}

type dataProductsService struct {
	dataProductStorage service.DataProductsStorage
	bigQueryStorage    service.BigQueryStorage
	bigQueryAPI        service.BigQueryAPI
	gcpProjects        *auth.TeamProjectsMapping
}

func (s *dataProductsService) GetAccessiblePseudoDatasetsForUser(ctx context.Context) ([]*service.PseudoDataset, error) {
	user := auth.GetUser(ctx)
	subjectsAsOwner := []string{user.Email}
	subjectsAsOwner = append(subjectsAsOwner, user.GoogleGroups.Emails()...)
	subjectsAsAccesser := []string{"user:" + user.Email}

	for _, geml := range user.GoogleGroups.Emails() {
		subjectsAsAccesser = append(subjectsAsAccesser, "group:"+geml)
	}

	pseudoDatasets, err := s.dataProductStorage.GetAccessiblePseudoDatasourcesByUser(ctx, subjectsAsOwner, subjectsAsAccesser)
	if err != nil {
		return nil, fmt.Errorf("dbGetAccessiblePseudoDatasourcesByUser: %w", err)
	}

	return pseudoDatasets, nil
}

func (s *dataProductsService) GetDatasetsMinimal(ctx context.Context) ([]*service.DatasetMinimal, error) {
	datasets, err := s.dataProductStorage.GetDatasetsMinimal(ctx)
	if err != nil {
		return nil, fmt.Errorf("dbGetDatasetsMinimal: %w", err)
	}

	return datasets, nil
}

func (s *dataProductsService) CreateDataproduct(ctx context.Context, input service.NewDataproduct) (*service.DataproductMinimal, error) {
	if err := ensureUserInGroup(ctx, input.Group); err != nil {
		return nil, fmt.Errorf("ensureUserInGroup: %w", err)
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	dataproduct, err := s.dataProductStorage.CreateDataproduct(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("creating dataproduct: %w", err)
	}

	return dataproduct, nil
}

func (s *dataProductsService) UpdateDataproduct(ctx context.Context, id string, input service.UpdateDataproductDto) (*service.DataproductMinimal, error) {
	dp, apierr := s.dataProductStorage.GetDataproduct(ctx, id)
	if apierr != nil {
		return nil, apierr
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, fmt.Errorf("ensureUserInGroup: %w", err)
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	dataproduct, err := s.dataProductStorage.UpdateDataproduct(ctx, id, input)
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct: %w", err)
	}

	return dataproduct, nil
}

func (s *dataProductsService) DeleteDataproduct(ctx context.Context, id string) (*service.DataproductWithDataset, error) {
	dp, apierr := s.dataProductStorage.GetDataproduct(ctx, id)
	if apierr != nil {
		return nil, apierr
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, fmt.Errorf("ensureUserInGroup: %w", err)
	}

	err := s.dataProductStorage.DeleteDataproduct(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("deleting dataproduct: %w", err)
	}

	return dp, nil
}

func (s *dataProductsService) CreateDataset(ctx context.Context, input service.NewDataset) (*string, error) {
	user := auth.GetUser(ctx)

	dp, apierr := s.dataProductStorage.GetDataproduct(ctx, input.DataproductID.String())
	if apierr != nil {
		return nil, apierr
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, fmt.Errorf("ensureUserInGroup: %w", err)
	}

	var referenceDatasource *service.NewBigQuery
	var pseudoBigQuery *service.NewBigQuery
	if len(input.PseudoColumns) > 0 {
		projectID, datasetID, tableID, err := s.bigQueryAPI.CreatePseudonymisedView(ctx, input.BigQuery.ProjectID,
			input.BigQuery.Dataset, input.BigQuery.Table, input.PseudoColumns)
		if err != nil {
			return nil, fmt.Errorf("createPseudonymisedView: %w", err)
		}

		referenceDatasource = &input.BigQuery

		pseudoBigQuery = &service.NewBigQuery{
			ProjectID: projectID,
			Dataset:   datasetID,
			Table:     tableID,
			PiiTags:   input.BigQuery.PiiTags,
		}
	}

	updatedInput, apierr := s.prepareBigQueryHandlePseudoView(ctx, input, pseudoBigQuery, dp.Owner.Group)
	if apierr != nil {
		return nil, apierr
	}

	if updatedInput.Description != nil && *updatedInput.Description != "" {
		*updatedInput.Description = html.EscapeString(*updatedInput.Description)
	}

	ds, err := s.dataProductStorage.CreateDataset(ctx, updatedInput, referenceDatasource, user)
	if err != nil {
		return nil, fmt.Errorf("dbCreateDataset: %w", err)
	}

	if pseudoBigQuery == nil && updatedInput.GrantAllUsers != nil && *updatedInput.GrantAllUsers {
		if err := s.bigQueryAPI.Grant(ctx, updatedInput.BigQuery.ProjectID, updatedInput.BigQuery.Dataset, updatedInput.BigQuery.Table, "group:all-users@nav.no"); err != nil {
			return nil, fmt.Errorf("grant: %w", err)
		}
	}

	return ds, nil
}

func (s *dataProductsService) prepareBigQueryHandlePseudoView(ctx context.Context, ds service.NewDataset, viewBQ *service.NewBigQuery, group string) (service.NewDataset, error) {
	if err := s.ensureGroupOwnsGCPProject(group, ds.BigQuery.ProjectID); err != nil {
		return service.NewDataset{}, fmt.Errorf("ensureGroupOwnsGCPProject: %w", err)
	}

	if viewBQ != nil {
		metadata, err := s.prepareBigQuery(ctx, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, viewBQ.ProjectID, viewBQ.Dataset, viewBQ.Table)
		if err != nil {
			return service.NewDataset{}, err
		}
		ds.BigQuery = *viewBQ
		ds.Metadata = *metadata
		return ds, nil
	}

	metadata, err := s.prepareBigQuery(ctx, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, ds.BigQuery.Table)
	if err != nil {
		return service.NewDataset{}, err
	}
	ds.Metadata = *metadata

	return ds, nil
}

func (s *dataProductsService) ensureGroupOwnsGCPProject(group, projectID string) error {
	groupProject, ok := s.gcpProjects.Get(auth.TrimNaisTeamPrefix(group))
	if !ok {
		return service.ErrUnauthorized
	}

	if groupProject == projectID {
		return nil
	}

	return service.ErrUnauthorized
}

func (s *dataProductsService) prepareBigQuery(ctx context.Context, srcProject, srcDataset, sinkProject, sinkDataset, sinkTable string) (*service.BigqueryMetadata, error) {
	metadata, err := s.bigQueryAPI.TableMetadata(ctx, sinkProject, sinkDataset, sinkTable)
	if err != nil {
		return nil, fmt.Errorf("getTableMetadata: %w", err)
	}

	switch metadata.TableType {
	case bigquery.RegularTable:
	case bigquery.ViewTable:
		fallthrough
	case bigquery.MaterializedView:
		if err := s.bigQueryAPI.AddToAuthorizedViews(ctx, srcProject, srcDataset, sinkProject, sinkDataset, sinkTable); err != nil {
			return nil, fmt.Errorf("addToAuthorizedViews: %w", err)
		}
	default:
		return nil, fmt.Errorf("prepareBigQuery: unsupported table type %v", metadata.TableType)
	}

	return &metadata, nil
}

func (s *dataProductsService) DeleteDataset(ctx context.Context, id string) (string, error) {
	ds, apierr := s.dataProductStorage.GetDataset(ctx, id)
	if apierr != nil {
		return "", apierr
	}

	dp, apierr := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return "", apierr
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return "", fmt.Errorf("ensureUserInGroup: %w", err)
	}

	err := s.dataProductStorage.DeleteDataset(ctx, uuid.MustParse(id))
	if err != nil {
		return "", fmt.Errorf("deleting dataset: %w", err)
	}

	return dp.ID.String(), nil
}

func (s *dataProductsService) UpdateDataset(ctx context.Context, id string, input service.UpdateDatasetDto) (string, error) {
	ds, apierr := s.dataProductStorage.GetDataset(ctx, id)
	if apierr != nil {
		return "", apierr
	}

	if input.DataproductID == nil {
		input.DataproductID = &ds.DataproductID
	}

	dp, apierr := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return "", apierr
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return "", fmt.Errorf("ensureUserInGroup: %w", err)
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	if input.Keywords == nil {
		input.Keywords = []string{}
	}

	if *input.DataproductID != ds.DataproductID {
		dp2, err := s.dataProductStorage.GetDataproduct(ctx, input.DataproductID.String())
		if err != nil {
			return "", err
		}
		if err := ensureUserInGroup(ctx, dp2.Owner.Group); err != nil {
			return "", fmt.Errorf("ensureUserInGroup: %w", err)
		}
		if dp.Owner.Group != dp2.Owner.Group {
			return "", fmt.Errorf("updateDataset: cannot move dataset between dataproducts owned by different groups")
		}
	}

	if len(input.PseudoColumns) > 0 {
		referenceDatasource, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, uuid.MustParse(id), true)
		if err != nil {
			return "", fmt.Errorf("get reference data source: %w", err)
		}

		_, _, _, err = s.bigQueryAPI.CreatePseudonymisedView(ctx, referenceDatasource.ProjectID,
			referenceDatasource.Dataset, referenceDatasource.Table, input.PseudoColumns)
		if err != nil {
			return "", fmt.Errorf("createPseudonymisedView: %w", err)
		}
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	updatedID, err := s.dataProductStorage.UpdateDataset(ctx, id, input)
	if err != nil {
		return "", fmt.Errorf("dbUpdateDataset: %w", err)
	}

	err = s.bigQueryStorage.UpdateBigqueryDatasource(ctx, service.BigQueryDataSourceUpdate{
		PiiTags:       input.PiiTags,
		PseudoColumns: input.PseudoColumns,
		DatasetID:     uuid.MustParse(id),
	})
	if err != nil {
		return "", fmt.Errorf("updateBigqueryDatasource: %w", err)
	}

	return updatedID, nil
}

func NewDataProductsService(dataProductStorage service.DataProductsStorage) *dataProductsService {
	return &dataProductsService{
		dataProductStorage: dataProductStorage,
	}
}
