package api

import (
	"time"

	"github.com/google/uuid"
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
