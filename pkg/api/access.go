package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
