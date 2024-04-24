package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/config"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type Polly struct {
	ID uuid.UUID `json:"id"`
	QueryPolly
}

type PollyInput struct {
	ID *uuid.UUID `json:"id"`
	QueryPolly
}

type QueryPolly struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}

type Access struct {
	ID              uuid.UUID  `json:"id"`
	Subject         string     `json:"subject"`
	Granter         string     `json:"granter"`
	Expires         *time.Time `json:"expires"`
	Created         time.Time  `json:"created"`
	Revoked         *time.Time `json:"revoked"`
	DatasetID       uuid.UUID  `json:"datasetID"`
	AccessRequestID *uuid.UUID `json:"accessRequestID"`
}

type AccessRequest struct {
	ID          uuid.UUID           `json:"id"`
	DatasetID   uuid.UUID           `json:"datasetID"`
	Subject     string              `json:"subject"`
	SubjectType string              `json:"subjectType"`
	Created     time.Time           `json:"created"`
	Status      AccessRequestStatus `json:"status"`
	Closed      *time.Time          `json:"closed"`
	Expires     *time.Time          `json:"expires"`
	Granter     *string             `json:"granter"`
	Owner       string              `json:"owner"`
	Polly       *Polly              `json:"polly"`
	Reason      *string             `json:"reason"`
}

type AccessRequestForGranter struct {
	AccessRequest
	DataproductID   uuid.UUID `json:"dataproductID"`
	DataproductSlug string    `json:"dataproductSlug"`
	DatasetName     string    `json:"datasetName"`
	DataproductName string    `json:"dataproductName"`
}

type AccessRequestsWrapper struct {
	AccessRequests []AccessRequest `json:"accessRequests"`
}

func accessRequestsFromSQL(ctx context.Context, accessRequestSQLs []gensql.DatasetAccessRequest) ([]AccessRequest, error) {
	var accessRequests []AccessRequest
	for _, ar := range accessRequestSQLs {
		accessRequestGraphql, err := accessRequestFromSQL(ctx, ar)
		if err != nil {
			return nil, err
		}
		accessRequests = append(accessRequests, *accessRequestGraphql)
	}
	return accessRequests, nil
}

func accessRequestFromSQL(ctx context.Context, dataproductAccessRequest gensql.DatasetAccessRequest) (*AccessRequest, error) {
	splits := strings.Split(dataproductAccessRequest.Subject, ":")
	if len(splits) != 2 {
		return nil, fmt.Errorf("%v is not a valid subject (can't split on :)", dataproductAccessRequest.Subject)
	}
	subject := splits[1]

	subjectType := splits[0]

	polly, err := pollySQLToGraphql(ctx, dataproductAccessRequest.PollyDocumentationID)
	if err != nil {
		return nil, err
	}

	status, err := accessRequestStatusFromDB(dataproductAccessRequest.Status)
	if err != nil {
		return nil, err
	}

	return &AccessRequest{
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

type AccessRequestStatus string

const (
	AccessRequestStatusPending  AccessRequestStatus = "pending"
	AccessRequestStatusApproved AccessRequestStatus = "approved"
	AccessRequestStatusDenied   AccessRequestStatus = "denied"
)

func accessRequestStatusFromDB(sqlStatus gensql.AccessRequestStatusType) (AccessRequestStatus, error) {
	switch sqlStatus {
	case gensql.AccessRequestStatusTypePending:
		return AccessRequestStatusPending, nil
	case gensql.AccessRequestStatusTypeApproved:
		return AccessRequestStatusApproved, nil
	case gensql.AccessRequestStatusTypeDenied:
		return AccessRequestStatusDenied, nil
	default:
		return "", fmt.Errorf("unknown access request status %q", sqlStatus)
	}
}

func getAccessRequests(ctx context.Context, datasetID string) (*AccessRequestsWrapper, *APIError) {
	datasetUUID, err := uuid.Parse(datasetID)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "invalid datasetID")
	}

	accessRequestsSQL, err := queries.ListAccessRequestsForDataset(ctx, datasetUUID)
	if err != nil && err != sql.ErrNoRows {
		return nil, DBErrorToAPIError(err, "getAccessRequests(): failed to get access requests")
	}

	accessRequests, err := accessRequestsFromSQL(ctx, accessRequestsSQL)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "getAccessRequests(): converting access requests from database")
	}
	return &AccessRequestsWrapper{
		AccessRequests: accessRequests,
	}, nil
}

func getAccessRequest(ctx context.Context, accessRequestID string) (*AccessRequest, *APIError) {
	accessRequestUUID, err := uuid.Parse(accessRequestID)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "invalid accessRequestID")
	}

	accessRequestsSQL, err := queries.GetAccessRequest(ctx, accessRequestUUID)
	if err == sql.ErrNoRows {
		return nil, NewAPIError(http.StatusNotFound, err, "getAccessRequest(): access request not found")
	} else if err != nil {
		return nil, DBErrorToAPIError(err, "getAccessRequest(): failed to get access requests")
	}

	accessRequest, err := accessRequestFromSQL(ctx, accessRequestsSQL)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "getAccessRequest(): converting access request from database")
	}
	return accessRequest, nil
}

func approveAccessRequest(ctx context.Context, accessRequestID string) *APIError {
	ar, apierr := getAccessRequest(ctx, accessRequestID)
	if apierr != nil {
		return apierr
	}

	ds, apierr := GetDataset(ctx, ar.DatasetID.String())
	if apierr != nil {
		return apierr
	}

	bq, apiError := getBigqueryDatasource(ctx, ds.ID, false)
	if apiError != nil {
		return apiError
	}

	dp, apiError := GetDataproduct(ctx, ds.DataproductID.String())
	if apiError != nil {
		return apiError
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return NewAPIError(http.StatusForbidden, err, "approveAccessRequest(): user is not member of owner group")
	}

	if ds.Pii == "sensitive" && ar.Subject == "all-users@nav.no" {
		return NewAPIError(http.StatusForbidden,
			fmt.Errorf("datasett som inneholder personopplysninger kan ikke gjøres tilgjengelig for alle interne brukere (all-users@nav.no)"),
			"approveAccessRequest() illegal action")
	}

	subjWithType := ar.SubjectType + ":" + ar.Subject
	if err := accessManager.Grant(ctx, bq.ProjectID, bq.Dataset, bq.Table, subjWithType); err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "approveAccessRequest(): failed to grant access")
	}

	tx, err := sqldb.Begin()
	if err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "approveAccessRequest(): failed to start transaction")
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Error("Rolling back grant access request transaction")
		}
	}()

	user := auth.GetUser(ctx)
	fmt.Println(ar.Subject)
	q := queries.WithTx(tx)

	_, err = q.GrantAccessToDataset(ctx, gensql.GrantAccessToDatasetParams{
		DatasetID:       ar.DatasetID,
		Subject:         subjWithType,
		Granter:         user.Email,
		Expires:         ptrToNullTime(ar.Expires),
		AccessRequestID: uuid.NullUUID{UUID: ar.ID, Valid: true},
	})
	if err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "approveAccessRequest(): failed to grant access")
	}

	err = q.ApproveAccessRequest(ctx, gensql.ApproveAccessRequestParams{
		ID:      ar.ID,
		Granter: sql.NullString{String: user.Email, Valid: true},
	})
	if err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "approveAccessRequest(): failed to approve access request")
	}

	if err := tx.Commit(); err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "approveAccessRequest(): failed to commit transaction")
	}

	return nil
}

func denyAccessRequest(ctx context.Context, accessRequestID string, reason *string) *APIError {
	ar, apierr := getAccessRequest(ctx, accessRequestID)
	if apierr != nil {
		return apierr
	}

	ds, apierr := GetDataset(ctx, ar.DatasetID.String())
	if apierr != nil {
		return apierr
	}

	dp, apierr := GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return apierr
	}

	err := ensureUserInGroup(ctx, dp.Owner.Group)
	if err != nil {
		return NewAPIError(http.StatusForbidden, err, "denyAccessRequest(): user is not member of owner group")
	}

	user := auth.GetUser(ctx)

	err = queries.DenyAccessRequest(ctx, gensql.DenyAccessRequestParams{
		ID:      uuid.MustParse(accessRequestID),
		Granter: sql.NullString{String: user.Email, Valid: true},
		Reason:  ptrToNullString(reason),
	})

	if err != nil {
		return DBErrorToAPIError(err, "denyAccessRequest(): failed to deny access request")
	}

	return nil
}

// id is the id of dataset_access table
func revokeAccessToDataset(ctx context.Context, id string) *APIError {
	accessID, err := uuid.Parse(id)
	if err != nil {
		return NewAPIError(http.StatusBadRequest, err, "revokeAccessToDataset(): invalid accessID")
	}
	access, err := getAccessToDataset(ctx, accessID)
	if err != nil {
		return DBErrorToAPIError(err, "revokeAccessToDataset(): failed to get dataset access")
	}

	ds, apierr := GetDataset(ctx, access.DatasetID.String())
	if apierr != nil {
		return apierr
	}

	dp, apierr := GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return apierr
	}

	bq, apierr := getBigqueryDatasource(ctx, access.DatasetID, false)
	if apierr != nil {
		return apierr
	}

	user := auth.GetUser(ctx)
	err = ensureUserInGroup(ctx, dp.Owner.Group)
	if err != nil && !strings.EqualFold("user:"+user.Email, access.Subject) {
		return NewAPIError(http.StatusForbidden, err, "revokeAccessToDataset(): user is not member of owner group or the subject")
	}

	subjectParts := strings.Split(access.Subject, ":")
	if len(subjectParts) != 2 {
		return NewAPIError(http.StatusBadRequest, fmt.Errorf("invalid subject %q", access.Subject), "revokeAccessToDataset(): invalid subject")
	}

	subjectWithoutType := subjectParts[1]

	if len(bq.PseudoColumns) > 0 {
		joinableViews, err := getJoinableViewsForReferenceAndUser(ctx, subjectWithoutType, ds.ID)
		if err != nil {
			return DBErrorToAPIError(err, "revokeAccessToDataset(): failed to get joinable views")
		}
		for _, jv := range joinableViews {
			joinableViewName := bigquery.MakeJoinableViewName(bq.ProjectID, bq.Dataset, bq.Table)
			if err := accessManager.Revoke(ctx, config.Conf.CentralDataProject, jv.Dataset, joinableViewName, access.Subject); err != nil {
				return NewAPIError(http.StatusInternalServerError, err, "revokeAccessToDataset(): failed to revoke access")
			}
		}
	}

	if err := accessManager.Revoke(ctx, bq.ProjectID, bq.Dataset, bq.Table, access.Subject); err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "revokeAccessToDataset(): failed to revoke access")
	}

	if err := queries.RevokeAccessToDataset(ctx, accessID); err != nil {
		return DBErrorToAPIError(err, "revokeAccessToDataset(): failed to revoke access")
	}

	eventManager.TriggerDatasetRevoke(ctx, access.DatasetID, access.Subject)
	return nil
}

func getAccessToDataset(ctx context.Context, id uuid.UUID) (*Access, error) {
	access, err := queries.GetAccessToDataset(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Access{
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
