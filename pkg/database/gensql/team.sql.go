// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: team.sql

package gensql

import (
	"context"

	"github.com/google/uuid"
)

const deleteNadaToken = `-- name: DeleteNadaToken :exec
DELETE 
FROM nada_tokens
WHERE team = $1
`

func (q *Queries) DeleteNadaToken(ctx context.Context, team string) error {
	_, err := q.db.ExecContext(ctx, deleteNadaToken, team)
	return err
}

const getNadaToken = `-- name: GetNadaToken :one
SELECT token
FROM nada_tokens
WHERE team = $1
`

func (q *Queries) GetNadaToken(ctx context.Context, team string) (uuid.UUID, error) {
	row := q.db.QueryRowContext(ctx, getNadaToken, team)
	var token uuid.UUID
	err := row.Scan(&token)
	return token, err
}
