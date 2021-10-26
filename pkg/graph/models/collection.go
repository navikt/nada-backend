package models

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type CollectionElement interface {
	IsCollectionElement()
}

type Collection struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Keywords     []string  `json:"keywords"`
	Owner        Owner     `json:"owner"`
	Slug         string    `json:"slug"`
}

func (c Collection) IsSearchResult() {}

type NewCollection struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Slug        *string  `json:"slug"`
	Group       string   `json:"group"`
	Keywords    []string `json:"keywords"`
}

type UpdateCollection struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Slug        *string  `json:"slug"`
	Keywords    []string `json:"keywords"`
}

type CollectionElementType string

const (
	CollectionElementTypeDataproduct CollectionElementType = "dataproduct"
)

var AllCollectionElementType = []CollectionElementType{
	CollectionElementTypeDataproduct,
}

func (e CollectionElementType) IsValid() bool {
	switch e {
	case CollectionElementTypeDataproduct:
		return true
	}
	return false
}

func (e CollectionElementType) String() string {
	return string(e)
}

func (e *CollectionElementType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = CollectionElementType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid CollectionElementType", str)
	}
	return nil
}

func (e CollectionElementType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
