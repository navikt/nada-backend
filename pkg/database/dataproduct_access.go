package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) AddRequesterToDataproduct(ctx context.Context, dataproductID uuid.UUID, subject string) error {
	requesters, err := r.querier.GetDataproductRequesters(ctx, dataproductID)
	if err != nil {
		return err
	}

	for _, r := range requesters {
		if subject == r {
			return nil
		}
	}

	return r.querier.CreateDataproductRequester(ctx, gensql.CreateDataproductRequesterParams{
		DataproductID: dataproductID,
		Subject:       subject,
	})
}

func (r *Repo) GetDataproductRequesters(ctx context.Context, id uuid.UUID) ([]string, error) {
	return r.querier.GetDataproductRequesters(ctx, id)
}

func (r *Repo) RemoveRequesterFromDataproduct(ctx context.Context, dataproductID uuid.UUID, subject string) error {
	return r.querier.DeleteDataproductRequester(ctx, gensql.DeleteDataproductRequesterParams{
		DataproductID: dataproductID,
		Subject:       subject,
	})
}

func (r *Repo) ListAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]*models.Access, error) {
	access, err := r.querier.ListAccessToDataproduct(ctx, dataproductID)
	if err != nil {
		return nil, err
	}

	ret := []*models.Access{}
	for _, a := range access {
		ret = append(ret, accessFromSQL(a))
	}

	return ret, nil
}

func (r *Repo) GrantAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID, expires *time.Time, subject, granter string) (*models.Access, error) {
	a, err := r.querier.GetActiveAccessToDataproductForSubject(ctx, gensql.GetActiveAccessToDataproductForSubjectParams{
		DataproductID: dataproductID,
		Subject:       subject,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	querier := r.querier.WithTx(tx)

	if len(a.Subject) > 0 {
		if err := querier.RevokeAccessToDataproduct(ctx, a.ID); err != nil {
			if err := tx.Rollback(); err != nil {
				r.log.WithError(err).Error("Rolling back revoke and grant access to dataproduct transaction")
			}
			return nil, err
		}
	}

	access, err := querier.GrantAccessToDataproduct(ctx, gensql.GrantAccessToDataproductParams{
		DataproductID: dataproductID,
		Subject:       subject,
		Expires:       ptrToNullTime(expires),
		Granter:       granter,
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

	if subject == "group:all-users@nav.no" {
		r.events.TriggerDataproductGrantAllUsers(ctx, dataproductID)
	} else {
		r.events.TriggerDataproductGrant(ctx, dataproductID, subject)
	}

	return accessFromSQL(access), nil
}

func (r *Repo) GetAccessToDataproduct(ctx context.Context, id uuid.UUID) (*models.Access, error) {
	access, err := r.querier.GetAccessToDataproduct(ctx, id)
	if err != nil {
		return nil, err
	}
	return accessFromSQL(access), nil
}

func (r *Repo) ListActiveAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]*models.Access, error) {
	access, err := r.querier.ListActiveAccessToDataproduct(ctx, dataproductID)
	if err != nil {
		return nil, err
	}

	var ret []*models.Access
	for _, e := range access {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

func (r *Repo) RevokeAccessToDataproduct(ctx context.Context, id uuid.UUID) error {
	if err := r.querier.RevokeAccessToDataproduct(ctx, id); err != nil {
		return err
	}

	access, err := r.GetAccessToDataproduct(ctx, id)
	if err != nil {
		return err
	}
	if access.Subject == "group:all-users@nav.no" {
		r.events.TriggerDataproductRevokeAllUsers(ctx, access.DataproductID)
	} else {
		r.events.TriggerDataproductRevoke(ctx, access.DataproductID, access.Subject)
	}

	return nil
}

func (r *Repo) GetUnrevokedExpiredAccess(ctx context.Context) ([]*models.Access, error) {
	expired, err := r.querier.ListUnrevokedExpiredAccessEntries(ctx)
	if err != nil {
		return nil, err
	}

	var ret []*models.Access
	for _, e := range expired {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

func accessFromSQL(access gensql.DataproductAccess) *models.Access {
	return &models.Access{
		ID:            access.ID,
		Subject:       access.Subject,
		Granter:       access.Granter,
		Expires:       nullTimeToPtr(access.Expires),
		Created:       access.Created,
		Revoked:       nullTimeToPtr(access.Revoked),
		DataproductID: access.DataproductID,
	}
}
