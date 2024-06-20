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
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"strings"
	"time"
)

type AccessQueries interface {
	ListAccessRequestsForOwner(ctx context.Context, owner []string) ([]gensql.DatasetAccessRequest, error)
	ListUnrevokedExpiredAccessEntries(ctx context.Context) ([]gensql.DatasetAccess, error)
	ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]gensql.DatasetAccess, error)
	ListAccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]gensql.DatasetAccessRequest, error)
	CreateAccessRequestForDataset(ctx context.Context, params gensql.CreateAccessRequestForDatasetParams) (gensql.DatasetAccessRequest, error)
	GetAccessRequest(ctx context.Context, id uuid.UUID) (gensql.DatasetAccessRequest, error)
	DeleteAccessRequest(ctx context.Context, id uuid.UUID) error
	UpdateAccessRequest(ctx context.Context, params gensql.UpdateAccessRequestParams) (gensql.DatasetAccessRequest, error)
	GrantAccessToDataset(ctx context.Context, params gensql.GrantAccessToDatasetParams) (gensql.DatasetAccess, error)
	ApproveAccessRequest(ctx context.Context, params gensql.ApproveAccessRequestParams) error
	GetActiveAccessToDatasetForSubject(ctx context.Context, params gensql.GetActiveAccessToDatasetForSubjectParams) (gensql.DatasetAccess, error)
	RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error
	DenyAccessRequest(ctx context.Context, params gensql.DenyAccessRequestParams) error
	GetAccessToDataset(ctx context.Context, id uuid.UUID) (gensql.DatasetAccess, error)
}

var _ service.AccessStorage = &accessStorage{}

type AccessQueriesWithTxFn func() (AccessQueries, database.Transacter, error)

type accessStorage struct {
	queries  AccessQueries
	withTxFn AccessQueriesWithTxFn
}

func (s *accessStorage) ListAccessRequestsForOwner(ctx context.Context, owner []string) ([]*service.AccessRequest, error) {
	const op errs.Op = "accessStorage.ListAccessRequestsForOwner"

	accessRequest, err := s.queries.ListAccessRequestsForOwner(ctx, owner)
	if err != nil {
		return nil, errs.E(errs.Database, op, err, errs.Parameter("owner"))
	}

	accessRequests, err := accessRequestsFromSQL(accessRequest)
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return accessRequests, nil
}

func (s *accessStorage) GetUnrevokedExpiredAccess(ctx context.Context) ([]*service.Access, error) {
	const op errs.Op = "accessStorage.GetUnrevokedExpiredAccess"

	expired, err := s.queries.ListUnrevokedExpiredAccessEntries(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	var ret []*service.Access
	for _, e := range expired {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

func (s *accessStorage) ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*service.Access, error) {
	const op errs.Op = "accessStorage.ListActiveAccessToDataset"

	access, err := s.queries.ListActiveAccessToDataset(ctx, datasetID)
	if err != nil {
		return nil, errs.E(errs.Database, op, err, errs.Parameter("datasetID"))
	}

	var ret []*service.Access
	for _, e := range access {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

func (s *accessStorage) ListAccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]*service.AccessRequest, error) {
	const op errs.Op = "accessStorage.ListAccessRequestsForDataset"

	accessRequestsSQL, err := s.queries.ListAccessRequestsForDataset(ctx, datasetID)
	if err != nil {
		return nil, errs.E(errs.Database, op, err, errs.Parameter("datasetID"))
	}

	accessRequests, err := accessRequestsFromSQL(accessRequestsSQL)
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return accessRequests, nil
}

func (s *accessStorage) CreateAccessRequestForDataset(ctx context.Context, datasetID uuid.UUID, pollyDocumentationID uuid.NullUUID, subject, owner string, expires *time.Time) (*service.AccessRequest, error) {
	const op errs.Op = "accessStorage.CreateAccessRequestForDataset"

	requestSQL, err := s.queries.CreateAccessRequestForDataset(ctx, gensql.CreateAccessRequestForDatasetParams{
		DatasetID:            datasetID,
		Subject:              emailOfSubjectToLower(subject),
		Owner:                owner,
		Expires:              ptrToNullTime(expires),
		PollyDocumentationID: pollyDocumentationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err, errs.Parameter("datasetID"))
		}

		return nil, errs.E(errs.Database, op, err)
	}

	ar, err := accessRequestFromSQL(requestSQL)
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return ar, nil
}

func (s *accessStorage) GetAccessRequest(ctx context.Context, accessRequestID string) (*service.AccessRequest, error) {
	const op errs.Op = "accessStorage.GetAccessRequest"

	id, err := uuid.Parse(accessRequestID)
	if err != nil {
		return nil, errs.E(errs.Validation, op, err, errs.Parameter("accessRequestID"))
	}

	accessRequestsSQL, err := s.queries.GetAccessRequest(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err, errs.Parameter("accessRequestID"))
		}

		return nil, errs.E(errs.Database, op, err)
	}

	accessRequest, err := accessRequestFromSQL(accessRequestsSQL)
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return accessRequest, nil
}

func (s *accessStorage) DeleteAccessRequest(ctx context.Context, accessRequestID string) error {
	const op errs.Op = "accessStorage.DeleteAccessRequest"

	id, err := uuid.Parse(accessRequestID)
	if err != nil {
		return errs.E(errs.Validation, op, err, errs.Parameter("accessRequestID"))
	}

	err = s.queries.DeleteAccessRequest(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *accessStorage) UpdateAccessRequest(ctx context.Context, input service.UpdateAccessRequestDTO) error {
	const op errs.Op = "accessStorage.UpdateAccessRequest"

	var pollyID uuid.NullUUID

	if input.Polly != nil && input.Polly.ID != nil {
		pollyID = uuid.NullUUID{UUID: *input.Polly.ID, Valid: true}
	}

	_, err := s.queries.UpdateAccessRequest(ctx, gensql.UpdateAccessRequestParams{
		Owner:                input.Owner,
		Expires:              ptrToNullTime(input.Expires),
		PollyDocumentationID: pollyID,
		ID:                   input.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errs.E(errs.NotExist, op, err)
		}

		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *accessStorage) GrantAccessToDatasetAndApproveRequest(ctx context.Context, datasetID, subject, accessRequestID string, expires *time.Time) error {
	const op errs.Op = "accessStorage.GrantAccessToDatasetAndApproveRequest"

	q, tx, err := s.withTxFn()
	if err != nil {
		return errs.E(errs.Database, op, err)
	}
	defer tx.Rollback()

	// FIXME: move this up the call chain
	user := auth.GetUser(ctx)

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
		if errors.Is(err, sql.ErrNoRows) {
			return errs.E(errs.NotExist, op, err)
		}

		return errs.E(errs.Database, op, err)
	}

	err = q.ApproveAccessRequest(ctx, gensql.ApproveAccessRequestParams{
		ID:      uuid.MustParse(accessRequestID),
		Granter: sql.NullString{String: user.Email, Valid: true},
	})
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	err = tx.Commit()
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *accessStorage) GrantAccessToDatasetAndRenew(ctx context.Context, datasetID uuid.UUID, expires *time.Time, subject, granter string) (err error) {
	const op errs.Op = "accessStorage.GrantAccessToDatasetAndRenew"

	a, err := s.queries.GetActiveAccessToDatasetForSubject(ctx, gensql.GetActiveAccessToDatasetForSubjectParams{
		DatasetID: datasetID,
		Subject:   subject,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errs.E(errs.Database, op, err)
	}

	q, tx, err := s.withTxFn()
	if err != nil {
		return errs.E(errs.Database, op, err)
	}
	defer tx.Rollback()

	if len(a.Subject) > 0 {
		if err := q.RevokeAccessToDataset(ctx, a.ID); err != nil {
			return errs.E(errs.Database, op, err)
		}
	}

	_, err = q.GrantAccessToDataset(ctx, gensql.GrantAccessToDatasetParams{
		DatasetID: datasetID,
		Subject:   emailOfSubjectToLower(subject),
		Expires:   ptrToNullTime(expires),
		Granter:   granter,
	})
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	err = tx.Commit()
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *accessStorage) DenyAccessRequest(ctx context.Context, accessRequestID string, reason *string) error {
	const op errs.Op = "accessStorage.DenyAccessRequest"

	// FIXME: move up the invocation chain
	user := auth.GetUser(ctx)

	err := s.queries.DenyAccessRequest(ctx, gensql.DenyAccessRequestParams{
		ID:      uuid.MustParse(accessRequestID),
		Granter: sql.NullString{String: user.Email, Valid: true},
		Reason:  ptrToNullString(reason),
	})
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *accessStorage) GetAccessToDataset(ctx context.Context, id uuid.UUID) (*service.Access, error) {
	const op errs.Op = "accessStorage.GetAccessToDataset"

	access, err := s.queries.GetAccessToDataset(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err, errs.Parameter("id"))
		}

		return nil, errs.E(errs.Database, op, err, errs.Parameter("id"))
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
	const op errs.Op = "accessStorage.RevokeAccessToDataset"

	err := s.queries.RevokeAccessToDataset(ctx, id)
	if err != nil {
		return errs.E(errs.Database, op, err)
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

func emailOfSubjectToLower(subjectWithType string) string {
	parts := strings.Split(subjectWithType, ":")
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

func accessRequestsFromSQL(accessRequestSQLs []gensql.DatasetAccessRequest) ([]*service.AccessRequest, error) {
	const op errs.Op = "accessStorage.accessRequestsFromSQL"

	var accessRequests []*service.AccessRequest

	for _, ar := range accessRequestSQLs {
		accessRequestGraphql, err := accessRequestFromSQL(ar)
		if err != nil {
			return nil, errs.E(op, err)
		}

		accessRequests = append(accessRequests, accessRequestGraphql)
	}

	return accessRequests, nil
}

func accessRequestFromSQL(dataproductAccessRequest gensql.DatasetAccessRequest) (*service.AccessRequest, error) {
	const op errs.Op = "accessStorage.accessRequestFromSQL"

	splits := strings.Split(dataproductAccessRequest.Subject, ":")
	if len(splits) != 2 {
		return nil, errs.E(op, fmt.Errorf("%v is not a valid subject (can't split on :)", dataproductAccessRequest.Subject))
	}
	subject := splits[1]

	subjectType := splits[0]

	status, err := accessRequestStatusFromDB(dataproductAccessRequest.Status)
	if err != nil {
		return nil, errs.E(op, err)
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

func accessRequestStatusFromDB(status gensql.AccessRequestStatusType) (service.AccessRequestStatus, error) {
	const op errs.Op = "accessStorage.accessRequestStatusFromDB"

	switch status {
	case gensql.AccessRequestStatusTypePending:
		return service.AccessRequestStatusPending, nil
	case gensql.AccessRequestStatusTypeApproved:
		return service.AccessRequestStatusApproved, nil
	case gensql.AccessRequestStatusTypeDenied:
		return service.AccessRequestStatusDenied, nil
	default:
		return "", errs.E(op, fmt.Errorf("unknown access request status %q", status))
	}
}

func NewAccessStorage(queries AccessQueries, fn AccessQueriesWithTxFn) *accessStorage {
	return &accessStorage{
		withTxFn: fn,
		queries:  queries,
	}
}
