// Code generated by sqlc. DO NOT EDIT.

package gensql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tabbed/pqtype"
)

type DatasourceType string

const (
	DatasourceTypeBigquery DatasourceType = "bigquery"
)

func (e *DatasourceType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = DatasourceType(s)
	case string:
		*e = DatasourceType(s)
	default:
		return fmt.Errorf("unsupported scan type for DatasourceType: %T", src)
	}
	return nil
}

type Dataproduct struct {
	ID           uuid.UUID
	Name         string
	Description  sql.NullString
	Group        string
	Pii          bool
	Created      time.Time
	LastModified time.Time
	Type         DatasourceType
	TsvDocument  interface{}
}

type DataproductCollection struct {
	ID           uuid.UUID
	Name         string
	Description  sql.NullString
	Slug         string
	Repo         sql.NullString
	Created      time.Time
	LastModified time.Time
	Group        string
	Keywords     []string
	TsvDocument  interface{}
}

type DatasourceBigquery struct {
	DataproductID uuid.UUID
	ProjectID     string
	Dataset       string
	TableName     string
	Schema        pqtype.NullRawMessage
}
