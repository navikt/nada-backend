// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.22.0
// source: polly.sql

package gensql

import (
	"context"

	"github.com/google/uuid"
)

const createPollyDocumentation = `-- name: CreatePollyDocumentation :one
INSERT INTO polly_documentation ("external_id",
                                 "name",
                                 "url")
VALUES ($1,
        $2,
        $3)
RETURNING id, external_id, name, url
`

type CreatePollyDocumentationParams struct {
	ExternalID string
	Name       string
	Url        string
}

func (q *Queries) CreatePollyDocumentation(ctx context.Context, arg CreatePollyDocumentationParams) (PollyDocumentation, error) {
	row := q.db.QueryRowContext(ctx, createPollyDocumentation, arg.ExternalID, arg.Name, arg.Url)
	var i PollyDocumentation
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.Name,
		&i.Url,
	)
	return i, err
}

const getPollyDocumentation = `-- name: GetPollyDocumentation :one
SELECT id, external_id, name, url
FROM polly_documentation
WHERE id = $1
`

func (q *Queries) GetPollyDocumentation(ctx context.Context, id uuid.UUID) (PollyDocumentation, error) {
	row := q.db.QueryRowContext(ctx, getPollyDocumentation, id)
	var i PollyDocumentation
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.Name,
		&i.Url,
	)
	return i, err
}
