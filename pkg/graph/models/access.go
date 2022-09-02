package models

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
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

type NewGrant struct {
	DatasetID   uuid.UUID    `json:"datasetID"`
	Expires     *time.Time   `json:"expires"`
	Subject     *string      `json:"subject"`
	SubjectType *SubjectType `json:"subjectType"`
}

type AccessRequest struct {
	ID          uuid.UUID           `json:"id"`
	DatasetID   uuid.UUID           `json:"datasetID"`
	Subject     string              `json:"subject"`
	SubjectType SubjectType         `json:"subjectType"`
	Created     time.Time           `json:"created"`
	Status      AccessRequestStatus `json:"status"`
	Closed      *time.Time          `json:"closed"`
	Expires     *time.Time          `json:"expires"`
	Granter     *string             `json:"granter"`
	Owner       string              `json:"owner"`
	Polly       *Polly              `json:"polly"`
	Reason      *string             `json:"reason"`
}

type NewAccessRequest struct {
	DatasetID   uuid.UUID    `json:"datasetID"`
	Subject     *string      `json:"subject"`
	SubjectType *SubjectType `json:"subjectType"`
	Owner       *string      `json:"owner"`
	Expires     *time.Time   `json:"expires"`
	Polly       *PollyInput  `json:"polly"`
}

type UpdateAccessRequest struct {
	ID      uuid.UUID   `json:"id"`
	Owner   string      `json:"owner"`
	Expires *time.Time  `json:"expires"`
	Polly   *PollyInput `json:"polly"`
}

type SubjectType string

const (
	SubjectTypeUser           SubjectType = "user"
	SubjectTypeGroup          SubjectType = "group"
	SubjectTypeServiceAccount SubjectType = "serviceAccount"
)

var AllSubjectType = []SubjectType{
	SubjectTypeUser,
	SubjectTypeGroup,
	SubjectTypeServiceAccount,
}

func StringToSubjectType(subjectType string) SubjectType {
	switch subjectType {
	case "user":
		return SubjectTypeUser
	case "group":
		return SubjectTypeGroup
	case "serviceaccount":
		return SubjectTypeServiceAccount
	default:
		return ""
	}
}

func (e SubjectType) IsValid() bool {
	switch e {
	case SubjectTypeUser, SubjectTypeGroup, SubjectTypeServiceAccount:
		return true
	}
	return false
}

func (e SubjectType) String() string {
	return string(e)
}

func (e *SubjectType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = SubjectType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid SubjectType", str)
	}
	return nil
}

func (e SubjectType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
