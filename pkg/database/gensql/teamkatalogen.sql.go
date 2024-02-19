// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: teamkatalogen.sql

package gensql

import (
	"context"

	"github.com/google/uuid"
)

const getAllTeams = `-- name: GetAllTeams :many
SELECT id, product_area_id, name
FROM tk_teams
`

func (q *Queries) GetAllTeams(ctx context.Context) ([]TkTeam, error) {
	rows, err := q.db.QueryContext(ctx, getAllTeams)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TkTeam{}
	for rows.Next() {
		var i TkTeam
		if err := rows.Scan(&i.ID, &i.ProductAreaID, &i.Name); err != nil {
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

const getProductArea = `-- name: GetProductArea :one
SELECT id, name, "areaType"
FROM tk_product_areas
WHERE id = $1
`

func (q *Queries) GetProductArea(ctx context.Context, id uuid.UUID) (TkProductArea, error) {
	row := q.db.QueryRowContext(ctx, getProductArea, id)
	var i TkProductArea
	err := row.Scan(&i.ID, &i.Name, &i.AreaType)
	return i, err
}

const getProductAreas = `-- name: GetProductAreas :many
SELECT id, name, "areaType"
FROM tk_product_areas
`

func (q *Queries) GetProductAreas(ctx context.Context) ([]TkProductArea, error) {
	rows, err := q.db.QueryContext(ctx, getProductAreas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TkProductArea{}
	for rows.Next() {
		var i TkProductArea
		if err := rows.Scan(&i.ID, &i.Name, &i.AreaType); err != nil {
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

const getTeamsInProductArea = `-- name: GetTeamsInProductArea :many
SELECT id, product_area_id, name
FROM tk_teams
WHERE product_area_id = $1
`

func (q *Queries) GetTeamsInProductArea(ctx context.Context, productAreaID uuid.NullUUID) ([]TkTeam, error) {
	rows, err := q.db.QueryContext(ctx, getTeamsInProductArea, productAreaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TkTeam{}
	for rows.Next() {
		var i TkTeam
		if err := rows.Scan(&i.ID, &i.ProductAreaID, &i.Name); err != nil {
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

const upsertProductArea = `-- name: UpsertProductArea :exec
INSERT INTO
    tk_product_areas (id, name)
VALUES
    ($1, $2) ON CONFLICT (id) DO
UPDATE
SET
    name = $2
`

type UpsertProductAreaParams struct {
	ID   uuid.UUID
	Name string
}

func (q *Queries) UpsertProductArea(ctx context.Context, arg UpsertProductAreaParams) error {
	_, err := q.db.ExecContext(ctx, upsertProductArea, arg.ID, arg.Name)
	return err
}

const upsertTeam = `-- name: UpsertTeam :exec
INSERT INTO
    tk_teams(id, product_area_id, name)
VALUES
    ($1, $2, $3) ON CONFLICT (id) DO
UPDATE
SET
    product_area_id = $2,
    name = $3
`

type UpsertTeamParams struct {
	ID            uuid.UUID
	ProductAreaID uuid.NullUUID
	Name          string
}

func (q *Queries) UpsertTeam(ctx context.Context, arg UpsertTeamParams) error {
	_, err := q.db.ExecContext(ctx, upsertTeam, arg.ID, arg.ProductAreaID, arg.Name)
	return err
}
