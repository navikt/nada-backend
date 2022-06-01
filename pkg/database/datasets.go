package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/tabbed/pqtype"
)

func (r *Repo) GetDataset(ctx context.Context, id uuid.UUID) (*models.Dataset, error) {
	res, err := r.querier.GetDataset(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting dataset from database: %w", err)
	}

	return datasetFromSQL(res), nil
}

func (r *Repo) GetDatasetsInDataproduct(ctx context.Context, id uuid.UUID) ([]*models.Dataset, error) {
	datasetsSQL, err := r.querier.GetDatasetsInDataproduct(ctx, id)
	if err != nil {
		return nil, err
	}

	var datasetsGraph []*models.Dataset
	for _, ds := range datasetsSQL {
		datasetsGraph = append(datasetsGraph, datasetFromSQL(ds))
	}

	return datasetsGraph, nil
}

func (r *Repo) CreateDataset(ctx context.Context, ds models.NewDataset) (*models.Dataset, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	if ds.Keywords == nil {
		ds.Keywords = []string{}
	}

	querier := r.querier.WithTx(tx)
	created, err := querier.CreateDataset(ctx, gensql.CreateDatasetParams{
		Name:          ds.Name,
		DataproductID: ds.DataproductID,
		Description:   ptrToNullString(ds.Description),
		Pii:           ds.Pii,
		Type:          "bigquery",
		Slug:          slugify(ds.Slug, ds.Name),
		Repo:          ptrToNullString(ds.Repo),
		Keywords:      ds.Keywords,
	})
	if err != nil {
		return nil, err
	}

	schemaJSON, err := json.Marshal(ds.Metadata.Schema.Columns)
	if err != nil {
		return nil, fmt.Errorf("marshalling schema: %w", err)
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
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("Rolling back dataset and datasource_bigquery transaction")
		}
		return nil, err
	}

	for _, subj := range ds.Requesters {
		err = querier.CreateDatasetRequester(ctx, gensql.CreateDatasetRequesterParams{
			DatasetID: created.ID,
			Subject:   subj,
		})
		if err != nil {
			if err := tx.Rollback(); err != nil {
				r.log.WithError(err).Error("Rolling back dataset and datasource_bigquery transaction")
			}
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	ret := datasetFromSQL(created)
	return ret, nil
}

func (r *Repo) UpdateDataset(ctx context.Context, id uuid.UUID, new models.UpdateDataset) (*models.Dataset, error) {
	if new.Keywords == nil {
		new.Keywords = []string{}
	}

	res, err := r.querier.UpdateDataset(ctx, gensql.UpdateDatasetParams{
		Name:        new.Name,
		Description: ptrToNullString(new.Description),
		ID:          id,
		Pii:         new.Pii,
		Slug:        slugify(new.Slug, new.Name),
		Repo:        ptrToNullString(new.Repo),
		Keywords:    new.Keywords,
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataset in database: %w", err)
	}

	return datasetFromSQL(res), nil
}

func (r *Repo) GetBigqueryDatasource(ctx context.Context, datasetID uuid.UUID) (models.BigQuery, error) {
	bq, err := r.querier.GetBigqueryDatasource(ctx, datasetID)
	if err != nil {
		return models.BigQuery{}, err
	}

	return models.BigQuery{
		DatasetID:    bq.DatasetID,
		ProjectID:    bq.ProjectID,
		Dataset:      bq.Dataset,
		Table:        bq.TableName,
		TableType:    models.BigQueryType(strings.ToLower(bq.TableType)),
		LastModified: bq.LastModified,
		Created:      bq.Created,
		Expires:      nullTimeToPtr(bq.Expires),
		Description:  bq.Description.String,
	}, nil
}

func (r *Repo) UpdateBigqueryDatasource(ctx context.Context, id uuid.UUID, schema json.RawMessage, lastModified, expires time.Time, description string) error {
	err := r.querier.UpdateBigqueryDatasourceSchema(ctx, gensql.UpdateBigqueryDatasourceSchemaParams{
		DatasetID: id,
		Schema: pqtype.NullRawMessage{
			RawMessage: schema,
			Valid:      true,
		},
		LastModified: lastModified,
		Expires:      sql.NullTime{Time: expires, Valid: !expires.IsZero()},
		Description:  sql.NullString{String: description, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("updating datasource_bigquery schema: %w", err)
	}

	return nil
}

func (r *Repo) GetDatasetMetadata(ctx context.Context, id uuid.UUID) ([]*models.TableColumn, error) {
	ds, err := r.querier.GetBigqueryDatasource(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting bigquery datasource from database: %w", err)
	}

	var schema []*models.TableColumn
	if ds.Schema.Valid {
		if err := json.Unmarshal(ds.Schema.RawMessage, &schema); err != nil {
			return nil, fmt.Errorf("unmarshalling schema: %w", err)
		}
	}

	return schema, nil
}

func (r *Repo) GetDatasetsByMetabase(ctx context.Context, limit, offset int) ([]*models.Dataset, error) {
	dss := []*models.Dataset{}

	res, err := r.querier.DatasetsByMetabase(ctx, gensql.DatasetsByMetabaseParams{
		Lim:  int32(limit),
		Offs: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("getting dataproducts by metabase from database: %w", err)
	}

	for _, entry := range res {
		dss = append(dss, datasetFromSQL(entry))
	}

	return dss, nil
}

func (r *Repo) GetDatasetsByUserAccess(ctx context.Context, user string) ([]*models.Dataset, error) {
	res, err := r.querier.GetDatasetsByUserAccess(ctx, user)
	if err != nil {
		return nil, err
	}

	dss := []*models.Dataset{}
	for _, d := range res {
		dss = append(dss, datasetFromSQL(d))
	}
	return dss, nil
}

func (r *Repo) DeleteDataset(ctx context.Context, id uuid.UUID) error {
	r.events.TriggerDataproductDelete(ctx, id)

	if err := r.querier.DeleteDataset(ctx, id); err != nil {
		return fmt.Errorf("deleting dataset from database: %w", err)
	}

	return nil
}

func datasetFromSQL(ds gensql.Dataset) *models.Dataset {
	return &models.Dataset{
		ID:            ds.ID,
		Name:          ds.Name,
		Created:       ds.Created,
		LastModified:  ds.LastModified,
		Description:   nullStringToPtr(ds.Description),
		Slug:          ds.Slug,
		Repo:          nullStringToPtr(ds.Repo),
		Pii:           ds.Pii,
		Keywords:      ds.Keywords,
		Type:          ds.Type,
		DataproductID: ds.DataproductID,
	}
}
