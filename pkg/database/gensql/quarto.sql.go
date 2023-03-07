// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0
// source: quarto.sql

package gensql

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const createQuartoStory = `-- name: CreateQuartoStory :one
INSERT INTO quarto_stories (
	"name",
    "creator",
	"description",
	"keywords",
	"teamkatalogen_url",
    "product_area_id",
    "team_id"
) VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
    $6,
    $7
)
RETURNING id, name, creator, created, last_modified, description, keywords, teamkatalogen_url, product_area_id, team_id, "group"
`

type CreateQuartoStoryParams struct {
	Name             string
	Creator          string
	Description      string
	Keywords         []string
	TeamkatalogenUrl sql.NullString
	ProductAreaID    sql.NullString
	TeamID           sql.NullString
}

func (q *Queries) CreateQuartoStory(ctx context.Context, arg CreateQuartoStoryParams) (QuartoStory, error) {
	row := q.db.QueryRowContext(ctx, createQuartoStory,
		arg.Name,
		arg.Creator,
		arg.Description,
		pq.Array(arg.Keywords),
		arg.TeamkatalogenUrl,
		arg.ProductAreaID,
		arg.TeamID,
	)
	var i QuartoStory
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Creator,
		&i.Created,
		&i.LastModified,
		&i.Description,
		pq.Array(&i.Keywords),
		&i.TeamkatalogenUrl,
		&i.ProductAreaID,
		&i.TeamID,
		&i.Group,
	)
	return i, err
}

const deleteQuartoStory = `-- name: DeleteQuartoStory :exec
DELETE FROM quarto_stories
WHERE id = $1
`

func (q *Queries) DeleteQuartoStory(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteQuartoStory, id)
	return err
}

const getQuartoStories = `-- name: GetQuartoStories :many
SELECT id, name, creator, created, last_modified, description, keywords, teamkatalogen_url, product_area_id, team_id, "group"
FROM quarto_stories
ORDER BY last_modified DESC
`

func (q *Queries) GetQuartoStories(ctx context.Context) ([]QuartoStory, error) {
	rows, err := q.db.QueryContext(ctx, getQuartoStories)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []QuartoStory{}
	for rows.Next() {
		var i QuartoStory
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Description,
			pq.Array(&i.Keywords),
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getQuartoStoriesByGroups = `-- name: GetQuartoStoriesByGroups :many
SELECT id, name, creator, created, last_modified, description, keywords, teamkatalogen_url, product_area_id, team_id, "group"
FROM quarto_stories
WHERE "group" = ANY ($1::text[])
ORDER BY last_modified DESC
`

func (q *Queries) GetQuartoStoriesByGroups(ctx context.Context, groups []string) ([]QuartoStory, error) {
	rows, err := q.db.QueryContext(ctx, getQuartoStoriesByGroups, pq.Array(groups))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []QuartoStory{}
	for rows.Next() {
		var i QuartoStory
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Description,
			pq.Array(&i.Keywords),
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getQuartoStoriesByIDs = `-- name: GetQuartoStoriesByIDs :many
SELECT id, name, creator, created, last_modified, description, keywords, teamkatalogen_url, product_area_id, team_id, "group"
FROM quarto_stories
WHERE id = ANY ($1::uuid[])
ORDER BY last_modified DESC
`

func (q *Queries) GetQuartoStoriesByIDs(ctx context.Context, ids []uuid.UUID) ([]QuartoStory, error) {
	rows, err := q.db.QueryContext(ctx, getQuartoStoriesByIDs, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []QuartoStory{}
	for rows.Next() {
		var i QuartoStory
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Description,
			pq.Array(&i.Keywords),
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getQuartoStoriesByProductArea = `-- name: GetQuartoStoriesByProductArea :many
SELECT id, name, creator, created, last_modified, description, keywords, teamkatalogen_url, product_area_id, team_id, "group"
FROM quarto_stories
WHERE product_area_id = $1
ORDER BY last_modified DESC
`

func (q *Queries) GetQuartoStoriesByProductArea(ctx context.Context, productAreaID sql.NullString) ([]QuartoStory, error) {
	rows, err := q.db.QueryContext(ctx, getQuartoStoriesByProductArea, productAreaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []QuartoStory{}
	for rows.Next() {
		var i QuartoStory
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Description,
			pq.Array(&i.Keywords),
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getQuartoStoriesByTeam = `-- name: GetQuartoStoriesByTeam :many
SELECT id, name, creator, created, last_modified, description, keywords, teamkatalogen_url, product_area_id, team_id, "group"
FROM quarto_stories
WHERE team_id = $1
ORDER BY last_modified DESC
`

func (q *Queries) GetQuartoStoriesByTeam(ctx context.Context, teamID sql.NullString) ([]QuartoStory, error) {
	rows, err := q.db.QueryContext(ctx, getQuartoStoriesByTeam, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []QuartoStory{}
	for rows.Next() {
		var i QuartoStory
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Creator,
			&i.Created,
			&i.LastModified,
			&i.Description,
			pq.Array(&i.Keywords),
			&i.TeamkatalogenUrl,
			&i.ProductAreaID,
			&i.TeamID,
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

const getQuartoStory = `-- name: GetQuartoStory :one
SELECT id, name, creator, created, last_modified, description, keywords, teamkatalogen_url, product_area_id, team_id, "group"
FROM quarto_stories
WHERE id = $1
`

func (q *Queries) GetQuartoStory(ctx context.Context, id uuid.UUID) (QuartoStory, error) {
	row := q.db.QueryRowContext(ctx, getQuartoStory, id)
	var i QuartoStory
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Creator,
		&i.Created,
		&i.LastModified,
		&i.Description,
		pq.Array(&i.Keywords),
		&i.TeamkatalogenUrl,
		&i.ProductAreaID,
		&i.TeamID,
		&i.Group,
	)
	return i, err
}

const replaceQuartoStoriesTag = `-- name: ReplaceQuartoStoriesTag :exec
UPDATE quarto_stories
SET "keywords" = array_replace(keywords, $1, $2)
`

type ReplaceQuartoStoriesTagParams struct {
	TagToReplace interface{}
	TagUpdated   interface{}
}

func (q *Queries) ReplaceQuartoStoriesTag(ctx context.Context, arg ReplaceQuartoStoriesTagParams) error {
	_, err := q.db.ExecContext(ctx, replaceQuartoStoriesTag, arg.TagToReplace, arg.TagUpdated)
	return err
}

const updateQuartoStory = `-- name: UpdateQuartoStory :one
UPDATE quarto_stories
SET
	"name" = $1,
    "creator" = $2,
	"description" = $3,
	"keywords" = $4,
	"teamkatalogen_url" = $5,
    "product_area_id" = $6,
    "team_id" = $7
WHERE id = $8
RETURNING id, name, creator, created, last_modified, description, keywords, teamkatalogen_url, product_area_id, team_id, "group"
`

type UpdateQuartoStoryParams struct {
	Name             string
	Creator          string
	Description      string
	Keywords         []string
	TeamkatalogenUrl sql.NullString
	ProductAreaID    sql.NullString
	TeamID           sql.NullString
	ID               uuid.UUID
}

func (q *Queries) UpdateQuartoStory(ctx context.Context, arg UpdateQuartoStoryParams) (QuartoStory, error) {
	row := q.db.QueryRowContext(ctx, updateQuartoStory,
		arg.Name,
		arg.Creator,
		arg.Description,
		pq.Array(arg.Keywords),
		arg.TeamkatalogenUrl,
		arg.ProductAreaID,
		arg.TeamID,
		arg.ID,
	)
	var i QuartoStory
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Creator,
		&i.Created,
		&i.LastModified,
		&i.Description,
		pq.Array(&i.Keywords),
		&i.TeamkatalogenUrl,
		&i.ProductAreaID,
		&i.TeamID,
		&i.Group,
	)
	return i, err
}
