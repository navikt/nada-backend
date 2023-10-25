package models

import (
	"fmt"
	"io"
	"strconv"
)

type MappingService string

const (
	MappingServiceMetabase MappingService = "metabase"
)

var AllMappingService = []MappingService{
	MappingServiceMetabase,
}

func (e MappingService) IsValid() bool {
	switch e {
	case MappingServiceMetabase:
		return true
	}
	return false
}

func (e MappingService) String() string {
	return string(e)
}

func (e *MappingService) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = MappingService(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid MappingService", str)
	}
	return nil
}

func (e MappingService) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
