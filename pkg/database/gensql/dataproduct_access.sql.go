// Code generated by sqlc. DO NOT EDIT.
// source: dataproduct_access.sql

package gensql

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const getAccessToDataproduct = `-- name: GetAccessToDataproduct :one
SELECT id, dataproduct_id, subject, granter, expires, created, revoked, access_request_id
FROM dataproduct_access
WHERE id = $1
`

func (q *Queries) GetAccessToDataproduct(ctx context.Context, id uuid.UUID) (DataproductAccess, error) {
	row := q.db.QueryRowContext(ctx, getAccessToDataproduct, id)
	var i DataproductAccess
	err := row.Scan(
		&i.ID,
		&i.DataproductID,
		&i.Subject,
		&i.Granter,
		&i.Expires,
		&i.Created,
		&i.Revoked,
		&i.AccessRequestID,
	)
	return i, err
}

const getActiveAccessToDataproductForSubject = `-- name: GetActiveAccessToDataproductForSubject :one
SELECT id, dataproduct_id, subject, granter, expires, created, revoked, access_request_id
FROM dataproduct_access
WHERE dataproduct_id = $1 
AND "subject" = $2 
AND revoked IS NULL 
AND (
  expires IS NULL 
  OR expires >= NOW()
)
`

type GetActiveAccessToDataproductForSubjectParams struct {
	DataproductID uuid.UUID
	Subject       string
}

func (q *Queries) GetActiveAccessToDataproductForSubject(ctx context.Context, arg GetActiveAccessToDataproductForSubjectParams) (DataproductAccess, error) {
	row := q.db.QueryRowContext(ctx, getActiveAccessToDataproductForSubject, arg.DataproductID, arg.Subject)
	var i DataproductAccess
	err := row.Scan(
		&i.ID,
		&i.DataproductID,
		&i.Subject,
		&i.Granter,
		&i.Expires,
		&i.Created,
		&i.Revoked,
		&i.AccessRequestID,
	)
	return i, err
}

const grantAccessToDataproduct = `-- name: GrantAccessToDataproduct :one
INSERT INTO dataproduct_access (dataproduct_id,
                                "subject",
                                granter,
                                expires,
                                access_request_id)
VALUES ($1,
        LOWER($2),
        LOWER($3),
        $4,
        $5)
RETURNING id, dataproduct_id, subject, granter, expires, created, revoked, access_request_id
`

type GrantAccessToDataproductParams struct {
	DataproductID   uuid.UUID
	Subject         string
	Granter         string
	Expires         sql.NullTime
	AccessRequestID uuid.NullUUID
}

func (q *Queries) GrantAccessToDataproduct(ctx context.Context, arg GrantAccessToDataproductParams) (DataproductAccess, error) {
	row := q.db.QueryRowContext(ctx, grantAccessToDataproduct,
		arg.DataproductID,
		arg.Subject,
		arg.Granter,
		arg.Expires,
		arg.AccessRequestID,
	)
	var i DataproductAccess
	err := row.Scan(
		&i.ID,
		&i.DataproductID,
		&i.Subject,
		&i.Granter,
		&i.Expires,
		&i.Created,
		&i.Revoked,
		&i.AccessRequestID,
	)
	return i, err
}

const listAccessToDataproduct = `-- name: ListAccessToDataproduct :many
SELECT id, dataproduct_id, subject, granter, expires, created, revoked, access_request_id
FROM dataproduct_access
WHERE dataproduct_id = $1
`

func (q *Queries) ListAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]DataproductAccess, error) {
	rows, err := q.db.QueryContext(ctx, listAccessToDataproduct, dataproductID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DataproductAccess{}
	for rows.Next() {
		var i DataproductAccess
		if err := rows.Scan(
			&i.ID,
			&i.DataproductID,
			&i.Subject,
			&i.Granter,
			&i.Expires,
			&i.Created,
			&i.Revoked,
			&i.AccessRequestID,
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

const listActiveAccessToDataproduct = `-- name: ListActiveAccessToDataproduct :many
SELECT id, dataproduct_id, subject, granter, expires, created, revoked, access_request_id
FROM dataproduct_access
WHERE dataproduct_id = $1 AND revoked IS NULL AND (expires IS NULL OR expires >= NOW())
`

func (q *Queries) ListActiveAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]DataproductAccess, error) {
	rows, err := q.db.QueryContext(ctx, listActiveAccessToDataproduct, dataproductID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DataproductAccess{}
	for rows.Next() {
		var i DataproductAccess
		if err := rows.Scan(
			&i.ID,
			&i.DataproductID,
			&i.Subject,
			&i.Granter,
			&i.Expires,
			&i.Created,
			&i.Revoked,
			&i.AccessRequestID,
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

const listUnrevokedExpiredAccessEntries = `-- name: ListUnrevokedExpiredAccessEntries :many
SELECT id, dataproduct_id, subject, granter, expires, created, revoked, access_request_id
FROM dataproduct_access
WHERE revoked IS NULL
  AND expires < NOW()
`

func (q *Queries) ListUnrevokedExpiredAccessEntries(ctx context.Context) ([]DataproductAccess, error) {
	rows, err := q.db.QueryContext(ctx, listUnrevokedExpiredAccessEntries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DataproductAccess{}
	for rows.Next() {
		var i DataproductAccess
		if err := rows.Scan(
			&i.ID,
			&i.DataproductID,
			&i.Subject,
			&i.Granter,
			&i.Expires,
			&i.Created,
			&i.Revoked,
			&i.AccessRequestID,
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

const revokeAccessToDataproduct = `-- name: RevokeAccessToDataproduct :exec
UPDATE dataproduct_access
SET revoked = NOW()
WHERE id = $1
`

func (q *Queries) RevokeAccessToDataproduct(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, revokeAccessToDataproduct, id)
	return err
}
