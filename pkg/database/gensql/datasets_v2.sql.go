// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: datasets_v2.sql

package gensql

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const getAccessibleDatasets = `-- name: GetAccessibleDatasets :many
SELECT
  DISTINCT ON (ds.id)
  ds.id, ds.name, ds.description, ds.pii, ds.created, ds.last_modified, ds.type, ds.tsv_document, ds.slug, ds.repo, ds.keywords, ds.dataproduct_id, ds.anonymisation_description, ds.target_user,
  dsa.subject AS "subject",
  dsa.owner AS "access_owner",
  dp.slug AS dp_slug,
  dp.name AS dp_name,
  dp.group
FROM
  datasets ds
  LEFT JOIN dataproducts dp ON ds.dataproduct_id = dp.id
  LEFT JOIN dataset_access dsa ON dsa.dataset_id = ds.id
WHERE
  array_length($1::TEXT[], 1) IS NOT NULL AND array_length($1::TEXT[], 1)!=0
  AND dp.group = ANY($1 :: TEXT [])
  OR (
    SPLIT_PART(dsa.subject, ':', 1) != 'serviceAccount'
    AND (
        $2::TEXT IS NOT NULL
        AND dsa.subject = LOWER($2)
        OR SPLIT_PART(dsa.subject, ':', 2) = ANY($1::TEXT[])
    )
  )
  AND revoked IS NULL
  AND (
    expires > NOW()
    OR expires IS NULL
  )
ORDER BY
  ds.id,
  ds.last_modified DESC
`

type GetAccessibleDatasetsParams struct {
	Groups    []string
	Requester string
}

type GetAccessibleDatasetsRow struct {
	ID                       uuid.UUID
	Name                     string
	Description              sql.NullString
	Pii                      PiiLevel
	Created                  time.Time
	LastModified             time.Time
	Type                     DatasourceType
	TsvDocument              interface{}
	Slug                     string
	Repo                     sql.NullString
	Keywords                 []string
	DataproductID            uuid.UUID
	AnonymisationDescription sql.NullString
	TargetUser               sql.NullString
	Subject                  sql.NullString
	AccessOwner              sql.NullString
	DpSlug                   sql.NullString
	DpName                   sql.NullString
	Group                    sql.NullString
}

func (q *Queries) GetAccessibleDatasets(ctx context.Context, arg GetAccessibleDatasetsParams) ([]GetAccessibleDatasetsRow, error) {
	rows, err := q.db.QueryContext(ctx, getAccessibleDatasets, pq.Array(arg.Groups), arg.Requester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetAccessibleDatasetsRow{}
	for rows.Next() {
		var i GetAccessibleDatasetsRow
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
			&i.Subject,
			&i.AccessOwner,
			&i.DpSlug,
			&i.DpName,
			&i.Group,
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

const getAccessibleDatasetsByOwnedServiceAccounts = `-- name: GetAccessibleDatasetsByOwnedServiceAccounts :many
SELECT
  ds.id, ds.name, ds.description, ds.pii, ds.created, ds.last_modified, ds.type, ds.tsv_document, ds.slug, ds.repo, ds.keywords, ds.dataproduct_id, ds.anonymisation_description, ds.target_user,
  dsa.subject AS "subject",
  dsa.owner AS "access_owner",
  dp.slug AS dp_slug,
  dp.name AS dp_name,
  dp.group
FROM
  datasets ds
  LEFT JOIN dataproducts dp ON ds.dataproduct_id = dp.id
  LEFT JOIN dataset_access dsa ON dsa.dataset_id = ds.id
WHERE
  SPLIT_PART("subject", ':', 1) = 'serviceAccount'
  AND (
    dsa.owner = $1
    OR dsa.owner = ANY($2::TEXT[])
  )  
  AND revoked IS NULL
  AND (
    expires > NOW()
    OR expires IS NULL
  )
ORDER BY
  ds.last_modified DESC
`

type GetAccessibleDatasetsByOwnedServiceAccountsParams struct {
	Requester string
	Groups    []string
}

type GetAccessibleDatasetsByOwnedServiceAccountsRow struct {
	ID                       uuid.UUID
	Name                     string
	Description              sql.NullString
	Pii                      PiiLevel
	Created                  time.Time
	LastModified             time.Time
	Type                     DatasourceType
	TsvDocument              interface{}
	Slug                     string
	Repo                     sql.NullString
	Keywords                 []string
	DataproductID            uuid.UUID
	AnonymisationDescription sql.NullString
	TargetUser               sql.NullString
	Subject                  sql.NullString
	AccessOwner              sql.NullString
	DpSlug                   sql.NullString
	DpName                   sql.NullString
	Group                    sql.NullString
}

func (q *Queries) GetAccessibleDatasetsByOwnedServiceAccounts(ctx context.Context, arg GetAccessibleDatasetsByOwnedServiceAccountsParams) ([]GetAccessibleDatasetsByOwnedServiceAccountsRow, error) {
	rows, err := q.db.QueryContext(ctx, getAccessibleDatasetsByOwnedServiceAccounts, arg.Requester, pq.Array(arg.Groups))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetAccessibleDatasetsByOwnedServiceAccountsRow{}
	for rows.Next() {
		var i GetAccessibleDatasetsByOwnedServiceAccountsRow
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
			&i.Subject,
			&i.AccessOwner,
			&i.DpSlug,
			&i.DpName,
			&i.Group,
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

const getAllDatasetsMinimal = `-- name: GetAllDatasetsMinimal :many
SELECT ds.id, ds.created, name, project_id, dataset, table_name 
FROM datasets ds 
JOIN datasource_bigquery dsb 
ON ds.id = dsb.dataset_id
`

type GetAllDatasetsMinimalRow struct {
	ID        uuid.UUID
	Created   time.Time
	Name      string
	ProjectID string
	Dataset   string
	TableName string
}

func (q *Queries) GetAllDatasetsMinimal(ctx context.Context) ([]GetAllDatasetsMinimalRow, error) {
	rows, err := q.db.QueryContext(ctx, getAllDatasetsMinimal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetAllDatasetsMinimalRow{}
	for rows.Next() {
		var i GetAllDatasetsMinimalRow
		if err := rows.Scan(
			&i.ID,
			&i.Created,
			&i.Name,
			&i.ProjectID,
			&i.Dataset,
			&i.TableName,
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

const getDatasetComplete = `-- name: GetDatasetComplete :many
SELECT
  ds_id, ds_name, ds_description, ds_created, ds_last_modified, ds_slug, pii, ds_keywords, bq_id, bq_created, bq_last_modified, bq_expires, bq_description, bq_missing_since, pii_tags, bq_project, bq_dataset, bq_table_name, bq_table_type, pseudo_columns, bq_schema, ds_dp_id, mapping_services, access_id, access_subject, access_owner, access_granter, access_expires, access_created, access_revoked, access_request_id, mb_database_id, mb_deleted_at
FROM
  dataset_view
WHERE
  ds_id = $1
`

func (q *Queries) GetDatasetComplete(ctx context.Context, id uuid.UUID) ([]DatasetView, error) {
	rows, err := q.db.QueryContext(ctx, getDatasetComplete, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DatasetView{}
	for rows.Next() {
		var i DatasetView
		if err := rows.Scan(
			&i.DsID,
			&i.DsName,
			&i.DsDescription,
			&i.DsCreated,
			&i.DsLastModified,
			&i.DsSlug,
			&i.Pii,
			pq.Array(&i.DsKeywords),
			&i.BqID,
			&i.BqCreated,
			&i.BqLastModified,
			&i.BqExpires,
			&i.BqDescription,
			&i.BqMissingSince,
			&i.PiiTags,
			&i.BqProject,
			&i.BqDataset,
			&i.BqTableName,
			&i.BqTableType,
			pq.Array(&i.PseudoColumns),
			&i.BqSchema,
			&i.DsDpID,
			pq.Array(&i.MappingServices),
			&i.AccessID,
			&i.AccessSubject,
			&i.AccessOwner,
			&i.AccessGranter,
			&i.AccessExpires,
			&i.AccessCreated,
			&i.AccessRevoked,
			&i.AccessRequestID,
			&i.MbDatabaseID,
			&i.MbDeletedAt,
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
