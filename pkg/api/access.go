package api

import (
	"context"
	"database/sql"
	"errors"
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

type NewAccessRequestDTO struct {
	DatasetID   uuid.UUID   `json:"datasetID"`
	Subject     *string     `json:"subject"`
	SubjectType *string     `json:"subjectType"`
	Owner       *string     `json:"owner"`
	Expires     *time.Time  `json:"expires"`
	Polly       *PollyInput `json:"polly"`
}

type UpdateAccessRequestDTO struct {
	ID      uuid.UUID   `json:"id"`
	Owner   string      `json:"owner"`
	Expires *time.Time  `json:"expires"`
	Polly   *PollyInput `json:"polly"`
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

const (
	SubjectTypeUser           string = "user"
	SubjectTypeGroup          string = "group"
	SubjectTypeServiceAccount string = "serviceAccount"
)

type GrantAccessData struct {
	DatasetID   uuid.UUID  `json:"datasetID"`
	Expires     *time.Time `json:"expires"`
	Subject     *string    `json:"subject"`
	SubjectType *string    `json:"subjectType"`
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

func createAccessRequest(ctx context.Context, input NewAccessRequestDTO) *APIError {
	user := auth.GetUser(ctx)
	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}

	owner := "user:" + user.Email
	if input.Owner != nil {
		owner = "group:" + *input.Owner
	}

	subjType := SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}

	subjWithType := subjType + ":" + subj

	var pollyID uuid.NullUUID
	if input.Polly != nil {
		dbPolly, err := createPollyDocumentation(ctx, *input.Polly)
		if err != nil {
			return NewAPIError(http.StatusInternalServerError, err, "createAccessRequest(): failed to create polly documentation")
		}

		pollyID = uuid.NullUUID{UUID: dbPolly.ID, Valid: true}
	}

	accessRequest, err := dbCreateAccessRequestForDataset(ctx, input.DatasetID, pollyID, subjWithType, owner, input.Expires)
	if err != nil {
		return DBErrorToAPIError(err, "createAccessRequest(): failed to create access request")
	}
	sendNewAccessRequestSlackNotification(ctx, accessRequest)
	return nil
}

func deleteAccessRequest(ctx context.Context, accessRequestID string) *APIError {
	accessRequestUUID, err := uuid.Parse(accessRequestID)
	if err != nil {
		return NewAPIError(http.StatusBadRequest, err, "deleteAccessRequest(): invalid accessRequestID")
	}

	accessRequest, apierr := getAccessRequest(ctx, accessRequestID)
	if apierr != nil {
		return apierr
	}

	splits := strings.Split(accessRequest.Owner, ":")
	if len(splits) != 2 {
		return NewAPIError(http.StatusInternalServerError, fmt.Errorf("%v is not a valid owner format (cannot split on :)", accessRequest.Owner),
			"deleteAccessRequest(): invalid owner format")
	}
	owner := splits[1]

	if err := ensureOwner(ctx, owner); err != nil {
		return NewAPIError(http.StatusForbidden, err, "deleteAccessRequest(): user is not owner")
	}

	if err := queries.DeleteAccessRequest(ctx, accessRequestUUID); err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "deleteAccessRequest(): failed to delete access request")
	}

	return nil
}

func updateAccessRequest(ctx context.Context, input UpdateAccessRequestDTO) *APIError {
	var pollyID uuid.NullUUID
	if input.Polly != nil {
		if input.Polly.ID != nil {
			// Keep existing polly
			pollyID = uuid.NullUUID{UUID: *input.Polly.ID, Valid: true}
		} else {
			dbPolly, err := createPollyDocumentation(ctx, *input.Polly)
			if err != nil {
				return NewAPIError(http.StatusInternalServerError, err, "updateAccessRequest(): failed to create polly documentation")
			}
			pollyID = uuid.NullUUID{UUID: dbPolly.ID, Valid: true}
		}
	}

	_, err := queries.UpdateAccessRequest(ctx, gensql.UpdateAccessRequestParams{
		Owner:                input.Owner,
		Expires:              ptrToNullTime(input.Expires),
		PollyDocumentationID: pollyID,
		ID:                   input.ID,
	})
	if err != nil {
		return DBErrorToAPIError(err, "updateAccessRequest(): failed to update access request")
	}
	return nil
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

func grantAccessToDataset(ctx context.Context, input GrantAccessData) *APIError {
	if input.Expires != nil && input.Expires.Before(time.Now()) {
		return NewAPIError(http.StatusBadRequest,
			fmt.Errorf("datoen tilgangen skal utløpe må være fram i tid"), "grantAccessToDataset(): invalid expires date")
	}

	user := auth.GetUser(ctx)
	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}
	ds, apierr := GetDataset(ctx, input.DatasetID.String())
	if apierr != nil {
		return NewAPIError(apierr.HttpStatus, apierr.Err, "grantAccessToDataset(): failed to get dataset")
	}

	dp, apierr := GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return NewAPIError(apierr.HttpStatus, apierr.Err, "grantAccessToDataset(): failed to get dataproduct")
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return NewAPIError(http.StatusForbidden, err, "grantAccessToDataset(): user is not member of owner group")
	}

	if ds.Pii == "sensitive" && subj == "all-users@nav.no" {
		return NewAPIError(http.StatusBadRequest,
			fmt.Errorf("datasett som inneholder personopplysninger kan ikke gjøres tilgjengelig for alle interne brukere (all-users@nav.no)"), "grantAccessToDataset(): illegal action")
	}

	bq, apierr := getBigqueryDatasource(ctx, ds.ID, false)
	if apierr != nil {
		return NewAPIError(http.StatusInternalServerError, apierr.Err, "grantAccessToDataset(): failed to get bigquery datasource")
	}

	subjType := SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}

	subjWithType := subjType + ":" + subj

	if len(bq.PseudoColumns) > 0 {
		joinableViews, err := getJoinableViewsForReferenceAndUser(ctx, subj, ds.ID)
		if err != nil {
			return NewAPIError(http.StatusInternalServerError, err, "grantAccessToDataset(): failed to get joinable views")
		}
		for _, jv := range joinableViews {
			joinableViewName := bigquery.MakeJoinableViewName(bq.ProjectID, bq.Dataset, bq.Table)
			if err := accessManager.Grant(ctx, config.Conf.CentralDataProject, jv.Dataset, joinableViewName, subjWithType); err != nil {
				return NewAPIError(http.StatusInternalServerError, err, "grantAccessToDataset(): failed to grant access to joinable views")
			}
		}
	}

	if err := accessManager.Grant(ctx, bq.ProjectID, bq.Dataset, bq.Table, subjWithType); err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "grantAccessToDataset(): failed to grant access")
	}

	err := dbGrantAccessToDataset(ctx, input.DatasetID, input.Expires, subjWithType, user.Email)
	if err != nil {
		return NewAPIError(http.StatusInternalServerError, err, "grantAccessToDataset(): failed to grant access")
	}
	eventManager.TriggerDatasetGrant(ctx, input.DatasetID, *input.Subject)
	return nil
}

func dbGrantAccessToDataset(ctx context.Context, datasetID uuid.UUID, expires *time.Time, subject, granter string) error {
	a, err := queries.GetActiveAccessToDatasetForSubject(ctx, gensql.GetActiveAccessToDatasetForSubjectParams{
		DatasetID: datasetID,
		Subject:   subject,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	tx, err := sqldb.Begin()
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Error("Rolling back grant access request transaction")
		}
	}()
	if err != nil {
		return err
	}

	querier := queries.WithTx(tx)

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

func dbCreateAccessRequestForDataset(ctx context.Context, datasetID uuid.UUID, pollyDocumentationID uuid.NullUUID, subject, owner string, expires *time.Time) (*AccessRequest, error) {
	requestSQL, err := queries.CreateAccessRequestForDataset(ctx, gensql.CreateAccessRequestForDatasetParams{
		DatasetID:            datasetID,
		Subject:              emailOfSubjectToLower(subject),
		Owner:                owner,
		Expires:              ptrToNullTime(expires),
		PollyDocumentationID: pollyDocumentationID,
	})
	if err != nil {
		return nil, err
	}
	return accessRequestFromSQL(ctx, requestSQL)
}

func sendNewAccessRequestSlackNotification(ctx context.Context, ar *AccessRequest) {
	ds, apierr := GetDataset(ctx, ar.DatasetID.String())
	if apierr != nil {
		log.Warn("Access request created but failed to fetch dataset during sending slack notification", apierr)
		return
	}

	dp, apierr := GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		log.Warn("Access request created but failed to fetch dataproduct during sending slack notification", apierr)
		return
	}

	if dp.Owner.TeamContact == nil || *dp.Owner.TeamContact == "" {
		log.Info("Access request created but skip slack message because teamcontact is empty")
		return
	}

	err := slackClient.InformNewAccessRequest(*dp.Owner.TeamContact, dp.ID.String(), dp.Name, ds.ID.String(), ds.Name, ar.Subject)
	if err != nil {
		log.Warn("Access request created, failed to send slack message", err)
	}
}
