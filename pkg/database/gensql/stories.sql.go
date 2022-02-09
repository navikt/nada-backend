// Code generated by sqlc. DO NOT EDIT.
// source: stories.sql

package gensql

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const createStory = `-- name: CreateStory :one
INSERT INTO stories (
	"name",
	"group",
	"description",
    "keywords"
) VALUES (
	$1,
	$2,
    $3,
    $4
)
RETURNING id, name, created, last_modified, "group", description, keywords
`

type CreateStoryParams struct {
	Name        string
	Grp         string
	Description sql.NullString
	Keywords    []string
}

func (q *Queries) CreateStory(ctx context.Context, arg CreateStoryParams) (Story, error) {
	row := q.db.QueryRowContext(ctx, createStory,
		arg.Name,
		arg.Grp,
		arg.Description,
		pq.Array(arg.Keywords),
	)
	var i Story
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Created,
		&i.LastModified,
		&i.Group,
		&i.Description,
		pq.Array(&i.Keywords),
	)
	return i, err
}

const createStoryView = `-- name: CreateStoryView :one
INSERT INTO story_views (
	"story_id",
	"sort",
	"type",
	"spec"
) VALUES (
	$1,
	$2,
	$3,
	$4
)
RETURNING id, story_id, sort, type, spec
`

type CreateStoryViewParams struct {
	StoryID uuid.UUID
	Sort    int32
	Type    StoryViewType
	Spec    json.RawMessage
}

func (q *Queries) CreateStoryView(ctx context.Context, arg CreateStoryViewParams) (StoryView, error) {
	row := q.db.QueryRowContext(ctx, createStoryView,
		arg.StoryID,
		arg.Sort,
		arg.Type,
		arg.Spec,
	)
	var i StoryView
	err := row.Scan(
		&i.ID,
		&i.StoryID,
		&i.Sort,
		&i.Type,
		&i.Spec,
	)
	return i, err
}

const deleteStory = `-- name: DeleteStory :exec
DELETE FROM stories
WHERE id = $1
`

func (q *Queries) DeleteStory(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteStory, id)
	return err
}

const deleteStoryViews = `-- name: DeleteStoryViews :exec
DELETE FROM story_views
WHERE story_id = $1
`

func (q *Queries) DeleteStoryViews(ctx context.Context, storyID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteStoryViews, storyID)
	return err
}

const getStories = `-- name: GetStories :many
SELECT id, name, created, last_modified, "group", description, keywords
FROM stories
ORDER BY created DESC
`

func (q *Queries) GetStories(ctx context.Context) ([]Story, error) {
	rows, err := q.db.QueryContext(ctx, getStories)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Story{}
	for rows.Next() {
		var i Story
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Created,
			&i.LastModified,
			&i.Group,
			&i.Description,
			pq.Array(&i.Keywords),
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

const getStoriesByIDs = `-- name: GetStoriesByIDs :many
SELECT id, name, created, last_modified, "group", description, keywords
FROM stories
WHERE id = ANY ($1::uuid[])
ORDER BY created DESC
`

func (q *Queries) GetStoriesByIDs(ctx context.Context, ids []uuid.UUID) ([]Story, error) {
	rows, err := q.db.QueryContext(ctx, getStoriesByIDs, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Story{}
	for rows.Next() {
		var i Story
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Created,
			&i.LastModified,
			&i.Group,
			&i.Description,
			pq.Array(&i.Keywords),
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

const getStory = `-- name: GetStory :one
SELECT id, name, created, last_modified, "group", description, keywords
FROM stories
WHERE id = $1
`

func (q *Queries) GetStory(ctx context.Context, id uuid.UUID) (Story, error) {
	row := q.db.QueryRowContext(ctx, getStory, id)
	var i Story
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Created,
		&i.LastModified,
		&i.Group,
		&i.Description,
		pq.Array(&i.Keywords),
	)
	return i, err
}

const getStoryFromToken = `-- name: GetStoryFromToken :one
SELECT id, name, created, last_modified, "group", description, keywords
FROM stories
WHERE id = (SELECT story_id FROM story_tokens WHERE token = $1)
`

func (q *Queries) GetStoryFromToken(ctx context.Context, token uuid.UUID) (Story, error) {
	row := q.db.QueryRowContext(ctx, getStoryFromToken, token)
	var i Story
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Created,
		&i.LastModified,
		&i.Group,
		&i.Description,
		pq.Array(&i.Keywords),
	)
	return i, err
}

const getStoryToken = `-- name: GetStoryToken :one
SELECT id, story_id, token
FROM story_tokens
WHERE story_id = $1
`

func (q *Queries) GetStoryToken(ctx context.Context, storyID uuid.UUID) (StoryToken, error) {
	row := q.db.QueryRowContext(ctx, getStoryToken, storyID)
	var i StoryToken
	err := row.Scan(&i.ID, &i.StoryID, &i.Token)
	return i, err
}

const getStoryView = `-- name: GetStoryView :one
SELECT id, story_id, sort, type, spec
FROM story_views
WHERE id = $1
`

func (q *Queries) GetStoryView(ctx context.Context, id uuid.UUID) (StoryView, error) {
	row := q.db.QueryRowContext(ctx, getStoryView, id)
	var i StoryView
	err := row.Scan(
		&i.ID,
		&i.StoryID,
		&i.Sort,
		&i.Type,
		&i.Spec,
	)
	return i, err
}

const getStoryViews = `-- name: GetStoryViews :many
SELECT id, story_id, sort, type, spec
FROM story_views
WHERE story_id = $1
ORDER BY sort ASC
`

func (q *Queries) GetStoryViews(ctx context.Context, storyID uuid.UUID) ([]StoryView, error) {
	rows, err := q.db.QueryContext(ctx, getStoryViews, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []StoryView{}
	for rows.Next() {
		var i StoryView
		if err := rows.Scan(
			&i.ID,
			&i.StoryID,
			&i.Sort,
			&i.Type,
			&i.Spec,
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

const updateStory = `-- name: UpdateStory :one
UPDATE stories
SET
	"name" = $1,
	"group" = $2
WHERE id = $3
RETURNING id, name, created, last_modified, "group", description, keywords
`

type UpdateStoryParams struct {
	Name string
	Grp  string
	ID   uuid.UUID
}

func (q *Queries) UpdateStory(ctx context.Context, arg UpdateStoryParams) (Story, error) {
	row := q.db.QueryRowContext(ctx, updateStory, arg.Name, arg.Grp, arg.ID)
	var i Story
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Created,
		&i.LastModified,
		&i.Group,
		&i.Description,
		pq.Array(&i.Keywords),
	)
	return i, err
}
