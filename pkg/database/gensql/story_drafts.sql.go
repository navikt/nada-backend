// Code generated by sqlc. DO NOT EDIT.
// source: story_drafts.sql

package gensql

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

const createStoryDraft = `-- name: CreateStoryDraft :one
INSERT INTO story_drafts (
	"name"
) VALUES (
	$1
)
RETURNING id, name, created
`

func (q *Queries) CreateStoryDraft(ctx context.Context, name string) (StoryDraft, error) {
	row := q.db.QueryRowContext(ctx, createStoryDraft, name)
	var i StoryDraft
	err := row.Scan(&i.ID, &i.Name, &i.Created)
	return i, err
}

const createStoryViewDraft = `-- name: CreateStoryViewDraft :one
INSERT INTO story_view_drafts (
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

type CreateStoryViewDraftParams struct {
	StoryID uuid.UUID
	Sort    int32
	Type    StoryViewType
	Spec    json.RawMessage
}

func (q *Queries) CreateStoryViewDraft(ctx context.Context, arg CreateStoryViewDraftParams) (StoryViewDraft, error) {
	row := q.db.QueryRowContext(ctx, createStoryViewDraft,
		arg.StoryID,
		arg.Sort,
		arg.Type,
		arg.Spec,
	)
	var i StoryViewDraft
	err := row.Scan(
		&i.ID,
		&i.StoryID,
		&i.Sort,
		&i.Type,
		&i.Spec,
	)
	return i, err
}

const deleteStoryDraft = `-- name: DeleteStoryDraft :exec
DELETE FROM story_drafts
WHERE id = $1
`

func (q *Queries) DeleteStoryDraft(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteStoryDraft, id)
	return err
}

const deleteStoryViewDraft = `-- name: DeleteStoryViewDraft :exec
DELETE FROM story_view_drafts
WHERE story_id = $1
`

func (q *Queries) DeleteStoryViewDraft(ctx context.Context, storyID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteStoryViewDraft, storyID)
	return err
}

const getStoryDraft = `-- name: GetStoryDraft :one
SELECT id, name, created
FROM story_drafts
WHERE id = $1
`

func (q *Queries) GetStoryDraft(ctx context.Context, id uuid.UUID) (StoryDraft, error) {
	row := q.db.QueryRowContext(ctx, getStoryDraft, id)
	var i StoryDraft
	err := row.Scan(&i.ID, &i.Name, &i.Created)
	return i, err
}

const getStoryDrafts = `-- name: GetStoryDrafts :many
SELECT id, name, created
FROM story_drafts
ORDER BY created DESC
`

func (q *Queries) GetStoryDrafts(ctx context.Context) ([]StoryDraft, error) {
	rows, err := q.db.QueryContext(ctx, getStoryDrafts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []StoryDraft{}
	for rows.Next() {
		var i StoryDraft
		if err := rows.Scan(&i.ID, &i.Name, &i.Created); err != nil {
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

const getStoryViewDraft = `-- name: GetStoryViewDraft :one
SELECT id, story_id, sort, type, spec
FROM story_view_drafts
WHERE id = $1
`

func (q *Queries) GetStoryViewDraft(ctx context.Context, id uuid.UUID) (StoryViewDraft, error) {
	row := q.db.QueryRowContext(ctx, getStoryViewDraft, id)
	var i StoryViewDraft
	err := row.Scan(
		&i.ID,
		&i.StoryID,
		&i.Sort,
		&i.Type,
		&i.Spec,
	)
	return i, err
}

const getStoryViewDrafts = `-- name: GetStoryViewDrafts :many
SELECT id, story_id, sort, type, spec
FROM story_view_drafts
WHERE story_id = $1
ORDER BY sort ASC
`

func (q *Queries) GetStoryViewDrafts(ctx context.Context, storyID uuid.UUID) ([]StoryViewDraft, error) {
	rows, err := q.db.QueryContext(ctx, getStoryViewDrafts, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []StoryViewDraft{}
	for rows.Next() {
		var i StoryViewDraft
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
