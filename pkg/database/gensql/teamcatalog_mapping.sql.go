// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.23.0
// source: teamcatalog_mapping.sql

package gensql

import (
	"context"
	"database/sql"
)

const createTeamAndProductAreaMapping = `-- name: CreateTeamAndProductAreaMapping :one
INSERT INTO "team_productarea_mapping" (
    "team_id",
    "product_area_id"
) VALUES (
    $1,
    $2
) RETURNING team_id, product_area_id
`

type CreateTeamAndProductAreaMappingParams struct {
	TeamID        string
	ProductAreaID sql.NullString
}

func (q *Queries) CreateTeamAndProductAreaMapping(ctx context.Context, arg CreateTeamAndProductAreaMappingParams) (TeamProductareaMapping, error) {
	row := q.db.QueryRowContext(ctx, createTeamAndProductAreaMapping, arg.TeamID, arg.ProductAreaID)
	var i TeamProductareaMapping
	err := row.Scan(&i.TeamID, &i.ProductAreaID)
	return i, err
}

const getTeamAndProductAreaID = `-- name: GetTeamAndProductAreaID :one
SELECT team_id, product_area_id
FROM "team_productarea_mapping"
WHERE team_id = $1
`

func (q *Queries) GetTeamAndProductAreaID(ctx context.Context, teamID string) (TeamProductareaMapping, error) {
	row := q.db.QueryRowContext(ctx, getTeamAndProductAreaID, teamID)
	var i TeamProductareaMapping
	err := row.Scan(&i.TeamID, &i.ProductAreaID)
	return i, err
}

const getTeamsAndProductAreaIDs = `-- name: GetTeamsAndProductAreaIDs :many
SELECT team_id, product_area_id
FROM "team_productarea_mapping"
`

func (q *Queries) GetTeamsAndProductAreaIDs(ctx context.Context) ([]TeamProductareaMapping, error) {
	rows, err := q.db.QueryContext(ctx, getTeamsAndProductAreaIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TeamProductareaMapping{}
	for rows.Next() {
		var i TeamProductareaMapping
		if err := rows.Scan(&i.TeamID, &i.ProductAreaID); err != nil {
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

const updateProductAreaForTeam = `-- name: UpdateProductAreaForTeam :exec
UPDATE "team_productarea_mapping"
SET product_area_id = $1
WHERE team_id = $2
`

type UpdateProductAreaForTeamParams struct {
	ProductAreaID sql.NullString
	TeamID        string
}

func (q *Queries) UpdateProductAreaForTeam(ctx context.Context, arg UpdateProductAreaForTeamParams) error {
	_, err := q.db.ExecContext(ctx, updateProductAreaForTeam, arg.ProductAreaID, arg.TeamID)
	return err
}
