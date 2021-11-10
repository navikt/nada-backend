package models

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type Dataproduct struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Description  *string   `json:"description"`
	Slug         string    `json:"slug"`
	Repo         *string   `json:"repo"`
	Pii          bool      `json:"pii"`
	Keywords     []string  `json:"keywords"`
	Owner        *Owner    `json:"owner"`
	Type         gensql.DatasourceType
}

func (Dataproduct) IsCollectionElement() {}
func (Dataproduct) IsSearchResult()      {}

type Datasource interface {
	IsDatasource()
}

type BigQuery struct {
	DataproductID uuid.UUID
	ProjectID     string       `json:"projectID"`
	Dataset       string       `json:"dataset"`
	Table         string       `json:"table"`
	TableType     BigQueryType `json:"tableType"`
	LastModified  time.Time    `json:"lastModified"`
	Created       time.Time    `json:"created"`
	Expires       *time.Time   `json:"expired"`
}

func (BigQuery) IsDatasource() {}

type NewBigQuery struct {
	ProjectID string `json:"projectID"`
	Dataset   string `json:"dataset"`
	Table     string `json:"table"`
}

type NewDataproduct struct {
	Name        string      `json:"name"`
	Description *string     `json:"description"`
	Slug        *string     `json:"slug"`
	Repo        *string     `json:"repo"`
	Pii         bool        `json:"pii"`
	Keywords    []string    `json:"keywords"`
	Group       string      `json:"group"`
	BigQuery    NewBigQuery `json:"bigquery"`
	Requesters  []string    `json:"requesters"`
	Metadata    BigqueryMetadata
}

type UpdateDataproduct struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Slug        *string  `json:"slug"`
	Repo        *string  `json:"repo"`
	Pii         bool     `json:"pii"`
	Keywords    []string `json:"keywords"`
	Requesters  []string `json:"requesters"`
}

type Access struct {
	ID            uuid.UUID  `json:"id"`
	Subject       string     `json:"subject"`
	Granter       string     `json:"granter"`
	Expires       *time.Time `json:"expires"`
	Created       time.Time  `json:"created"`
	Revoked       *time.Time `json:"revoked"`
	DataproductID uuid.UUID
}

type Owner struct {
	Group         string `json:"group"`
	Teamkatalogen string `json:"teamkatalogen"`
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