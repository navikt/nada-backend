package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateAccessRequestForDataset(ctx context.Context, datasetID uuid.UUID, pollyDocumentationID uuid.NullUUID, subject, owner string, expires *time.Time) (*models.AccessRequest, error) {
	accessRequestSQL, err := r.querier.CreateAccessRequestForDataset(ctx, gensql.CreateAccessRequestForDatasetParams{
		DatasetID:            datasetID,
		Subject:              emailOfSubjectToLower(subject),
		Owner:                owner,
		Expires:              ptrToNullTime(expires),
		PollyDocumentationID: pollyDocumentationID,
	})
	if err != nil {
		return nil, err
	}

	return r.accessRequestSQLToGraphql(ctx, accessRequestSQL)
}

func (r *Repo) ListAccessRequestsForOwner(ctx context.Context, owners []string) ([]*models.AccessRequest, error) {
	accessRequestSQLs, err := r.querier.ListAccessRequestsForOwner(ctx, owners)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return r.accessRequestSQLsToGraphql(ctx, accessRequestSQLs)
}

func (r *Repo) ListAccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]*models.AccessRequest, error) {
	accessRequestSQLs, err := r.querier.ListAccessRequestsForDataset(ctx, datasetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return r.accessRequestSQLsToGraphql(ctx, accessRequestSQLs)
}

func (r *Repo) GetAccessRequest(ctx context.Context, id uuid.UUID) (*models.AccessRequest, error) {
	dataproductAccessRequest, err := r.querier.GetAccessRequest(ctx, id)
	if err != nil {
		return nil, err
	}

	return r.accessRequestSQLToGraphql(ctx, dataproductAccessRequest)
}

func (r *Repo) DenyAccessRequest(ctx context.Context, id uuid.UUID, granter string, reason *string) error {
	return r.querier.DenyAccessRequest(ctx, gensql.DenyAccessRequestParams{
		ID:      id,
		Granter: sql.NullString{String: granter, Valid: true},
		Reason:  ptrToNullString(reason),
	})
}

func (r *Repo) ApproveAccessRequest(ctx context.Context, id uuid.UUID, granter string) error {
	ar, err := r.querier.GetAccessRequest(ctx, id)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	querier := r.querier.WithTx(tx)

	_, err = querier.GrantAccessToDataset(ctx, gensql.GrantAccessToDatasetParams{
		DatasetID:       ar.DatasetID,
		Subject:         emailOfSubjectToLower(ar.Subject),
		Granter:         granter,
		Expires:         ar.Expires,
		AccessRequestID: uuid.NullUUID{UUID: ar.ID, Valid: true},
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("Rolling back grant access request transaction")
		}
		return err
	}

	err = querier.ApproveAccessRequest(ctx, gensql.ApproveAccessRequestParams{
		ID:      id,
		Granter: sql.NullString{String: granter, Valid: true},
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("Rolling back grant access request transaction")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *Repo) UpdateAccessRequest(ctx context.Context, id uuid.UUID, pollyID uuid.NullUUID, owner string, expires *time.Time) (*models.AccessRequest, error) {
	accessRequestSQL, err := r.querier.UpdateAccessRequest(ctx, gensql.UpdateAccessRequestParams{
		Owner:                owner,
		Expires:              ptrToNullTime(expires),
		PollyDocumentationID: pollyID,
		ID:                   id,
	})
	if err != nil {
		return nil, err
	}

	return r.accessRequestSQLToGraphql(ctx, accessRequestSQL)
}

func (r *Repo) DeleteAccessRequest(ctx context.Context, id uuid.UUID) error {
	return r.querier.DeleteAccessRequest(ctx, id)
}

func (r *Repo) accessRequestSQLsToGraphql(ctx context.Context, accessRequestSQLs []gensql.DatasetAccessRequest) ([]*models.AccessRequest, error) {
	var accessRequests []*models.AccessRequest
	for _, ar := range accessRequestSQLs {
		accessRequestGraphql, err := r.accessRequestSQLToGraphql(ctx, ar)
		if err != nil {
			return nil, err
		}
		accessRequests = append(accessRequests, accessRequestGraphql)
	}
	return accessRequests, nil
}

func (r *Repo) accessRequestSQLToGraphql(ctx context.Context, dataproductAccessRequest gensql.DatasetAccessRequest) (*models.AccessRequest, error) {
	splits := strings.Split(dataproductAccessRequest.Subject, ":")
	if len(splits) != 2 {
		return nil, fmt.Errorf("%v is not a valid subject (can't split on :)", dataproductAccessRequest.Subject)
	}
	subject := splits[1]

	subjectType := models.StringToSubjectType(splits[0])

	polly, err := r.pollySQLToGraphql(ctx, dataproductAccessRequest.PollyDocumentationID)
	if err != nil {
		return nil, err
	}

	status, err := accessRequestStatusToGraphql(dataproductAccessRequest.Status)
	if err != nil {
		return nil, err
	}

	return &models.AccessRequest{
		ID:          dataproductAccessRequest.ID,
		DatasetID:   dataproductAccessRequest.DatasetID,
		Subject:     subject,
		SubjectType: subjectType,
		Created:     dataproductAccessRequest.Created,
		Status:      status,
		Closed:      nullTimeToPtr(dataproductAccessRequest.Closed),
		Expires:     nullTimeToPtr(dataproductAccessRequest.Expires),
		Granter:     nullStringToPtr(dataproductAccessRequest.Granter),
		Owner:       dataproductAccessRequest.Owner,
		Polly:       polly,
		Reason:      nullStringToPtr(dataproductAccessRequest.Reason),
	}, nil
}

func (r *Repo) pollySQLToGraphql(ctx context.Context, id uuid.NullUUID) (*models.Polly, error) {
	if !id.Valid {
		return nil, nil
	}

	pollyDoc, err := r.querier.GetPollyDocumentation(ctx, id.UUID)
	if err != nil {
		return nil, err
	}

	return &models.Polly{
		ID: pollyDoc.ID,
		QueryPolly: models.QueryPolly{
			ExternalID: pollyDoc.ExternalID,
			Name:       pollyDoc.Name,
			URL:        pollyDoc.Url,
		},
	}, nil
}

func accessRequestStatusToGraphql(sqlStatus gensql.AccessRequestStatusType) (models.AccessRequestStatus, error) {
	switch sqlStatus {
	case gensql.AccessRequestStatusTypePending:
		return models.AccessRequestStatusPending, nil
	case gensql.AccessRequestStatusTypeApproved:
		return models.AccessRequestStatusApproved, nil
	case gensql.AccessRequestStatusTypeDenied:
		return models.AccessRequestStatusDenied, nil
	default:
		return "", fmt.Errorf("unknown access request status %q", sqlStatus)
	}
}
