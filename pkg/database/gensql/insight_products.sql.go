// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0
// source: insight_products.sql

package gensql

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const createInsightProduct = `-- name: CreateInsightProduct :one
INSERT INTO
    insight_product (
        "name",
        "creator",
        "description",
        "type",
        "link",
        "keywords",
        "group",
        "teamkatalogen_url",
        "product_area_id",
        "team_id"
    )
VALUES
    (
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10
    ) RETURNING id, name, description, creator, created, last_modified, type, tsv_document, link, keywords, "group", teamkatalogen_url, product_area_id, team_id
`

type CreateInsightProductParams struct {
	Name             string
	Creator          string
	Description      sql.NullString
	Type             string
	Link             string
	Keywords         []string
	OwnerGroup       string
	TeamkatalogenUrl sql.NullString
	ProductAreaID    sql.NullString
	TeamID           sql.NullString
}

func (q *Queries) CreateInsightProduct(ctx context.Context, arg CreateInsightProductParams) (InsightProduct, error) {
	row := q.db.QueryRowContext(ctx, createInsightProduct,
		arg.Name,
		arg.Creator,
		arg.Description,
		arg.Type,
		arg.Link,
		pq.Array(arg.Keywords),
		arg.OwnerGroup,
		arg.TeamkatalogenUrl,
		arg.ProductAreaID,
		arg.TeamID,
	)
	var i InsightProduct
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.Creator,
		&i.Created,
		&i.LastModified,
		&i.Type,
		&i.TsvDocument,
		&i.Link,
		pq.Array(&i.Keywords),
		&i.Group,
		&i.TeamkatalogenUrl,
		&i.ProductAreaID,
		&i.TeamID,
	)
	return i, err
}

const deleteInsightProduct = `-- name: DeleteInsightProduct :exec
DELETE FROM
    insight_product
WHERE
    id = $1
`

func (q *Queries) DeleteInsightProduct(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteInsightProduct, id)
	return err
}

const getInsightProduct = `-- name: GetInsightProduct :one
SELECT
    id, name, description, creator, created, last_modified, type, tsv_document, link, keywords, "group", teamkatalogen_url, product_area_id, team_id
FROM
    insight_product
WHERE
    id = $1
`

func (q *Queries) GetInsightProduct(ctx context.Context, id uuid.UUID) (InsightProduct, error) {
	row := q.db.QueryRowContext(ctx, getInsightProduct, id)
	var i InsightProduct
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.Creator,
		&i.Created,
		&i.LastModified,
		&i.Type,
		&i.TsvDocument,
		&i.Link,
		pq.Array(&i.Keywords),
		&i.Group,
		&i.TeamkatalogenUrl,
		&i.ProductAreaID,
		&i.TeamID,
	)
	return i, err
}

const getInsightProductByGroups = `-- name: GetInsightProductByGroups :many
SELECT
    id, name, description, creator, created, last_modified, type, tsv_document, link, keywords, "group", teamkatalogen_url, product_area_id, team_id
FROM
    insight_product
WHERE
    "group" = ANY ($1 :: text [])
ORDER BY
    last_modified DESC
`

func (q *Queries) GetInsightProductByGroups(ctx context.Context, groups []string) ([]InsightProduct, error) {
	rows, err := q.db.QueryContext(ctx, getInsightProductByGroups, pq.Array(groups))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []InsightProduct{}
	for rows.Next() {
		var i InsightProduct
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Link,
			pq.Array(&i.Keywords),
			&i.Group,
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getInsightProducts = `-- name: GetInsightProducts :many
SELECT
    id, name, description, creator, created, last_modified, type, tsv_document, link, keywords, "group", teamkatalogen_url, product_area_id, team_id
FROM
    insight_product
ORDER BY
    last_modified DESC
`

func (q *Queries) GetInsightProducts(ctx context.Context) ([]InsightProduct, error) {
	rows, err := q.db.QueryContext(ctx, getInsightProducts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []InsightProduct{}
	for rows.Next() {
		var i InsightProduct
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Link,
			pq.Array(&i.Keywords),
			&i.Group,
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getInsightProductsByIDs = `-- name: GetInsightProductsByIDs :many
SELECT
    id, name, description, creator, created, last_modified, type, tsv_document, link, keywords, "group", teamkatalogen_url, product_area_id, team_id
FROM
    insight_product
WHERE
    id = ANY ($1 :: uuid [])
ORDER BY
    last_modified DESC
`

func (q *Queries) GetInsightProductsByIDs(ctx context.Context, ids []uuid.UUID) ([]InsightProduct, error) {
	rows, err := q.db.QueryContext(ctx, getInsightProductsByIDs, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []InsightProduct{}
	for rows.Next() {
		var i InsightProduct
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Link,
			pq.Array(&i.Keywords),
			&i.Group,
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getInsightProductsByProductArea = `-- name: GetInsightProductsByProductArea :many
SELECT
    id, name, description, creator, created, last_modified, type, tsv_document, link, keywords, "group", teamkatalogen_url, product_area_id, team_id
FROM
    insight_product
WHERE
    product_area_id = $1
ORDER BY
    last_modified DESC
`

func (q *Queries) GetInsightProductsByProductArea(ctx context.Context, productAreaID sql.NullString) ([]InsightProduct, error) {
	rows, err := q.db.QueryContext(ctx, getInsightProductsByProductArea, productAreaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []InsightProduct{}
	for rows.Next() {
		var i InsightProduct
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Link,
			pq.Array(&i.Keywords),
			&i.Group,
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getInsightProductsByTeam = `-- name: GetInsightProductsByTeam :many
SELECT
    id, name, description, creator, created, last_modified, type, tsv_document, link, keywords, "group", teamkatalogen_url, product_area_id, team_id
FROM
    insight_product
WHERE
    team_id = $1
ORDER BY
    last_modified DESC
`

func (q *Queries) GetInsightProductsByTeam(ctx context.Context, teamID sql.NullString) ([]InsightProduct, error) {
	rows, err := q.db.QueryContext(ctx, getInsightProductsByTeam, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []InsightProduct{}
	for rows.Next() {
		var i InsightProduct
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Type,
			&i.TsvDocument,
			&i.Link,
			pq.Array(&i.Keywords),
			&i.Group,
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const updateInsightProduct = `-- name: UpdateInsightProduct :one
UPDATE
    insight_product
SET
    "name" = $1,
    "creator" = $2,
    "description" = $3,
    "type" = $4,
    "link" = $5,
    "keywords" = $6,
    "teamkatalogen_url" = $7,
    "product_area_id" = $8,
    "team_id" = $9
WHERE
    id = $10 RETURNING id, name, description, creator, created, last_modified, type, tsv_document, link, keywords, "group", teamkatalogen_url, product_area_id, team_id
`

type UpdateInsightProductParams struct {
	Name             string
	Creator          string
	Description      sql.NullString
	Type             string
	Link             string
	Keywords         []string
	TeamkatalogenUrl sql.NullString
	ProductAreaID    sql.NullString
	TeamID           sql.NullString
	ID               uuid.UUID
}

func (q *Queries) UpdateInsightProduct(ctx context.Context, arg UpdateInsightProductParams) (InsightProduct, error) {
	row := q.db.QueryRowContext(ctx, updateInsightProduct,
		arg.Name,
		arg.Creator,
		arg.Description,
		arg.Type,
		arg.Link,
		pq.Array(arg.Keywords),
		arg.TeamkatalogenUrl,
		arg.ProductAreaID,
		arg.TeamID,
		arg.ID,
	)
	var i InsightProduct
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.Creator,
		&i.Created,
		&i.LastModified,
		&i.Type,
		&i.TsvDocument,
		&i.Link,
		pq.Array(&i.Keywords),
		&i.Group,
		&i.TeamkatalogenUrl,
		&i.ProductAreaID,
		&i.TeamID,
	)
	return i, err
}