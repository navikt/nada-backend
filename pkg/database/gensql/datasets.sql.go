// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: datasets.sql

package gensql

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/tabbed/pqtype"
)

const createBigqueryDatasource = `-- name: CreateBigqueryDatasource :one
INSERT INTO datasource_bigquery ("dataset_id",
                                 "project_id",
                                 "dataset",
                                 "table_name",
                                 "schema",
                                 "last_modified",
                                 "created",
                                 "expires",
                                 "table_type",
                                 "pii_tags")
VALUES ($1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10)
RETURNING dataset_id, project_id, dataset, table_name, schema, last_modified, created, expires, table_type, description, pii_tags
`

type CreateBigqueryDatasourceParams struct {
	DatasetID    uuid.UUID
	ProjectID    string
	Dataset      string
	TableName    string
	Schema       pqtype.NullRawMessage
	LastModified time.Time
	Created      time.Time
	Expires      sql.NullTime
	TableType    string
	PiiTags      pqtype.NullRawMessage
}

func (q *Queries) CreateBigqueryDatasource(ctx context.Context, arg CreateBigqueryDatasourceParams) (DatasourceBigquery, error) {
	row := q.db.QueryRowContext(ctx, createBigqueryDatasource,
		arg.DatasetID,
		arg.ProjectID,
		arg.Dataset,
		arg.TableName,
		arg.Schema,
		arg.LastModified,
		arg.Created,
		arg.Expires,
		arg.TableType,
		arg.PiiTags,
	)
	var i DatasourceBigquery
	err := row.Scan(
		&i.DatasetID,
		&i.ProjectID,
		&i.Dataset,
		&i.TableName,
		&i.Schema,
		&i.LastModified,
		&i.Created,
		&i.Expires,
		&i.TableType,
		&i.Description,
		&i.PiiTags,
	)
	return i, err
}

const createDataset = `-- name: CreateDataset :one
INSERT INTO datasets ("dataproduct_id",
                      "name",
                      "description",
                      "pii",
                      "type",
                      "slug",
                      "repo",
                      "keywords",
                      "anonymisation_description",
                      "target_user"
                      )
VALUES ($1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10)
RETURNING id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
`

type CreateDatasetParams struct {
	DataproductID            uuid.UUID
	Name                     string
	Description              sql.NullString
	Pii                      PiiLevel
	Type                     DatasourceType
	Slug                     string
	Repo                     sql.NullString
	Keywords                 []string
	AnonymisationDescription sql.NullString
	TargetUser               sql.NullString
}

func (q *Queries) CreateDataset(ctx context.Context, arg CreateDatasetParams) (Dataset, error) {
	row := q.db.QueryRowContext(ctx, createDataset,
		arg.DataproductID,
		arg.Name,
		arg.Description,
		arg.Pii,
		arg.Type,
		arg.Slug,
		arg.Repo,
		pq.Array(arg.Keywords),
		arg.AnonymisationDescription,
		arg.TargetUser,
	)
	var i Dataset
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.Pii,
		&i.Created,
		&i.LastModified,
		&i.Type,
		&i.TsvDocument,
		&i.Slug,
		&i.Repo,
		pq.Array(&i.Keywords),
		&i.DataproductID,
		&i.AnonymisationDescription,
		&i.TargetUser,
	)
	return i, err
}

const createDatasetRequester = `-- name: CreateDatasetRequester :exec
INSERT INTO dataset_requesters (dataset_id, "subject")
VALUES ($1, LOWER($2))
`

type CreateDatasetRequesterParams struct {
	DatasetID uuid.UUID
	Subject   string
}

func (q *Queries) CreateDatasetRequester(ctx context.Context, arg CreateDatasetRequesterParams) error {
	_, err := q.db.ExecContext(ctx, createDatasetRequester, arg.DatasetID, arg.Subject)
	return err
}

const datasetKeywords = `-- name: DatasetKeywords :many
SELECT keyword::text, count(1) as "count"
FROM (
	SELECT unnest(keywords) as keyword
	FROM datasets
) s
WHERE true
AND CASE WHEN coalesce(TRIM($1), '') = '' THEN true ELSE keyword ILIKE $1::text || '%' END
GROUP BY keyword
ORDER BY "count" DESC
LIMIT 15
`

type DatasetKeywordsRow struct {
	Keyword string
	Count   int64
}

func (q *Queries) DatasetKeywords(ctx context.Context, keyword string) ([]DatasetKeywordsRow, error) {
	rows, err := q.db.QueryContext(ctx, datasetKeywords, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DatasetKeywordsRow{}
	for rows.Next() {
		var i DatasetKeywordsRow
		if err := rows.Scan(&i.Keyword, &i.Count); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const datasetsByMetabase = `-- name: DatasetsByMetabase :many
SELECT id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
FROM datasets
WHERE id IN (
	SELECT dataset_id
	FROM metabase_metadata
  WHERE "deleted_at" IS NULL
)
ORDER BY last_modified DESC
LIMIT $2 OFFSET $1
`

type DatasetsByMetabaseParams struct {
	Offs int32
	Lim  int32
}

func (q *Queries) DatasetsByMetabase(ctx context.Context, arg DatasetsByMetabaseParams) ([]Dataset, error) {
	rows, err := q.db.QueryContext(ctx, datasetsByMetabase, arg.Offs, arg.Lim)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Dataset{}
	for rows.Next() {
		var i Dataset
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Pii,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Slug,
			&i.Repo,
			pq.Array(&i.Keywords),
			&i.DataproductID,
			&i.AnonymisationDescription,
			&i.TargetUser,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const deleteDataset = `-- name: DeleteDataset :exec
DELETE
FROM datasets
WHERE id = $1
`

func (q *Queries) DeleteDataset(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteDataset, id)
	return err
}

const deleteDatasetRequester = `-- name: DeleteDatasetRequester :exec
DELETE
FROM dataset_requesters
WHERE dataset_id = $1
  AND "subject" = LOWER($2)
`

type DeleteDatasetRequesterParams struct {
	DatasetID uuid.UUID
	Subject   string
}

func (q *Queries) DeleteDatasetRequester(ctx context.Context, arg DeleteDatasetRequesterParams) error {
	_, err := q.db.ExecContext(ctx, deleteDatasetRequester, arg.DatasetID, arg.Subject)
	return err
}

const getBigqueryDatasource = `-- name: GetBigqueryDatasource :one
SELECT dataset_id, project_id, dataset, table_name, schema, last_modified, created, expires, table_type, description, pii_tags
FROM datasource_bigquery
WHERE dataset_id = $1
`

func (q *Queries) GetBigqueryDatasource(ctx context.Context, datasetID uuid.UUID) (DatasourceBigquery, error) {
	row := q.db.QueryRowContext(ctx, getBigqueryDatasource, datasetID)
	var i DatasourceBigquery
	err := row.Scan(
		&i.DatasetID,
		&i.ProjectID,
		&i.Dataset,
		&i.TableName,
		&i.Schema,
		&i.LastModified,
		&i.Created,
		&i.Expires,
		&i.TableType,
		&i.Description,
		&i.PiiTags,
	)
	return i, err
}

const getBigqueryDatasources = `-- name: GetBigqueryDatasources :many
SELECT dataset_id, project_id, dataset, table_name, schema, last_modified, created, expires, table_type, description, pii_tags
FROM datasource_bigquery
`

func (q *Queries) GetBigqueryDatasources(ctx context.Context) ([]DatasourceBigquery, error) {
	rows, err := q.db.QueryContext(ctx, getBigqueryDatasources)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DatasourceBigquery{}
	for rows.Next() {
		var i DatasourceBigquery
		if err := rows.Scan(
			&i.DatasetID,
			&i.ProjectID,
			&i.Dataset,
			&i.TableName,
			&i.Schema,
			&i.LastModified,
			&i.Created,
			&i.Expires,
			&i.TableType,
			&i.Description,
			&i.PiiTags,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDataset = `-- name: GetDataset :one
SELECT id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
FROM datasets
WHERE id = $1
`

func (q *Queries) GetDataset(ctx context.Context, id uuid.UUID) (Dataset, error) {
	row := q.db.QueryRowContext(ctx, getDataset, id)
	var i Dataset
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.Pii,
		&i.Created,
		&i.LastModified,
		&i.Type,
		&i.TsvDocument,
		&i.Slug,
		&i.Repo,
		pq.Array(&i.Keywords),
		&i.DataproductID,
		&i.AnonymisationDescription,
		&i.TargetUser,
	)
	return i, err
}

const getDatasetRequesters = `-- name: GetDatasetRequesters :many
SELECT "subject"
FROM dataset_requesters
WHERE dataset_id = $1
`

func (q *Queries) GetDatasetRequesters(ctx context.Context, datasetID uuid.UUID) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getDatasetRequesters, datasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, err
		}
		items = append(items, subject)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDatasets = `-- name: GetDatasets :many
SELECT id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
FROM datasets
ORDER BY last_modified DESC
LIMIT $2 OFFSET $1
`

type GetDatasetsParams struct {
	Offset int32
	Limit  int32
}

func (q *Queries) GetDatasets(ctx context.Context, arg GetDatasetsParams) ([]Dataset, error) {
	rows, err := q.db.QueryContext(ctx, getDatasets, arg.Offset, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Dataset{}
	for rows.Next() {
		var i Dataset
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Pii,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Slug,
			&i.Repo,
			pq.Array(&i.Keywords),
			&i.DataproductID,
			&i.AnonymisationDescription,
			&i.TargetUser,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDatasetsByGroups = `-- name: GetDatasetsByGroups :many
SELECT id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
FROM datasets
WHERE "group" = ANY ($1::text[])
ORDER BY last_modified DESC
`

func (q *Queries) GetDatasetsByGroups(ctx context.Context, groups []string) ([]Dataset, error) {
	rows, err := q.db.QueryContext(ctx, getDatasetsByGroups, pq.Array(groups))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Dataset{}
	for rows.Next() {
		var i Dataset
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Pii,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Slug,
			&i.Repo,
			pq.Array(&i.Keywords),
			&i.DataproductID,
			&i.AnonymisationDescription,
			&i.TargetUser,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDatasetsByIDs = `-- name: GetDatasetsByIDs :many
SELECT id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
FROM datasets
WHERE id = ANY ($1::uuid[])
ORDER BY last_modified DESC
`

func (q *Queries) GetDatasetsByIDs(ctx context.Context, ids []uuid.UUID) ([]Dataset, error) {
	rows, err := q.db.QueryContext(ctx, getDatasetsByIDs, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Dataset{}
	for rows.Next() {
		var i Dataset
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Pii,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Slug,
			&i.Repo,
			pq.Array(&i.Keywords),
			&i.DataproductID,
			&i.AnonymisationDescription,
			&i.TargetUser,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDatasetsByUserAccess = `-- name: GetDatasetsByUserAccess :many
SELECT id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
FROM datasets
WHERE id = ANY (SELECT dataset_id
                FROM dataset_access
                WHERE "subject" = LOWER($1)
                  AND revoked IS NULL
                  AND (expires > NOW() OR expires IS NULL))
ORDER BY last_modified DESC
`

func (q *Queries) GetDatasetsByUserAccess(ctx context.Context, id string) ([]Dataset, error) {
	rows, err := q.db.QueryContext(ctx, getDatasetsByUserAccess, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Dataset{}
	for rows.Next() {
		var i Dataset
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Pii,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Slug,
			&i.Repo,
			pq.Array(&i.Keywords),
			&i.DataproductID,
			&i.AnonymisationDescription,
			&i.TargetUser,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDatasetsInDataproduct = `-- name: GetDatasetsInDataproduct :many
SELECT id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
FROM datasets
WHERE dataproduct_id = $1
`

func (q *Queries) GetDatasetsInDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]Dataset, error) {
	rows, err := q.db.QueryContext(ctx, getDatasetsInDataproduct, dataproductID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Dataset{}
	for rows.Next() {
		var i Dataset
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Pii,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Slug,
			&i.Repo,
			pq.Array(&i.Keywords),
			&i.DataproductID,
			&i.AnonymisationDescription,
			&i.TargetUser,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const replaceDatasetsTag = `-- name: ReplaceDatasetsTag :exec
UPDATE datasets
SET "keywords"          = array_replace(keywords, $1, $2)
`

type ReplaceDatasetsTagParams struct {
	TagToReplace interface{}
	TagUpdated   interface{}
}

func (q *Queries) ReplaceDatasetsTag(ctx context.Context, arg ReplaceDatasetsTagParams) error {
	_, err := q.db.ExecContext(ctx, replaceDatasetsTag, arg.TagToReplace, arg.TagUpdated)
	return err
}

const updateBigqueryDatasourcePiiTags = `-- name: UpdateBigqueryDatasourcePiiTags :exec
UPDATE datasource_bigquery
SET "pii_tags"        = $1
WHERE dataset_id = $2
`

type UpdateBigqueryDatasourcePiiTagsParams struct {
	PiiTags   pqtype.NullRawMessage
	DatasetID uuid.UUID
}

func (q *Queries) UpdateBigqueryDatasourcePiiTags(ctx context.Context, arg UpdateBigqueryDatasourcePiiTagsParams) error {
	_, err := q.db.ExecContext(ctx, updateBigqueryDatasourcePiiTags, arg.PiiTags, arg.DatasetID)
	return err
}

const updateBigqueryDatasourceSchema = `-- name: UpdateBigqueryDatasourceSchema :exec
UPDATE datasource_bigquery
SET "schema"        = $1,
    "last_modified" = $2,
    "expires"       = $3,
    "description"   = $4
WHERE dataset_id = $5
`

type UpdateBigqueryDatasourceSchemaParams struct {
	Schema       pqtype.NullRawMessage
	LastModified time.Time
	Expires      sql.NullTime
	Description  sql.NullString
	DatasetID    uuid.UUID
}

func (q *Queries) UpdateBigqueryDatasourceSchema(ctx context.Context, arg UpdateBigqueryDatasourceSchemaParams) error {
	_, err := q.db.ExecContext(ctx, updateBigqueryDatasourceSchema,
		arg.Schema,
		arg.LastModified,
		arg.Expires,
		arg.Description,
		arg.DatasetID,
	)
	return err
}

const updateDataset = `-- name: UpdateDataset :one
UPDATE datasets
SET "name"                      = $1,
    "description"               = $2,
    "pii"                       = $3,
    "slug"                      = $4,
    "repo"                      = $5,
    "keywords"                  = $6,
    "dataproduct_id"            = $7,
    "anonymisation_description" = $8,
    "target_user"               = $9
WHERE id = $10
RETURNING id, name, description, pii, created, last_modified, type, tsv_document, slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
`

type UpdateDatasetParams struct {
	Name                     string
	Description              sql.NullString
	Pii                      PiiLevel
	Slug                     string
	Repo                     sql.NullString
	Keywords                 []string
	DataproductID            uuid.UUID
	AnonymisationDescription sql.NullString
	TargetUser               sql.NullString
	ID                       uuid.UUID
}

func (q *Queries) UpdateDataset(ctx context.Context, arg UpdateDatasetParams) (Dataset, error) {
	row := q.db.QueryRowContext(ctx, updateDataset,
		arg.Name,
		arg.Description,
		arg.Pii,
		arg.Slug,
		arg.Repo,
		pq.Array(arg.Keywords),
		arg.DataproductID,
		arg.AnonymisationDescription,
		arg.TargetUser,
		arg.ID,
	)
	var i Dataset
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.Pii,
		&i.Created,
		&i.LastModified,
		&i.Type,
		&i.TsvDocument,
		&i.Slug,
		&i.Repo,
		pq.Array(&i.Keywords),
		&i.DataproductID,
		&i.AnonymisationDescription,
		&i.TargetUser,
	)
	return i, err
}
