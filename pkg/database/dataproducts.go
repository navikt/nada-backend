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

func (r *Repo) GetDataproducts(ctx context.Context, limit, offset int) ([]*models.Dataproduct, error) {
	datasets := []*models.Dataproduct{}

	res, err := r.querier.GetDataproducts(ctx, gensql.GetDataproductsParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, fmt.Errorf("getting datasets from database: %w", err)
	}

	for _, entry := range res {
		datasets = append(datasets, dataproductFromSQL(entry))
	}

	return datasets, nil
}

func (r *Repo) GetDataproductsByUserAccess(ctx context.Context, user string) ([]*models.Dataproduct, error) {
	res, err := r.querier.GetDataproductsByUserAccess(ctx, user)
	if err != nil {
		return nil, err
	}

	dps := []*models.Dataproduct{}
	for _, d := range res {
		dps = append(dps, dataproductFromSQL(d))
	}
	return dps, nil
}

func (r *Repo) GetDataproductsByGroups(ctx context.Context, groups []string) ([]*models.Dataproduct, error) {
	dps := []*models.Dataproduct{}

	res, err := r.querier.GetDataproductsByGroups(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("getting dataproducts by group from database: %w", err)
	}

	for _, entry := range res {
		dps = append(dps, dataproductFromSQL(entry))
	}

	return dps, nil
}

func (r *Repo) GetDataproduct(ctx context.Context, id uuid.UUID) (*models.Dataproduct, error) {
	res, err := r.querier.GetDataproduct(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting dataproduct from database: %w", err)
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) GetDataproductsByMetabase(ctx context.Context, limit, offset int) ([]*models.Dataproduct, error) {
	dps := []*models.Dataproduct{}

	res, err := r.querier.DataproductsByMetabase(ctx, gensql.DataproductsByMetabaseParams{
		Lim:  int32(limit),
		Offs: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("getting dataproducts by metabase from database: %w", err)
	}

	for _, entry := range res {
		dps = append(dps, dataproductFromSQL(entry))
	}

	return dps, nil
}

func (r *Repo) CreateDataproduct(ctx context.Context, dp models.NewDataproduct) (*models.Dataproduct, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	if dp.Keywords == nil {
		dp.Keywords = []string{}
	}

	querier := r.querier.WithTx(tx)
	created, err := querier.CreateDataproduct(ctx, gensql.CreateDataproductParams{
		Name:                  dp.Name,
		Description:           ptrToNullString(dp.Description),
		Pii:                   dp.Pii,
		Type:                  "bigquery",
		OwnerGroup:            dp.Group,
		OwnerTeamkatalogenUrl: ptrToNullString(dp.TeamkatalogenURL),
		Slug:                  slugify(dp.Slug, dp.Name),
		Repo:                  ptrToNullString(dp.Repo),
		Keywords:              dp.Keywords,
	})
	if err != nil {
		return nil, err
	}

	schemaJSON, err := json.Marshal(dp.Metadata.Schema.Columns)
	if err != nil {
		return nil, fmt.Errorf("marshalling schema: %w", err)
	}

	_, err = querier.CreateBigqueryDatasource(ctx, gensql.CreateBigqueryDatasourceParams{
		DataproductID: created.ID,
		ProjectID:     dp.BigQuery.ProjectID,
		Dataset:       dp.BigQuery.Dataset,
		TableName:     dp.BigQuery.Table,
		Schema:        pqtype.NullRawMessage{RawMessage: schemaJSON, Valid: len(schemaJSON) > 4},
		LastModified:  dp.Metadata.LastModified,
		Created:       dp.Metadata.Created,
		Expires:       sql.NullTime{Time: dp.Metadata.Expires, Valid: !dp.Metadata.Expires.IsZero()},
		TableType:     string(dp.Metadata.TableType),
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("Rolling back dataproduct and datasource_bigquery transaction")
		}
		return nil, err
	}

	for _, subj := range dp.Requesters {
		err = querier.CreateDataproductRequester(ctx, gensql.CreateDataproductRequesterParams{
			DataproductID: created.ID,
			Subject:       subj,
		})
		if err != nil {
			if err := tx.Rollback(); err != nil {
				r.log.WithError(err).Error("Rolling back dataproduct and datasource_bigquery transaction")
			}
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	ret := dataproductFromSQL(created)
	return ret, nil
}

func (r *Repo) UpdateDataproduct(ctx context.Context, id uuid.UUID, new models.UpdateDataproduct) (*models.Dataproduct, error) {
	if new.Keywords == nil {
		new.Keywords = []string{}
	}

	res, err := r.querier.UpdateDataproduct(ctx, gensql.UpdateDataproductParams{
		Name:                  new.Name,
		Description:           ptrToNullString(new.Description),
		ID:                    id,
		Pii:                   new.Pii,
		OwnerTeamkatalogenUrl: ptrToNullString(new.TeamkatalogenURL),
		Slug:                  slugify(new.Slug, new.Name),
		Repo:                  ptrToNullString(new.Repo),
		Keywords:              new.Keywords,
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct in database: %w", err)
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) DeleteDataproduct(ctx context.Context, id uuid.UUID) error {
	r.events.TriggerDataproductDelete(ctx, id)

	if err := r.querier.DeleteDataproduct(ctx, id); err != nil {
		return fmt.Errorf("deleting dataproduct from database: %w", err)
	}

	return nil
}

func (r *Repo) GetBigqueryDatasources(ctx context.Context) ([]gensql.DatasourceBigquery, error) {
	return r.querier.GetBigqueryDatasources(ctx)
}

func (r *Repo) GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID) (models.BigQuery, error) {
	bq, err := r.querier.GetBigqueryDatasource(ctx, dataproductID)
	if err != nil {
		return models.BigQuery{}, err
	}

	return models.BigQuery{
		DataproductID: bq.DataproductID,
		ProjectID:     bq.ProjectID,
		Dataset:       bq.Dataset,
		Table:         bq.TableName,
		TableType:     models.BigQueryType(strings.ToLower(bq.TableType)),
		LastModified:  bq.LastModified,
		Created:       bq.Created,
		Expires:       nullTimeToPtr(bq.Expires),
		Description:   bq.Description.String,
	}, nil
}

func (r *Repo) UpdateBigqueryDatasource(ctx context.Context, id uuid.UUID, schema json.RawMessage, lastModified, expires time.Time, description string) error {
	err := r.querier.UpdateBigqueryDatasourceSchema(ctx, gensql.UpdateBigqueryDatasourceSchemaParams{
		DataproductID: id,
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

func (r *Repo) GetDataproductMetadata(ctx context.Context, id uuid.UUID) ([]*models.TableColumn, error) {
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

func (r *Repo) DataproductKeywords(ctx context.Context, prefix string) ([]*models.Keyword, error) {
	kws, err := r.querier.DataproductKeywords(ctx, prefix)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.Keyword, len(kws))
	for i, kw := range kws {
		ret[i] = &models.Keyword{
			Keyword: kw.Keyword,
			Count:   int(kw.Count),
		}
	}
	return ret, nil
}

func (r *Repo) DataproductGroupStats(ctx context.Context, limit, offset int) ([]*models.GroupStats, error) {
	stats, err := r.querier.DataproductGroupStats(ctx, gensql.DataproductGroupStatsParams{
		Lim:  int32(limit),
		Offs: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	ret := make([]*models.GroupStats, len(stats))
	for i, s := range stats {
		ret[i] = &models.GroupStats{
			Email:        s.Group,
			Dataproducts: int(s.Count),
		}
	}
	return ret, nil
}

func (r *Repo) CreateDataproductExtract(ctx context.Context, bq *models.BigQuery, bucketPath, jobID, email string) (*models.DataproductExtractInfo, error) {
	extract, err := r.querier.CreateDataproductExtract(ctx, gensql.CreateDataproductExtractParams{
		DataproductID: bq.DataproductID,
		Email:         email,
		BucketPath:    bucketPath,
		JobID:         jobID,
	})
	if err != nil {
		return nil, err
	}

	return &models.DataproductExtractInfo{
		ID:            extract.ID,
		DataproductID: extract.DataproductID,
		Email:         extract.Email,
		Created:       extract.Created,
		BucketPath:    extract.BucketPath,
		Ready:         nullTimeToPtr(extract.ReadyAt),
		Expired:       nullTimeToPtr(extract.ExpiredAt),
	}, nil
}

func (r *Repo) GetUnreadyDataproductExtractions(ctx context.Context) ([]gensql.DataproductExtraction, error) {
	return r.querier.GetUnreadyDataproductExtractions(ctx)
}

func (r *Repo) SetDataproductExtractReady(ctx context.Context, id uuid.UUID) error {
	return r.querier.SetDataproductExtractReady(ctx, id)
}

func (r *Repo) SetDataproductExtractExpired(ctx context.Context, id uuid.UUID) error {
	return r.querier.SetDataproductExtractExpired(ctx, id)
}

func (r *Repo) GetDataproductExtractionsForUser(ctx context.Context, email string) ([]*models.DataproductExtractInfo, error) {
	extractionsSQL, err := r.querier.GetDataproductExtractionsForUser(ctx, email)
	if err != nil {
		return nil, err
	}

	extractions := make([]*models.DataproductExtractInfo, len(extractionsSQL))
	for _, e := range extractionsSQL {
		extractions = append(extractions, &models.DataproductExtractInfo{
			ID:            e.ID,
			DataproductID: e.DataproductID,
			Email:         e.Email,
			Created:       e.Created,
			BucketPath:    e.BucketPath,
			Ready:         nullTimeToPtr(e.ReadyAt),
			Expired:       nullTimeToPtr(e.ExpiredAt),
		})
	}
	return extractions, nil
}

func dataproductFromSQL(dp gensql.Dataproduct) *models.Dataproduct {
	return &models.Dataproduct{
		ID:           dp.ID,
		Name:         dp.Name,
		Created:      dp.Created,
		LastModified: dp.LastModified,
		Description:  nullStringToPtr(dp.Description),
		Slug:         dp.Slug,
		Repo:         nullStringToPtr(dp.Repo),
		Pii:          dp.Pii,
		Keywords:     dp.Keywords,
		Owner: &models.Owner{
			Group:            dp.Group,
			TeamkatalogenURL: nullStringToPtr(dp.TeamkatalogenUrl),
		},
		Type: dp.Type,
	}
}
