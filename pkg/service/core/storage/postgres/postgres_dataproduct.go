package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"
	"github.com/sqlc-dev/pqtype"
)

var _ service.DataProductsStorage = &dataProductStorage{}

type dataProductStorage struct {
	databasesBaseURL string
	db               *database.Repo
	log              zerolog.Logger
}

func (s *dataProductStorage) GetDataproductKeywords(ctx context.Context, dpid uuid.UUID) ([]string, error) {
	const op errs.Op = "dataProductStorage.GetDataproductKeywords"

	keywords, err := s.db.Querier.GetDataproductKeywords(ctx, dpid)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	return keywords, nil
}

func (s *dataProductStorage) GetDataproductsByTeamID(ctx context.Context, teamIDs []uuid.UUID) ([]*service.Dataproduct, error) {
	const op errs.Op = "dataProductStorage.GetDataproductsByTeamID"

	raw, err := s.db.Querier.GetDataproductsByProductArea(ctx, teamIDs)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	dps := make([]*service.Dataproduct, len(raw))
	for idx, dp := range raw {
		dps[idx] = dataproductFromSQL(&dp)

		keywords, err := s.GetDataproductKeywords(ctx, dps[idx].ID)
		if err != nil {
			return nil, errs.E(op, err)
		}

		if keywords == nil {
			keywords = []string{}
		}

		dps[idx].Keywords = keywords
	}

	return dps, nil
}

func (s *dataProductStorage) GetDataproductsNumberByTeam(ctx context.Context, teamID uuid.UUID) (int64, error) {
	const op errs.Op = "dataProductStorage.GetDataproductsNumberByTeam"

	n, err := s.db.Querier.GetDataproductsNumberByTeam(ctx, uuidToNullUUID(teamID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, errs.E(errs.Database, op, err)
	}

	return n, nil
}

func (s *dataProductStorage) GetAccessibleDatasets(ctx context.Context, userGroups []string, requester string) (owned []*service.AccessibleDataset, granted []*service.AccessibleDataset, serviceAccountGranted []*service.AccessibleDataset, err error) {
	const op errs.Op = "dataProductStorage.GetAccessibleDatasets"

	datasetsSQL, err := s.db.Querier.GetAccessibleDatasets(ctx, gensql.GetAccessibleDatasetsParams{
		Groups:    userGroups,
		Requester: requester,
	})
	if err != nil {
		return nil, nil, nil, errs.E(errs.Database, op, err)
	}

	for _, d := range datasetsSQL {
		if matchAny(nullStringToString(d.Group), userGroups) {
			owned = append(owned, accessibleDatasetFromSql(&d))
		} else {
			granted = append(granted, accessibleDatasetFromSql(&d))
		}
	}

	serviceAccountAccessible, err := s.db.Querier.GetAccessibleDatasetsByOwnedServiceAccounts(ctx, gensql.GetAccessibleDatasetsByOwnedServiceAccountsParams{
		Requester: requester,
		Groups:    userGroups,
	})
	if err != nil {
		return nil, nil, nil, errs.E(errs.Database, op, err)
	}

	for _, d := range serviceAccountAccessible {
		serviceAccountGranted = append(serviceAccountGranted, &service.AccessibleDataset{
			Dataset: service.Dataset{
				ID:            d.ID,
				Name:          d.Name,
				DataproductID: d.DataproductID,
				Keywords:      d.Keywords,
				Slug:          d.Slug,
				Description:   nullStringToPtr(d.Description),
				Created:       d.Created,
				LastModified:  d.LastModified,
			},
			Group:           nullStringToString(d.Group),
			DpSlug:          *nullStringToPtr(d.DpSlug),
			DataproductName: nullStringToString(d.DpName),
			Subject:         nullStringToPtr(d.Subject),
		})
	}

	return
}

func accessibleDatasetFromSql(d *gensql.GetAccessibleDatasetsRow) *service.AccessibleDataset {
	return &service.AccessibleDataset{
		Dataset: service.Dataset{
			ID:            d.ID,
			Name:          d.Name,
			DataproductID: d.DataproductID,
			Keywords:      d.Keywords,
			Slug:          d.Slug,
			Description:   nullStringToPtr(d.Description),
			Created:       d.Created,
			LastModified:  d.LastModified,
		},
		Group:           nullStringToString(d.Group),
		DpSlug:          *nullStringToPtr(d.DpSlug),
		DataproductName: nullStringToString(d.DpName),
		Subject:         nullStringToPtr(d.Subject),
	}
}

func (s *dataProductStorage) GetDataproductsWithDatasetsAndAccessRequests(ctx context.Context, ids []uuid.UUID, groups []string) ([]service.DataproductWithDataset, []service.AccessRequestForGranter, error) {
	const op errs.Op = "dataProductStorage.GetDataproductsWithDatasetsAndAccessRequests"

	dpres, err := s.db.Querier.GetDataproductsWithDatasetsAndAccessRequests(ctx, gensql.GetDataproductsWithDatasetsAndAccessRequestsParams{
		Ids:    ids,
		Groups: groups,
	})
	if err != nil {
		return nil, nil, errs.E(errs.Database, op, err)
	}

	dataproducts, accessRequests, err := dataproductsWithDatasetAndAccessRequestsForGranterFromSQL(dpres)
	if err != nil {
		return nil, nil, errs.E(errs.Internal, op, err)
	}

	return dataproducts, accessRequests, nil
}

func dataproductsWithDatasetAndAccessRequestsForGranterFromSQL(dprrows []gensql.GetDataproductsWithDatasetsAndAccessRequestsRow) ([]service.DataproductWithDataset, []service.AccessRequestForGranter, error) {
	const op errs.Op = "dataProductStorage.dataproductsWithDatasetAndAccessRequestsForGranterFromSQL"

	if dprrows == nil {
		return nil, nil, nil
	}

	dprows := make([]gensql.GetDataproductsWithDatasetsRow, len(dprrows))
	for i, dprrow := range dprrows {
		dprows[i] = gensql.GetDataproductsWithDatasetsRow{
			DpID:             dprrow.DpID,
			DpName:           dprrow.DpName,
			DpCreated:        dprrow.DpCreated,
			DpLastModified:   dprrow.DpLastModified,
			DpDescription:    dprrow.DpDescription,
			DpSlug:           dprrow.DpSlug,
			DpGroup:          dprrow.DpGroup,
			TeamkatalogenUrl: dprrow.TeamkatalogenUrl,
			TeamContact:      dprrow.TeamContact,
			TeamID:           dprrow.TeamID,
		}
	}
	dp := dataproductsWithDatasetFromSQL(dprows)

	arrows := make([]gensql.DatasetAccessRequest, 0)

	for _, dprrow := range dprrows {
		if dprrow.DarID.Valid {
			arrows = append(arrows, gensql.DatasetAccessRequest{
				ID:                   dprrow.DarID.UUID,
				DatasetID:            dprrow.DarDatasetID.UUID,
				Subject:              dprrow.DarSubject.String,
				Created:              dprrow.DarCreated.Time,
				Status:               dprrow.DarStatus.AccessRequestStatusType,
				Closed:               dprrow.DarClosed,
				Expires:              dprrow.DarExpires,
				Granter:              dprrow.DarGranter,
				Owner:                dprrow.DarOwner.String,
				PollyDocumentationID: dprrow.DarPollyDocumentationID,
				Reason:               dprrow.DarReason,
			})
		}
	}
	ars, err := From(DatasetAccessRequests(arrows))
	if err != nil {
		return nil, nil, errs.E(errs.Internal, op, err)
	}

	arfg := make([]service.AccessRequestForGranter, len(ars))
	for i, ar := range ars {
		dataproductID := uuid.Nil
		datasetName := ""
		dataproductName := ""
		dataproductSlug := ""
		for _, dprrow := range dprrows {
			if dprrow.DarDatasetID.UUID == ar.DatasetID {
				dataproductID = dprrow.DpID
				datasetName = dprrow.DsName.String
				dataproductName = dprrow.DpName
				dataproductSlug = dprrow.DpSlug
				break
			}
		}

		arfg[i] = service.AccessRequestForGranter{
			AccessRequest:   *ar,
			DatasetName:     datasetName,
			DataproductName: dataproductName,
			DataproductID:   dataproductID,
			DataproductSlug: dataproductSlug,
		}
	}

	return dp, arfg, nil
}

func (s *dataProductStorage) GetAccessiblePseudoDatasourcesByUser(ctx context.Context, subjectsAsOwner []string, subjectsAsAccesser []string) ([]*service.PseudoDataset, error) {
	const op errs.Op = "dataProductStorage.GetAccessiblePseudoDatasourcesByUser"

	rows, err := s.db.Querier.GetAccessiblePseudoDatasetsByUser(ctx, gensql.GetAccessiblePseudoDatasetsByUserParams{
		OwnerSubjects:  subjectsAsOwner,
		AccessSubjects: subjectsAsAccesser,
	})
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	var pseudoDatasets []*service.PseudoDataset
	bqIDMap := make(map[string]int)
	for _, d := range rows {
		pseudoDataset := &service.PseudoDataset{
			// name is the name of the dataset
			Name: d.Name,
			// datasetID is the id of the dataset
			DatasetID: d.DatasetID,
			// datasourceID is the id of the bigquery datasource
			DatasourceID: d.BqDatasourceID,
		}
		bqID := fmt.Sprintf("%v.%v.%v", d.BqProjectID, d.BqDatasetID, d.BqTableID)

		_, exist := bqIDMap[bqID]
		if exist {
			continue
		}

		bqIDMap[bqID] = 1
		pseudoDatasets = append(pseudoDatasets, pseudoDataset)
	}

	return pseudoDatasets, nil
}

func (s *dataProductStorage) GetDatasetsMinimal(ctx context.Context) ([]*service.DatasetMinimal, error) {
	const op errs.Op = "dataProductStorage.GetDatasetsMinimal"

	sqldss, err := s.db.Querier.GetAllDatasetsMinimal(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	dss := make([]*service.DatasetMinimal, len(sqldss))
	for i, ds := range sqldss {
		dss[i] = &service.DatasetMinimal{
			ID:              ds.ID,
			Name:            ds.Name,
			Created:         ds.Created,
			BigQueryProject: ds.ProjectID,
			BigQueryDataset: ds.Dataset,
			BigQueryTable:   ds.TableName,
		}
	}

	return dss, nil
}

func (s *dataProductStorage) UpdateDataset(ctx context.Context, id uuid.UUID, input service.UpdateDatasetDto) (string, error) {
	const op errs.Op = "dataProductStorage.UpdateDataset"

	if input.Keywords == nil {
		input.Keywords = []string{}
	}

	res, err := s.db.Querier.UpdateDataset(ctx, gensql.UpdateDatasetParams{
		Name:                     input.Name,
		Description:              ptrToNullString(input.Description),
		ID:                       id,
		Pii:                      gensql.PiiLevel(input.Pii),
		Slug:                     slugify(input.Slug, input.Name),
		Repo:                     ptrToNullString(input.Repo),
		Keywords:                 input.Keywords,
		DataproductID:            *input.DataproductID,
		AnonymisationDescription: ptrToNullString(input.AnonymisationDescription),
		TargetUser:               ptrToNullString(input.TargetUser),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errs.E(errs.NotExist, op, err)
		}

		return "", errs.E(errs.Database, op, err)
	}

	// TODO: tags table should be removed
	for _, keyword := range input.Keywords {
		err = s.db.Querier.CreateTagIfNotExist(ctx, keyword)
		if err != nil {
			return "", errs.E(errs.Database, op, err)
		}
	}

	if !json.Valid([]byte(*input.PiiTags)) {
		return "", errs.E(errs.InvalidRequest, op, err, errs.Parameter("pii_tags"))
	}

	return res.ID.String(), nil
}

func (s *dataProductStorage) CreateDataset(ctx context.Context, ds service.NewDataset, referenceDatasource *service.NewBigQuery, user *service.User) (*service.Dataset, error) {
	const op errs.Op = "dataProductStorage.CreateDataset"

	tx, err := s.db.GetDB().Begin()
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}
	defer tx.Rollback()

	if ds.Keywords == nil {
		ds.Keywords = []string{}
	}

	querier := s.db.Querier.WithTx(tx)

	created, err := querier.CreateDataset(ctx, gensql.CreateDatasetParams{
		Name:                     ds.Name,
		DataproductID:            ds.DataproductID,
		Description:              ptrToNullString(ds.Description),
		Pii:                      gensql.PiiLevel(ds.Pii),
		Type:                     "bigquery",
		Slug:                     slugify(ds.Slug, ds.Name),
		Repo:                     ptrToNullString(ds.Repo),
		Keywords:                 ds.Keywords,
		AnonymisationDescription: ptrToNullString(ds.AnonymisationDescription),
		TargetUser:               ptrToNullString(ds.TargetUser),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err)
		}

		return nil, errs.E(errs.Database, op, err)
	}

	schemaJSON, err := json.Marshal(ds.Metadata.Schema.Columns)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err, errs.Parameter("schema_columns"))
	}

	if ds.BigQuery.PiiTags != nil && !json.Valid([]byte(*ds.BigQuery.PiiTags)) {
		return nil, errs.E(errs.InvalidRequest, op, err, errs.Parameter("pii_tags"))
	}

	_, err = querier.CreateBigqueryDatasource(ctx, gensql.CreateBigqueryDatasourceParams{
		DatasetID:    created.ID,
		ProjectID:    ds.BigQuery.ProjectID,
		Dataset:      ds.BigQuery.Dataset,
		TableName:    ds.BigQuery.Table,
		Schema:       pqtype.NullRawMessage{RawMessage: schemaJSON, Valid: len(schemaJSON) > 4},
		LastModified: ds.Metadata.LastModified,
		Created:      ds.Metadata.Created,
		Expires:      sql.NullTime{Time: ds.Metadata.Expires, Valid: !ds.Metadata.Expires.IsZero()},
		TableType:    string(ds.Metadata.TableType),
		PiiTags: pqtype.NullRawMessage{
			RawMessage: json.RawMessage([]byte(ptrToString(ds.BigQuery.PiiTags))),
			Valid:      len(ptrToString(ds.BigQuery.PiiTags)) > 4,
		},
		PseudoColumns: ds.PseudoColumns,
		IsReference:   false,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err)
		}

		return nil, errs.E(errs.Database, op, err)
	}

	if len(ds.PseudoColumns) > 0 && referenceDatasource != nil {
		_, err = querier.CreateBigqueryDatasource(ctx, gensql.CreateBigqueryDatasourceParams{
			DatasetID:    created.ID,
			ProjectID:    referenceDatasource.ProjectID,
			Dataset:      referenceDatasource.Dataset,
			TableName:    referenceDatasource.Table,
			Schema:       pqtype.NullRawMessage{RawMessage: schemaJSON, Valid: len(schemaJSON) > 4},
			LastModified: ds.Metadata.LastModified,
			Created:      ds.Metadata.Created,
			Expires:      sql.NullTime{Time: ds.Metadata.Expires, Valid: !ds.Metadata.Expires.IsZero()},
			TableType:    string(ds.Metadata.TableType),
			PiiTags: pqtype.NullRawMessage{
				RawMessage: json.RawMessage([]byte(ptrToString(ds.BigQuery.PiiTags))),
				Valid:      len(ptrToString(ds.BigQuery.PiiTags)) > 4,
			},
			PseudoColumns: ds.PseudoColumns,
			IsReference:   true,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errs.E(errs.NotExist, op, err)
			}

			return nil, errs.E(errs.Database, op, err)
		}
	}

	if ds.GrantAllUsers != nil && *ds.GrantAllUsers {
		_, err = querier.GrantAccessToDataset(ctx, gensql.GrantAccessToDatasetParams{
			DatasetID: created.ID,
			Expires:   sql.NullTime{},
			Subject:   emailOfSubjectToLower("group:all-users@nav.no"),
			Granter:   user.Email,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errs.E(errs.NotExist, op, err)
			}

			return nil, errs.E(errs.Database, op, err)
		}
	}

	for _, keyword := range ds.Keywords {
		err = querier.CreateTagIfNotExist(ctx, keyword)
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to create tag when creating dataset in database")
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	ret, err := s.GetDataset(ctx, created.ID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return ret, nil
}

func (s *dataProductStorage) DeleteDataproduct(ctx context.Context, id uuid.UUID) error {
	const op errs.Op = "dataProductStorage.DeleteDataproduct"

	err := s.db.Querier.DeleteDataproduct(ctx, id)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *dataProductStorage) UpdateDataproduct(ctx context.Context, id uuid.UUID, input service.UpdateDataproductDto) (*service.DataproductMinimal, error) {
	const op errs.Op = "dataProductStorage.UpdateDataproduct"

	res, err := s.db.Querier.UpdateDataproduct(ctx, gensql.UpdateDataproductParams{
		Name:                  input.Name,
		Description:           ptrToNullString(input.Description),
		ID:                    id,
		OwnerTeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamContact:           ptrToNullString(input.TeamContact),
		Slug:                  slugify(input.Slug, input.Name),
		TeamID:                uuidPtrToNullUUID(input.TeamID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err)
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return dataproductMinimalFromSQL(&res), nil
}

func (s *dataProductStorage) CreateDataproduct(ctx context.Context, input service.NewDataproduct) (*service.DataproductMinimal, error) {
	const op errs.Op = "dataProductStorage.CreateDataproduct"

	dataproduct, err := s.db.Querier.CreateDataproduct(ctx, gensql.CreateDataproductParams{
		Name:                  input.Name,
		Description:           ptrToNullString(input.Description),
		OwnerGroup:            input.Group,
		OwnerTeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		Slug:                  slugify(input.Slug, input.Name),
		TeamContact:           ptrToNullString(input.TeamContact),
		TeamID:                uuidPtrToNullUUID(input.TeamID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err)
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return dataproductMinimalFromSQL(&dataproduct), nil
}

func (s *dataProductStorage) SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error {
	const op errs.Op = "dataProductStorage.SetDatasourceDeleted"

	err := s.db.Querier.SetDatasourceDeleted(ctx, id)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *dataProductStorage) GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error) {
	const op errs.Op = "dataProductStorage.GetOwnerGroupOfDataset"

	owner, err := s.db.Querier.GetOwnerGroupOfDataset(ctx, datasetID)
	if err != nil {
		return "", errs.E(errs.Database, op, err)
	}

	return owner, nil
}

func (s *dataProductStorage) DeleteDataset(ctx context.Context, id uuid.UUID) error {
	const op errs.Op = "dataProductStorage.DeleteDataset"

	err := s.db.Querier.DeleteDataset(ctx, id)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *dataProductStorage) GetDataset(ctx context.Context, id uuid.UUID) (*service.Dataset, error) {
	const op errs.Op = "dataProductStorage.GetDataset"

	rawDataset, err := s.db.Querier.GetDatasetComplete(ctx, id)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	ds, err := s.datasetFromSQL(rawDataset)
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return ds, nil
}

func (s *dataProductStorage) datasetFromSQL(dsrows []gensql.DatasetView) (*service.Dataset, error) {
	const op errs.Op = "dataProductStorage.datasetFromSQL"

	var dataset *service.Dataset

	for _, dsrow := range dsrows {
		piiTags := "{}"
		if dsrow.PiiTags.RawMessage != nil {
			piiTags = string(dsrow.PiiTags.RawMessage)
		}
		if dataset == nil {
			dataset = &service.Dataset{
				ID:                dsrow.DsID,
				Name:              dsrow.DsName,
				Created:           dsrow.DsCreated,
				LastModified:      dsrow.DsLastModified,
				Description:       nullStringToPtr(dsrow.DsDescription),
				Slug:              dsrow.DsSlug,
				Keywords:          dsrow.DsKeywords,
				DataproductID:     dsrow.DsDpID,
				Repo:              nullStringToPtr(dsrow.DsRepo),
				Mappings:          []string{},
				Access:            []*service.Access{},
				Datasource:        nil,
				Pii:               service.PiiLevel(dsrow.Pii),
				MetabaseDeletedAt: nullTimeToPtr(dsrow.MbDeletedAt),
			}
		}

		if dsrow.BqID != uuid.Nil {
			var schema []*service.BigqueryColumn
			if dsrow.BqSchema.Valid {
				if err := json.Unmarshal(dsrow.BqSchema.RawMessage, &schema); err != nil {
					return nil, errs.E(errs.Internal, op, fmt.Errorf("unmarshalling schema: %w", err))
				}
			}

			dsrc := &service.BigQuery{
				ID:            dsrow.BqID,
				DatasetID:     dsrow.DsID,
				ProjectID:     dsrow.BqProject,
				Dataset:       dsrow.BqDataset,
				Table:         dsrow.BqTableName,
				TableType:     service.BigQueryTableType(dsrow.BqTableType),
				Created:       dsrow.BqCreated,
				LastModified:  dsrow.BqLastModified,
				Expires:       nullTimeToPtr(dsrow.BqExpires),
				Description:   dsrow.BqDescription.String,
				PiiTags:       &piiTags,
				MissingSince:  nullTimeToPtr(dsrow.BqMissingSince),
				PseudoColumns: dsrow.PseudoColumns,
				Schema:        schema,
			}
			dataset.Datasource = dsrc
		}

		if len(dsrow.MappingServices) > 0 {
			for _, service := range dsrow.MappingServices {
				exist := false
				for _, mapping := range dataset.Mappings {
					if mapping == service {
						exist = true
						break
					}
				}
				if !exist {
					dataset.Mappings = append(dataset.Mappings, service)
				}
			}
		}

		if dsrow.AccessID.Valid {
			exist := false
			for _, dsAccess := range dataset.Access {
				if dsAccess.ID == dsrow.AccessID.UUID {
					exist = true
					break
				}
			}
			if !exist {
				access := &service.Access{
					ID:              dsrow.AccessID.UUID,
					Subject:         dsrow.AccessSubject.String,
					Granter:         dsrow.AccessGranter.String,
					Expires:         nullTimeToPtr(dsrow.AccessExpires),
					Created:         dsrow.AccessCreated.Time,
					Revoked:         nullTimeToPtr(dsrow.AccessRevoked),
					DatasetID:       dsrow.DsID,
					Owner:           dsrow.AccessOwner.String,
					AccessRequestID: nullUUIDToUUIDPtr(dsrow.AccessRequestID),
				}
				dataset.Access = append(dataset.Access, access)
			}
		}

		if dataset.MetabaseUrl == nil && dsrow.MbDatabaseID.Valid {
			metabaseURL := fmt.Sprintf("%s/%v", s.databasesBaseURL, dsrow.MbDatabaseID.Int32)
			dataset.MetabaseUrl = &metabaseURL
		}
	}

	return dataset, nil
}

func (s *dataProductStorage) GetDataproducts(ctx context.Context, ids []uuid.UUID) ([]service.DataproductWithDataset, error) {
	const op errs.Op = "dataProductStorage.GetDataproducts"

	dp, err := s.db.Querier.GetDataproductsWithDatasets(ctx, gensql.GetDataproductsWithDatasetsParams{
		Ids:    ids,
		Groups: []string{},
	})
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	products, err := dataproductsWithDatasetFromSQL(dp), nil
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return products, nil
}

func (s *dataProductStorage) GetDataproduct(ctx context.Context, id uuid.UUID) (*service.DataproductWithDataset, error) {
	const op errs.Op = "dataProductStorage.GetDataproduct"

	dps, err := s.GetDataproducts(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	if len(dps) == 0 {
		return nil, errs.E(errs.NotExist, op, fmt.Errorf("dataproduct with id %v does not exist", id))
	}

	// it is safe to directly use the first element without checking the length
	// because if the length was 0, the sql query in GetDataproducts should have returned no row
	dataProduct := dps[0]

	if dataProduct.Keywords == nil {
		dataProduct.Keywords = make([]string, 0)
	}

	if dataProduct.Datasets == nil {
		dataProduct.Datasets = make([]*service.DatasetInDataproduct, 0)
	}

	return &dataProduct, nil
}

func dataproductsWithDatasetFromSQL(dprows []gensql.GetDataproductsWithDatasetsRow) []service.DataproductWithDataset {
	if dprows == nil {
		return []service.DataproductWithDataset{}
	}

	datasets := datasetsInDataProductFromSQL(dprows)

	var dataproducts []service.DataproductWithDataset

__loop_rows:
	for _, dprow := range dprows {
		for _, dp := range dataproducts {
			if dp.ID == dprow.DpID {
				continue __loop_rows
			}
		}
		dataproduct := service.DataproductWithDataset{
			Dataproduct: service.Dataproduct{
				ID:           dprow.DpID,
				Name:         dprow.DpName,
				Created:      dprow.DpCreated,
				LastModified: dprow.DpLastModified,
				Description:  nullStringToPtr(dprow.DpDescription),
				Slug:         dprow.DpSlug,
				Owner: &service.DataproductOwner{
					Group:            dprow.DpGroup,
					TeamkatalogenURL: nullStringToPtr(dprow.TeamkatalogenUrl),
					TeamContact:      nullStringToPtr(dprow.TeamContact),
					TeamID:           nullUUIDToUUIDPtr(dprow.TeamID),
					ProductAreaID:    nullUUIDToUUIDPtr(dprow.PaID),
				},
			},
		}

		var dpdatasets []*service.DatasetInDataproduct
		for _, ds := range datasets {
			if ds.DataproductID == dataproduct.ID {
				dpdatasets = append(dpdatasets, ds)
			}
		}

		keywordsMap := make(map[string]bool)
		for _, ds := range dpdatasets {
			for _, k := range ds.Keywords {
				keywordsMap[k] = true
			}
		}
		var keywords []string
		for k := range keywordsMap {
			keywords = append(keywords, k)
		}

		dataproduct.Datasets = dpdatasets
		dataproduct.Keywords = keywords
		dataproducts = append(dataproducts, dataproduct)
	}
	return dataproducts
}

func datasetsInDataProductFromSQL(dsrows []gensql.GetDataproductsWithDatasetsRow) []*service.DatasetInDataproduct {
	var datasets []*service.DatasetInDataproduct

	for _, dsrow := range dsrows {
		if !dsrow.DsID.Valid {
			continue
		}

		var ds *service.DatasetInDataproduct

		for _, dsIn := range datasets {
			if dsIn.ID == dsrow.DsID.UUID {
				ds = dsIn
				break
			}
		}
		if ds == nil {
			ds = &service.DatasetInDataproduct{
				ID:                     dsrow.DsID.UUID,
				Name:                   dsrow.DsName.String,
				Created:                dsrow.DsCreated.Time,
				LastModified:           dsrow.DsLastModified.Time,
				Description:            nullStringToPtr(dsrow.DsDescription),
				Slug:                   dsrow.DsSlug.String,
				Keywords:               dsrow.DsKeywords,
				DataproductID:          dsrow.DpID,
				DataSourceLastModified: dsrow.DsrcLastModified.Time,
			}
			datasets = append(datasets, ds)
		}
	}

	return datasets
}

func dataproductMinimalFromSQL(dp *gensql.Dataproduct) *service.DataproductMinimal {
	return &service.DataproductMinimal{
		ID:           dp.ID,
		Name:         dp.Name,
		Created:      dp.Created,
		LastModified: dp.LastModified,
		Description:  &dp.Description.String,
		Slug:         dp.Slug,
		Owner: &service.DataproductOwner{
			Group:            dp.Group,
			TeamkatalogenURL: &dp.TeamkatalogenUrl.String,
			TeamContact:      &dp.TeamContact.String,
			TeamID:           nullUUIDToUUIDPtr(dp.TeamID),
		},
	}
}

func dataproductFromSQL(dp *gensql.DataproductWithTeamkatalogenView) *service.Dataproduct {
	return &service.Dataproduct{
		ID:          dp.ID,
		Name:        dp.Name,
		Description: &dp.Description.String,
		Owner: &service.DataproductOwner{
			Group:            dp.Group,
			TeamkatalogenURL: nullStringToPtr(dp.TeamkatalogenUrl),
			TeamContact:      nullStringToPtr(dp.TeamContact),
			TeamID:           nullUUIDToUUIDPtr(dp.TeamID),
			ProductAreaID:    nullUUIDToUUIDPtr(dp.PaID),
		},
		Created:         dp.Created,
		LastModified:    dp.LastModified,
		Slug:            dp.Slug,
		TeamName:        nullStringToPtr(dp.TeamName),
		ProductAreaName: nullStringToString(dp.PaName),
	}
}

func NewDataProductStorage(databasesBaseURL string, db *database.Repo, log zerolog.Logger) *dataProductStorage {
	return &dataProductStorage{
		db:               db,
		databasesBaseURL: databasesBaseURL,
		log:              log,
	}
}
