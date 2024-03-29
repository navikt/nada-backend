package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) ListAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*models.Access, error) {
	access, err := r.Querier.ListAccessToDataset(ctx, datasetID)
	if err != nil {
		return nil, err
	}

	ret := []*models.Access{}
	for _, a := range access {
		ret = append(ret, accessFromSQL(a))
	}

	return ret, nil
}

func (r *Repo) GrantAccessToDataset(ctx context.Context, datasetID uuid.UUID, expires *time.Time, subject, granter string) (*models.Access, error) {
	a, err := r.Querier.GetActiveAccessToDatasetForSubject(ctx, gensql.GetActiveAccessToDatasetForSubjectParams{
		DatasetID: datasetID,
		Subject:   subject,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	querier := r.Querier.WithTx(tx)

	if len(a.Subject) > 0 {
		if err := querier.RevokeAccessToDataset(ctx, a.ID); err != nil {
			if err := tx.Rollback(); err != nil {
				r.log.WithError(err).Error("Rolling back revoke and grant access to dataproduct transaction")
			}
			return nil, err
		}
	}

	access, err := querier.GrantAccessToDataset(ctx, gensql.GrantAccessToDatasetParams{
		DatasetID: datasetID,
		Subject:   emailOfSubjectToLower(subject),
		Expires:   ptrToNullTime(expires),
		Granter:   granter,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("Rolling back revoke and grant access to dataproduct transaction")
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	r.events.TriggerDatasetGrant(ctx, datasetID, subject)

	return accessFromSQL(access), nil
}

func (r *Repo) GetAccessToDataset(ctx context.Context, id uuid.UUID) (*models.Access, error) {
	access, err := r.Querier.GetAccessToDataset(ctx, id)
	if err != nil {
		return nil, err
	}
	return accessFromSQL(access), nil
}

func (r *Repo) ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*models.Access, error) {
	access, err := r.Querier.ListActiveAccessToDataset(ctx, datasetID)
	if err != nil {
		return nil, err
	}

	var ret []*models.Access
	for _, e := range access {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

func (r *Repo) RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error {
	if err := r.Querier.RevokeAccessToDataset(ctx, id); err != nil {
		return err
	}

	access, err := r.GetAccessToDataset(ctx, id)
	if err != nil {
		return err
	}

	r.events.TriggerDatasetRevoke(ctx, access.DatasetID, access.Subject)

	return nil
}

func (r *Repo) GetUnrevokedExpiredAccess(ctx context.Context) ([]*models.Access, error) {
	expired, err := r.Querier.ListUnrevokedExpiredAccessEntries(ctx)
	if err != nil {
		return nil, err
	}

	var ret []*models.Access
	for _, e := range expired {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

func accessFromSQL(access gensql.DatasetAccess) *models.Access {
	return &models.Access{
		ID:              access.ID,
		Subject:         access.Subject,
		Granter:         access.Granter,
		Expires:         nullTimeToPtr(access.Expires),
		Created:         access.Created,
		Revoked:         nullTimeToPtr(access.Revoked),
		DatasetID:       access.DatasetID,
		AccessRequestID: nullUUIDToUUIDPtr(access.AccessRequestID),
	}
}

func emailOfSubjectToLower(subectWithType string) string {
	parts := strings.Split(subectWithType, ":")
	parts[1] = strings.ToLower(parts[1])

	return strings.Join(parts, ":")
}
