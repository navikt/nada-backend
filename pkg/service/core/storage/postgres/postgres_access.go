package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
	"strings"
	"time"
)

type accessStorage struct {
	db *database.Repo
}

func (s *accessStorage) GetUnrevokedExpiredAccess(ctx context.Context) ([]*service.Access, error) {
	expired, err := s.db.Querier.ListUnrevokedExpiredAccessEntries(ctx)
	if err != nil {
		return nil, err
	}

	var ret []*service.Access
	for _, e := range expired {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

var _ service.AccessStorage = &accessStorage{}

func (s *accessStorage) ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*service.Access, error) {
	access, err := s.db.Querier.ListActiveAccessToDataset(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("list active access to dataset: %w", err)
	}

	var ret []*service.Access
	for _, e := range access {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

func (s *accessStorage) ListAccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]*service.AccessRequest, error) {
	accessRequestsSQL, err := s.db.Querier.ListAccessRequestsForDataset(ctx, datasetID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("list access requests for dataset: %w", err)
	}

	accessRequests, err := AccessRequestsFromSQL(accessRequestsSQL)
	if err != nil {
		return nil, fmt.Errorf("access requests from sql: %w", err)
	}

	return accessRequests, nil
}

func (s *accessStorage) CreateAccessRequestForDataset(ctx context.Context, datasetID uuid.UUID, pollyDocumentationID uuid.NullUUID, subject, owner string, expires *time.Time) (*service.AccessRequest, error) {
	requestSQL, err := s.db.Querier.CreateAccessRequestForDataset(ctx, gensql.CreateAccessRequestForDatasetParams{
		DatasetID:            datasetID,
		Subject:              emailOfSubjectToLower(subject),
		Owner:                owner,
		Expires:              ptrToNullTime(expires),
		PollyDocumentationID: pollyDocumentationID,
	})
	if err != nil {
		return nil, fmt.Errorf("create access request for dataset: %w", err)
	}

	return AccessRequestFromSQL(requestSQL)
}

func (s *accessStorage) GetAccessRequest(ctx context.Context, accessRequestID string) (*service.AccessRequest, error) {
	accessRequestUUID, err := uuid.Parse(accessRequestID)
	if err != nil {
		return nil, fmt.Errorf("parsing access request id: %w", err)
	}

	accessRequestsSQL, err := s.db.Querier.GetAccessRequest(ctx, accessRequestUUID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get access request: %w", err)
	} else if err != nil {
		return nil, fmt.Errorf("get access request: %w", err)
	}

	accessRequest, err := AccessRequestFromSQL(accessRequestsSQL)
	if err != nil {
		return nil, fmt.Errorf("access request from sql: %w", err)
	}

	return accessRequest, nil
}

func (s *accessStorage) DeleteAccessRequest(ctx context.Context, accessRequestID string) error {
	accessRequestUUID, err := uuid.Parse(accessRequestID)
	if err != nil {
		return fmt.Errorf("parsing access request id: %w", err)
	}

	if err := s.db.Querier.DeleteAccessRequest(ctx, accessRequestUUID); err != nil {
		return fmt.Errorf("delete access request: %w", err)
	}

	return nil
}

func (s *accessStorage) UpdateAccessRequest(ctx context.Context, input service.UpdateAccessRequestDTO) error {
	var pollyID uuid.NullUUID

	if input.Polly != nil && input.Polly.ID != nil {
		pollyID = uuid.NullUUID{UUID: *input.Polly.ID, Valid: true}
	}

	_, err := s.db.Querier.UpdateAccessRequest(ctx, gensql.UpdateAccessRequestParams{
		Owner:                input.Owner,
		Expires:              ptrToNullTime(input.Expires),
		PollyDocumentationID: pollyID,
		ID:                   input.ID,
	})
	if err != nil {
		return fmt.Errorf("update access request: %w", err)
	}

	return nil
}

func (s *accessStorage) GrantAccessToDatasetAndApproveRequest(ctx context.Context, datasetID, subject, accessRequestID string, expires *time.Time) (err error) {
	tx, err := s.db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			err = fmt.Errorf("rollback transaction: %w", err)
		}
	}()

	user := auth.GetUser(ctx)
	q := s.db.Querier.WithTx(tx)

	_, err = q.GrantAccessToDataset(ctx, gensql.GrantAccessToDatasetParams{
		DatasetID: uuid.MustParse(datasetID),
		Subject:   subject,
		Granter:   user.Email,
		Expires:   ptrToNullTime(expires),
		AccessRequestID: uuid.NullUUID{
			UUID:  uuid.MustParse(accessRequestID),
			Valid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("grant access to dataset: %w", err)
	}

	err = q.ApproveAccessRequest(ctx, gensql.ApproveAccessRequestParams{
		ID:      uuid.MustParse(accessRequestID),
		Granter: sql.NullString{String: user.Email, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("approve access request: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (s *accessStorage) GrantAccessToDatasetAndRenew(ctx context.Context, datasetID uuid.UUID, expires *time.Time, subject, granter string) (err error) {
	a, err := s.db.Querier.GetActiveAccessToDatasetForSubject(ctx, gensql.GetActiveAccessToDatasetForSubjectParams{
		DatasetID: datasetID,
		Subject:   subject,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	tx, err := s.db.GetDB().Begin()
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			err = fmt.Errorf("rollback transaction: %w", err)
		}
	}()
	if err != nil {
		return err
	}

	querier := s.db.Querier.WithTx(tx)

	if len(a.Subject) > 0 {
		if err := querier.RevokeAccessToDataset(ctx, a.ID); err != nil {
			return err
		}
	}

	_, err = querier.GrantAccessToDataset(ctx, gensql.GrantAccessToDatasetParams{
		DatasetID: datasetID,
		Subject:   emailOfSubjectToLower(subject),
		Expires:   ptrToNullTime(expires),
		Granter:   granter,
	})
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *accessStorage) DenyAccessRequest(ctx context.Context, accessRequestID string, reason *string) error {
	// FIXME: bah
	user := auth.GetUser(ctx)

	err := s.db.Querier.DenyAccessRequest(ctx, gensql.DenyAccessRequestParams{
		ID:      uuid.MustParse(accessRequestID),
		Granter: sql.NullString{String: user.Email, Valid: true},
		Reason:  ptrToNullString(reason),
	})
	if err != nil {
		return fmt.Errorf("deny access request: %w", err)
	}

	return nil
}

func (s *accessStorage) GetAccessToDataset(ctx context.Context, id uuid.UUID) (*service.Access, error) {
	access, err := s.db.Querier.GetAccessToDataset(ctx, id)
	if err != nil {
		return nil, err
	}
	return &service.Access{
		ID:              access.ID,
		Subject:         access.Subject,
		Granter:         access.Granter,
		Expires:         nullTimeToPtr(access.Expires),
		Created:         access.Created,
		Revoked:         nullTimeToPtr(access.Revoked),
		DatasetID:       access.DatasetID,
		AccessRequestID: nullUUIDToUUIDPtr(access.AccessRequestID),
	}, nil
}

func (s *accessStorage) RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error {
	err := s.db.Querier.RevokeAccessToDataset(ctx, id)
	if err != nil {
		return fmt.Errorf("revoke access to dataset: %w", err)
	}

	return nil
}

func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: *s, Valid: true}
}

// FIXME: move all of these into a helpers.go file
func ptrToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}

	return sql.NullTime{Time: *t, Valid: true}
}

func emailOfSubjectToLower(subectWithType string) string {
	parts := strings.Split(subectWithType, ":")
	parts[1] = strings.ToLower(parts[1])

	return strings.Join(parts, ":")
}

func accessFromSQL(access gensql.DatasetAccess) *service.Access {
	return &service.Access{
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

func AccessRequestsFromSQL(accessRequestSQLs []gensql.DatasetAccessRequest) ([]*service.AccessRequest, error) {
	var accessRequests []*service.AccessRequest
	for _, ar := range accessRequestSQLs {
		accessRequestGraphql, err := AccessRequestFromSQL(ar)
		if err != nil {
			return nil, err
		}
		accessRequests = append(accessRequests, accessRequestGraphql)
	}
	return accessRequests, nil
}

func AccessRequestFromSQL(dataproductAccessRequest gensql.DatasetAccessRequest) (*service.AccessRequest, error) {
	splits := strings.Split(dataproductAccessRequest.Subject, ":")
	if len(splits) != 2 {
		return nil, fmt.Errorf("%v is not a valid subject (can't split on :)", dataproductAccessRequest.Subject)
	}
	subject := splits[1]

	subjectType := splits[0]

	status, err := accessRequestStatusFromDB(dataproductAccessRequest.Status)
	if err != nil {
		return nil, err
	}

	var polly *service.Polly

	if dataproductAccessRequest.PollyDocumentationID.Valid {
		polly = &service.Polly{
			ID: dataproductAccessRequest.PollyDocumentationID.UUID,
		}
	}

	return &service.AccessRequest{
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

func accessRequestStatusFromDB(sqlStatus gensql.AccessRequestStatusType) (service.AccessRequestStatus, error) {
	switch sqlStatus {
	case gensql.AccessRequestStatusTypePending:
		return service.AccessRequestStatusPending, nil
	case gensql.AccessRequestStatusTypeApproved:
		return service.AccessRequestStatusApproved, nil
	case gensql.AccessRequestStatusTypeDenied:
		return service.AccessRequestStatusDenied, nil
	default:
		return "", fmt.Errorf("unknown access request status %q", sqlStatus)
	}
}

func NewAccessStorage(db *database.Repo) *accessStorage {
	return &accessStorage{
		db: db,
	}
}
