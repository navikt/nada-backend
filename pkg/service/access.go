package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AccessStorage interface {
	ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*Access, error)
	ListAccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]*AccessRequest, error)
	CreateAccessRequestForDataset(ctx context.Context, datasetID uuid.UUID, pollyDocumentationID uuid.NullUUID, subject, owner string, expires *time.Time) (*AccessRequest, error)
	GetAccessRequest(ctx context.Context, accessRequestID string) (*AccessRequest, error)
	DeleteAccessRequest(ctx context.Context, accessRequestID string) error
	UpdateAccessRequest(ctx context.Context, input UpdateAccessRequestDTO) error
	GrantAccessToDatasetAndApproveRequest(ctx context.Context, datasetID, subject, accessRequestID string, expires *time.Time) error
	GrantAccessToDatasetAndRenew(ctx context.Context, datasetID uuid.UUID, expires *time.Time, subject, granter string) error
	DenyAccessRequest(ctx context.Context, accessRequestID string, reason *string) error
	GetAccessToDataset(ctx context.Context, id uuid.UUID) (*Access, error)
	RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error
	GetUnrevokedExpiredAccess(ctx context.Context) ([]*Access, error)
}

type AccessService interface {
	GetAccessRequests(ctx context.Context, datasetID string) (*AccessRequestsWrapper, error)
	CreateAccessRequest(ctx context.Context, input NewAccessRequestDTO) error
	DeleteAccessRequest(ctx context.Context, accessRequestID string) error
	UpdateAccessRequest(ctx context.Context, input UpdateAccessRequestDTO) error
	ApproveAccessRequest(ctx context.Context, accessRequestID string) error
	DenyAccessRequest(ctx context.Context, accessRequestID string, reason *string) error
	RevokeAccessToDataset(ctx context.Context, id, gcpProjectID string) error
	GrantAccessToDataset(ctx context.Context, input GrantAccessData, gcpProjectID string) error
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
	AccessRequests []*AccessRequest `json:"accessRequests"`
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

type AccessRequestStatus string

const (
	AccessRequestStatusPending  AccessRequestStatus = "pending"
	AccessRequestStatusApproved AccessRequestStatus = "approved"
	AccessRequestStatusDenied   AccessRequestStatus = "denied"
)
