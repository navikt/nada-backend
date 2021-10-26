package models

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

type GCPProject struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group *Group `json:"group"`
}

type Group struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type BigQueryTable struct {
	Description  string       `json:"description"`
	LastModified time.Time    `json:"lastModified"`
	Name         string       `json:"name"`
	Type         BigQueryType `json:"type"`
}

type BigQueryType string

const (
	BigQueryTypeTable            BigQueryType = "table"
	BigQueryTypeView             BigQueryType = "view"
	BigQueryTypeMaterializedView BigQueryType = "materialized_view"
)

var AllBigQueryType = []BigQueryType{
	BigQueryTypeTable,
	BigQueryTypeView,
	BigQueryTypeMaterializedView,
}

func (e BigQueryType) IsValid() bool {
	switch e {
	case BigQueryTypeTable, BigQueryTypeView, BigQueryTypeMaterializedView:
		return true
	}
	return false
}

func (e BigQueryType) String() string {
	return string(e)
}

func (e *BigQueryType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = BigQueryType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid BigQueryType", str)
	}
	return nil
}

func (e BigQueryType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
