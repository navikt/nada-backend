package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
	log "github.com/sirupsen/logrus"
	"github.com/sqlc-dev/pqtype"
	"net/url"
	"os"
)

var _ service.DataProductsStorage = &dataProductPostgres{}

type dataProductPostgres struct {
	db *database.Repo
}

func (p *dataProductPostgres) GetAccessiblePseudoDatasourcesByUser(ctx context.Context, subjectsAsOwner []string, subjectsAsAccesser []string) ([]*service.PseudoDataset, error) {
	rows, err := p.db.Querier.GetAccessiblePseudoDatasetsByUser(ctx, gensql.GetAccessiblePseudoDatasetsByUserParams{
		OwnerSubjects:  subjectsAsOwner,
		AccessSubjects: subjectsAsAccesser,
	})
	if err != nil {
		return nil, err
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

func (p *dataProductPostgres) GetDatasetsMinimal(ctx context.Context) ([]*service.DatasetMinimal, error) {
	sqldss, err := p.db.Querier.GetAllDatasetsMinimal(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting all datasets minimal: %w", err)
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

func (p *dataProductPostgres) UpdateDataset(ctx context.Context, id string, input service.UpdateDatasetDto) (string, error) {
	if input.Keywords == nil {
		input.Keywords = []string{}
	}

	res, err := p.db.Querier.UpdateDataset(ctx, gensql.UpdateDatasetParams{
		Name:                     input.Name,
		Description:              ptrToNullString(input.Description),
		ID:                       uuid.MustParse(id),
		Pii:                      gensql.PiiLevel(input.Pii),
		Slug:                     slugify(input.Slug, input.Name),
		Repo:                     ptrToNullString(input.Repo),
		Keywords:                 input.Keywords,
		DataproductID:            *input.DataproductID,
		AnonymisationDescription: ptrToNullString(input.AnonymisationDescription),
		TargetUser:               ptrToNullString(input.TargetUser),
	})
	if err != nil {
		return "", fmt.Errorf("updating dataset in database: %w", err)
	}

	// TODO: tags table should be removed
	for _, keyword := range input.Keywords {
		err = p.db.Querier.CreateTagIfNotExist(ctx, keyword)
		if err != nil {
			return "", fmt.Errorf("creating tag: %w", err)
		}
	}

	if !json.Valid([]byte(*input.PiiTags)) {
		return "", fmt.Errorf("invalid pii tags, must be json map or null: %w", err)
	}

	return res.ID.String(), nil
}

func (p *dataProductPostgres) CreateDataset(ctx context.Context, ds service.NewDataset, referenceDatasource *service.NewBigQuery, user *auth.User) (*string, error) {
	tx, err := p.db.GetDB().Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if ds.Keywords == nil {
		ds.Keywords = []string{}
	}

	querier := p.db.Querier.WithTx(tx)

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
		return nil, err
	}

	schemaJSON, err := json.Marshal(ds.Metadata.Schema.Columns)
	if err != nil {
		return nil, fmt.Errorf("marshalling schema: %w", err)
	}

	if ds.BigQuery.PiiTags != nil && !json.Valid([]byte(*ds.BigQuery.PiiTags)) {
		return nil, fmt.Errorf("invalid pii tags, must be json map or null: %w", err)
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
		return nil, err
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
			return nil, err
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
			return nil, err
		}
	}

	for _, keyword := range ds.Keywords {
		err = querier.CreateTagIfNotExist(ctx, keyword)
		if err != nil {
			log.WithError(err).Warn("failed to create tag when creating dataset in database")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &created.Slug, nil
}

func (p *dataProductPostgres) DeleteDataproduct(ctx context.Context, id string) error {
	if err := p.db.Querier.DeleteDataproduct(ctx, uuid.MustParse(id)); err != nil {
		return fmt.Errorf("deleting dataproduct: %w", err)
	}

	return nil
}

func (p *dataProductPostgres) UpdateDataproduct(ctx context.Context, id string, input service.UpdateDataproductDto) (*service.DataproductMinimal, error) {
	res, err := p.db.Querier.UpdateDataproduct(ctx, gensql.UpdateDataproductParams{
		Name:                  input.Name,
		Description:           ptrToNullString(input.Description),
		ID:                    uuid.MustParse(id),
		OwnerTeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamContact:           ptrToNullString(input.TeamContact),
		Slug:                  slugify(input.Slug, input.Name),
		TeamID:                ptrToNullString(input.TeamID),
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct: %w", err)
	}

	return dataproductMinimalFromSQL(&res), nil
}

func (p *dataProductPostgres) CreateDataproduct(ctx context.Context, input service.NewDataproduct) (*service.DataproductMinimal, error) {
	dataproduct, err := p.db.Querier.CreateDataproduct(ctx, gensql.CreateDataproductParams{
		Name:                  input.Name,
		Description:           ptrToNullString(input.Description),
		OwnerGroup:            input.Group,
		OwnerTeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		Slug:                  slugify(input.Slug, input.Name),
		TeamContact:           ptrToNullString(input.TeamContact),
		TeamID:                ptrToNullString(input.TeamID),
	})
	if err != nil {
		return nil, fmt.Errorf("creating dataproduct: %w", err)
	}

	return dataproductMinimalFromSQL(&dataproduct), nil
}

func (p *dataProductPostgres) SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error {
	return p.db.Querier.SetDatasourceDeleted(ctx, id)
}

func (p *dataProductPostgres) GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error) {
	return p.db.Querier.GetOwnerGroupOfDataset(ctx, datasetID)
}

func (p *dataProductPostgres) DeleteDataset(ctx context.Context, id uuid.UUID) error {
	return p.db.Querier.DeleteDataset(ctx, id)
}

func (p *dataProductPostgres) GetDataset(ctx context.Context, id string) (*service.Dataset, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing dataset id: %w", err)
	}

	sqlds, err := p.db.Querier.GetDatasetComplete(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}

	ds, apiErr := datasetFromSQL(sqlds)
	if err != nil {
		return nil, apiErr
	}

	return ds, nil
}

func datasetFromSQL(dsrows []gensql.DatasetView) (*service.Dataset, error) {
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
					return nil, fmt.Errorf("unmarshalling schema: %w", err)
				}
			}

			dsrc := &service.BigQuery{
				ID:            dsrow.BqID,
				DatasetID:     dsrow.DsID,
				ProjectID:     dsrow.BqProject,
				Dataset:       dsrow.BqDataset,
				Table:         dsrow.BqTableName,
				TableType:     service.BigQueryType(dsrow.BqTableType),
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
					AccessRequestID: nullUUIDToUUIDPtr(dsrow.AccessRequestID),
				}
				dataset.Access = append(dataset.Access, access)
			}
		}

		// FIXME: these should all be configured during startup and injected
		if dataset.MetabaseUrl == nil && dsrow.MbDatabaseID.Valid {
			base := "https://metabase.intern.dev.nav.no/browse/databases/%v"
			if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
				base = "https://metabase.intern.nav.no/browse/databases/%v"
			}
			url := fmt.Sprintf(base, dsrow.MbDatabaseID.Int32)
			dataset.MetabaseUrl = &url
		}
	}

	return dataset, nil
}

func (p *dataProductPostgres) GetDataproducts(ctx context.Context, ids []uuid.UUID) ([]service.DataproductWithDataset, error) {
	dp, err := p.db.Querier.GetDataproductsWithDatasets(ctx, gensql.GetDataproductsWithDatasetsParams{
		Ids:    ids,
		Groups: []string{},
	})
	if err != nil {
		return nil, fmt.Errorf("getting dataproducts: %w", err)
	}

	return dataproductsWithDatasetFromSQL(dp), nil
}

func (p *dataProductPostgres) GetDataproduct(ctx context.Context, id string) (*service.DataproductWithDataset, error) {
	dpuuid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing dataproduct id: %w", err)
	}

	dps, err := p.GetDataproducts(ctx, []uuid.UUID{dpuuid})
	if err != nil {
		return nil, fmt.Errorf("getting dataproduct: %w", err)
	}

	// it is safe to directly use the first element without checking the length
	// because if the length was 0, the sql query in GetDataproducts should have returned no row
	return &dps[0], nil
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
					TeamID:           nullStringToPtr(dprow.TeamID),
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
		keywords := []string{}
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
			TeamID:           &dp.TeamID.String,
		},
	}
}

func slugify(maybeslug *string, fallback string) string {
	if maybeslug != nil {
		return *maybeslug
	}
	// TODO(thokra): Smartify this?
	return url.PathEscape(fallback)
}

func ptrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}

	return &ns.String
}

func nullUUIDToUUIDPtr(nu uuid.NullUUID) *uuid.UUID {
	if !nu.Valid {
		return nil
	}
	return &nu.UUID
}

func NewDataProductStorage(db *database.Repo) *dataProductPostgres {
	return &dataProductPostgres{
		db: db,
	}
}
