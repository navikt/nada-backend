// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: dataproducts_v2.sql

package gensql

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const getDataproductKeywords = `-- name: GetDataproductKeywords :many
SELECT DISTINCT unnest(keywords)::text FROM datasets ds WHERE ds.dataproduct_id = $1
`

func (q *Queries) GetDataproductKeywords(ctx context.Context, dpid uuid.UUID) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getDataproductKeywords, dpid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var column_1 string
		if err := rows.Scan(&column_1); err != nil {
			return nil, err
		}
		items = append(items, column_1)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDataproductWithDatasetsBasic = `-- name: GetDataproductWithDatasetsBasic :many
SELECT dp.id, dp.name, dp.description, "group", dp.created, dp.last_modified, dp.tsv_document, dp.slug, teamkatalogen_url, team_contact, team_id, team_name, pa_name, pa_id, ds.id, ds.name, ds.description, pii, ds.created, ds.last_modified, type, ds.tsv_document, ds.slug, repo, keywords, dataproduct_id, anonymisation_description, target_user
FROM dataproduct_with_teamkatalogen_view dp LEFT JOIN datasets ds ON ds.dataproduct_id = dp.id
WHERE dp.id = $1
`

type GetDataproductWithDatasetsBasicRow struct {
	ID                       uuid.UUID
	Name                     string
	Description              sql.NullString
	Group                    string
	Created                  time.Time
	LastModified             time.Time
	TsvDocument              interface{}
	Slug                     string
	TeamkatalogenUrl         sql.NullString
	TeamContact              sql.NullString
	TeamID                   uuid.NullUUID
	TeamName                 sql.NullString
	PaName                   sql.NullString
	PaID                     uuid.NullUUID
	ID_2                     uuid.NullUUID
	Name_2                   sql.NullString
	Description_2            sql.NullString
	Pii                      NullPiiLevel
	Created_2                sql.NullTime
	LastModified_2           sql.NullTime
	Type                     NullDatasourceType
	TsvDocument_2            interface{}
	Slug_2                   sql.NullString
	Repo                     sql.NullString
	Keywords                 []string
	DataproductID            uuid.NullUUID
	AnonymisationDescription sql.NullString
	TargetUser               sql.NullString
}

func (q *Queries) GetDataproductWithDatasetsBasic(ctx context.Context, id uuid.UUID) ([]GetDataproductWithDatasetsBasicRow, error) {
	rows, err := q.db.QueryContext(ctx, getDataproductWithDatasetsBasic, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetDataproductWithDatasetsBasicRow{}
	for rows.Next() {
		var i GetDataproductWithDatasetsBasicRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Group,
			&i.Created,
			&i.LastModified,
			&i.TsvDocument,
			&i.Slug,
			&i.TeamkatalogenUrl,
			&i.TeamContact,
			&i.TeamID,
			&i.TeamName,
			&i.PaName,
			&i.PaID,
			&i.ID_2,
			&i.Name_2,
			&i.Description_2,
			&i.Pii,
			&i.Created_2,
			&i.LastModified_2,
			&i.Type,
			&i.TsvDocument_2,
			&i.Slug_2,
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

const getDataproductsNumberByTeam = `-- name: GetDataproductsNumberByTeam :one
SELECT COUNT(*) as "count"
FROM dataproducts
WHERE team_id = $1
`

func (q *Queries) GetDataproductsNumberByTeam(ctx context.Context, teamID uuid.NullUUID) (int64, error) {
	row := q.db.QueryRowContext(ctx, getDataproductsNumberByTeam, teamID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const getDataproductsWithDatasets = `-- name: GetDataproductsWithDatasets :many
SELECT dp.dp_id, dp.dp_name, dp.dp_description, dp.dp_group, dp.dp_created, dp.dp_last_modified, dp.dp_slug, dp.teamkatalogen_url, dp.team_contact, dp.team_id, dp.team_name, dp.pa_name, dp.pa_id, dp.ds_dp_id, dp.ds_id, dp.ds_name, dp.ds_description, dp.ds_created, dp.ds_last_modified, dp.ds_slug, dp.ds_keywords, dsrc.last_modified as "dsrc_last_modified"
FROM dataproduct_view dp
LEFT JOIN datasource_bigquery dsrc ON dsrc.dataset_id = dp.ds_id
WHERE (array_length($1::uuid[], 1) IS NULL OR dp_id = ANY ($1))
 AND (array_length($2::TEXT[], 1) IS NULL OR dp_group = ANY ($2))
ORDER BY ds_name ASC
`

type GetDataproductsWithDatasetsParams struct {
	Ids    []uuid.UUID
	Groups []string
}

type GetDataproductsWithDatasetsRow struct {
	DpID             uuid.UUID
	DpName           string
	DpDescription    sql.NullString
	DpGroup          string
	DpCreated        time.Time
	DpLastModified   time.Time
	DpSlug           string
	TeamkatalogenUrl sql.NullString
	TeamContact      sql.NullString
	TeamID           uuid.NullUUID
	TeamName         sql.NullString
	PaName           sql.NullString
	PaID             uuid.NullUUID
	DsDpID           uuid.NullUUID
	DsID             uuid.NullUUID
	DsName           sql.NullString
	DsDescription    sql.NullString
	DsCreated        sql.NullTime
	DsLastModified   sql.NullTime
	DsSlug           sql.NullString
	DsKeywords       []string
	DsrcLastModified sql.NullTime
}

func (q *Queries) GetDataproductsWithDatasets(ctx context.Context, arg GetDataproductsWithDatasetsParams) ([]GetDataproductsWithDatasetsRow, error) {
	rows, err := q.db.QueryContext(ctx, getDataproductsWithDatasets, pq.Array(arg.Ids), pq.Array(arg.Groups))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetDataproductsWithDatasetsRow{}
	for rows.Next() {
		var i GetDataproductsWithDatasetsRow
		if err := rows.Scan(
			&i.DpID,
			&i.DpName,
			&i.DpDescription,
			&i.DpGroup,
			&i.DpCreated,
			&i.DpLastModified,
			&i.DpSlug,
			&i.TeamkatalogenUrl,
			&i.TeamContact,
			&i.TeamID,
			&i.TeamName,
			&i.PaName,
			&i.PaID,
			&i.DsDpID,
			&i.DsID,
			&i.DsName,
			&i.DsDescription,
			&i.DsCreated,
			&i.DsLastModified,
			&i.DsSlug,
			pq.Array(&i.DsKeywords),
			&i.DsrcLastModified,
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

const getDataproductsWithDatasetsAndAccessRequests = `-- name: GetDataproductsWithDatasetsAndAccessRequests :many
SELECT dp.dp_id, dp.dp_name, dp.dp_description, dp.dp_group, dp.dp_created, dp.dp_last_modified, dp.dp_slug, dp.teamkatalogen_url, dp.team_contact, dp.team_id, dp.team_name, dp.pa_name, dp.pa_id, dp.ds_dp_id, dp.ds_id, dp.ds_name, dp.ds_description, dp.ds_created, dp.ds_last_modified, dp.ds_slug, dp.ds_keywords, dsrc.last_modified as "dsrc_last_modified",
 dar.id as "dar_id", dar.dataset_id as "dar_dataset_id", dar.subject as "dar_subject", dar.owner as "dar_owner",
  dar.expires as "dar_expires", dar.status as "dar_status", dar.granter as "dar_granter", dar.reason as "dar_reason", 
  dar.closed as "dar_closed", dar.polly_documentation_id as "dar_polly_documentation_id", dar.created as "dar_created"
FROM dataproduct_view dp
LEFT JOIN datasource_bigquery dsrc ON dsrc.dataset_id = dp.ds_id
LEFT JOIN dataset_access_requests dar ON dar.dataset_id = dp.ds_id AND dar.status = 'pending'
WHERE (array_length($1::uuid[], 1) IS NULL OR dp_id = ANY ($1))
 AND (array_length($2::TEXT[], 1) IS NULL OR dp_group = ANY ($2))
`

type GetDataproductsWithDatasetsAndAccessRequestsParams struct {
	Ids    []uuid.UUID
	Groups []string
}

type GetDataproductsWithDatasetsAndAccessRequestsRow struct {
	DpID                    uuid.UUID
	DpName                  string
	DpDescription           sql.NullString
	DpGroup                 string
	DpCreated               time.Time
	DpLastModified          time.Time
	DpSlug                  string
	TeamkatalogenUrl        sql.NullString
	TeamContact             sql.NullString
	TeamID                  uuid.NullUUID
	TeamName                sql.NullString
	PaName                  sql.NullString
	PaID                    uuid.NullUUID
	DsDpID                  uuid.NullUUID
	DsID                    uuid.NullUUID
	DsName                  sql.NullString
	DsDescription           sql.NullString
	DsCreated               sql.NullTime
	DsLastModified          sql.NullTime
	DsSlug                  sql.NullString
	DsKeywords              []string
	DsrcLastModified        sql.NullTime
	DarID                   uuid.NullUUID
	DarDatasetID            uuid.NullUUID
	DarSubject              sql.NullString
	DarOwner                sql.NullString
	DarExpires              sql.NullTime
	DarStatus               NullAccessRequestStatusType
	DarGranter              sql.NullString
	DarReason               sql.NullString
	DarClosed               sql.NullTime
	DarPollyDocumentationID uuid.NullUUID
	DarCreated              sql.NullTime
}

func (q *Queries) GetDataproductsWithDatasetsAndAccessRequests(ctx context.Context, arg GetDataproductsWithDatasetsAndAccessRequestsParams) ([]GetDataproductsWithDatasetsAndAccessRequestsRow, error) {
	rows, err := q.db.QueryContext(ctx, getDataproductsWithDatasetsAndAccessRequests, pq.Array(arg.Ids), pq.Array(arg.Groups))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetDataproductsWithDatasetsAndAccessRequestsRow{}
	for rows.Next() {
		var i GetDataproductsWithDatasetsAndAccessRequestsRow
		if err := rows.Scan(
			&i.DpID,
			&i.DpName,
			&i.DpDescription,
			&i.DpGroup,
			&i.DpCreated,
			&i.DpLastModified,
			&i.DpSlug,
			&i.TeamkatalogenUrl,
			&i.TeamContact,
			&i.TeamID,
			&i.TeamName,
			&i.PaName,
			&i.PaID,
			&i.DsDpID,
			&i.DsID,
			&i.DsName,
			&i.DsDescription,
			&i.DsCreated,
			&i.DsLastModified,
			&i.DsSlug,
			pq.Array(&i.DsKeywords),
			&i.DsrcLastModified,
			&i.DarID,
			&i.DarDatasetID,
			&i.DarSubject,
			&i.DarOwner,
			&i.DarExpires,
			&i.DarStatus,
			&i.DarGranter,
			&i.DarReason,
			&i.DarClosed,
			&i.DarPollyDocumentationID,
			&i.DarCreated,
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
