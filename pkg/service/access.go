package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AccessStorage interface {
	CreateAccessRequestForDataset(ctx context.Context, datasetID uuid.UUID, pollyDocumentationID uuid.NullUUID, subject, owner string, expires *time.Time) (*AccessRequest, error)
	DeleteAccessRequest(ctx context.Context, accessRequestID uuid.UUID) error
	DenyAccessRequest(ctx context.Context, user *User, accessRequestID uuid.UUID, reason *string) error
	GetAccessRequest(ctx context.Context, accessRequestID uuid.UUID) (*AccessRequest, error)
	GetAccessToDataset(ctx context.Context, id uuid.UUID) (*Access, error)
	GetUnrevokedExpiredAccess(ctx context.Context) ([]*Access, error)
	GrantAccessToDatasetAndApproveRequest(ctx context.Context, user *User, datasetID uuid.UUID, subject, accessRequestOwner string, accessRequestID uuid.UUID, expires *time.Time) error
	GrantAccessToDatasetAndRenew(ctx context.Context, datasetID uuid.UUID, expires *time.Time, subject, owner, granter string) error
	ListAccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]*AccessRequest, error)
	ListAccessRequestsForOwner(ctx context.Context, owner []string) ([]*AccessRequest, error)
	ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*Access, error)
	RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error
	UpdateAccessRequest(ctx context.Context, input UpdateAccessRequestDTO) error
}

type AccessService interface {
	GetAccessRequests(ctx context.Context, datasetID uuid.UUID) (*AccessRequestsWrapper, error)
	CreateAccessRequest(ctx context.Context, user *User, input NewAccessRequestDTO) error
	DeleteAccessRequest(ctx context.Context, user *User, accessRequestID uuid.UUID) error
	UpdateAccessRequest(ctx context.Context, input UpdateAccessRequestDTO) error
	ApproveAccessRequest(ctx context.Context, user *User, accessRequestID uuid.UUID) error
	DenyAccessRequest(ctx context.Context, user *User, accessRequestID uuid.UUID, reason *string) error
	RevokeAccessToDataset(ctx context.Context, user *User, id uuid.UUID, gcpProjectID string) error
	GrantAccessToDataset(ctx context.Context, user *User, input GrantAccessData, gcpProjectID string) error
}

type Access struct {
	ID              uuid.UUID  `json:"id"`
	Subject         string     `json:"subject"`
	Owner           string     `json:"owner"`
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
	Owner       *string    `json:"owner"`
	SubjectType *string    `json:"subjectType"`
}

type AccessRequestStatus string

const (
	AccessRequestStatusPending  AccessRequestStatus = "pending"
	AccessRequestStatusApproved AccessRequestStatus = "approved"
	AccessRequestStatusDenied   AccessRequestStatus = "denied"
)
