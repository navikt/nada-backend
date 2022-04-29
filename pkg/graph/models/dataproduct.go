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

func (Dataproduct) IsSearchResult() {}

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
	Description   string       `json:"description"`
}

func (BigQuery) IsDatasource() {}

type NewBigQuery struct {
	ProjectID string `json:"projectID"`
	Dataset   string `json:"dataset"`
	Table     string `json:"table"`
}

type NewDataproduct struct {
	Name             string      `json:"name"`
	Description      *string     `json:"description"`
	Slug             *string     `json:"slug"`
	Repo             *string     `json:"repo"`
	Pii              bool        `json:"pii"`
	Keywords         []string    `json:"keywords"`
	Group            string      `json:"group"`
	TeamkatalogenURL *string     `json:"teamkatalogenURL"`
	BigQuery         NewBigQuery `json:"bigquery"`
	Requesters       []string    `json:"requesters"`
	Metadata         BigqueryMetadata
}

type AccessRequest struct {
	DataproductID uuid.UUID    `json:"dataproductID"`
	Subject       *string      `json:"subject"`
	SubjectType   *SubjectType `json:"subjectType"`
	Owner         *string      `json:"owner"`
	Polly         *Polly       `json:"polly"`
}

type NewAccessRequest struct {
	DataproductID uuid.UUID    `json:"dataproductID"`
	Subject       *string      `json:"subject"`
	SubjectType   *SubjectType `json:"subjectType"`
	Owner         *string      `json:"owner"`
	Polly         *NewPolly    `json:"polly"`
}

type UpdateAccessRequest struct {
	ID       uuid.UUID  `json:"id"`
	Owner    string     `json:"owner"`
	PollyID  *uuid.UUID `json:"polly_id"`
	NewPolly *NewPolly  `json:"new_polly"`
}

type NewGrant struct {
	DataproductID uuid.UUID    `json:"dataproductID"`
	Expires       *time.Time   `json:"expires"`
	Subject       *string      `json:"subject"`
	SubjectType   *SubjectType `json:"subjectType"`
}

type UpdateDataproduct struct {
	Name             string   `json:"name"`
	Description      *string  `json:"description"`
	Slug             *string  `json:"slug"`
	Repo             *string  `json:"repo"`
	Pii              bool     `json:"pii"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	Keywords         []string `json:"keywords"`
	Requesters       []string `json:"requesters"`
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

type DataproductServices struct {
	Metabase *string `json:"metabase"`
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
	case "serviceAccount":
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

type Keyword struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

type GroupStats struct {
	Email        string `json:"email"`
	Dataproducts int    `json:"dataproducts"`
}
