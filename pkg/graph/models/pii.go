package models

import (
	"fmt"
	"io"
	"strconv"
)

type PiiLevel string

const (
	PiiLevelSensitive  PiiLevel = "sensitive"
	PiiLevelAnonymised PiiLevel = "anonymised"
	PiiLevelNone       PiiLevel = "none"
)

var AllPiiLevel = []PiiLevel{
	PiiLevelSensitive,
	PiiLevelAnonymised,
	PiiLevelNone,
}

func (e PiiLevel) IsValid() bool {
	switch e {
	case PiiLevelSensitive:
		return true
	case PiiLevelAnonymised:
		return true
	case PiiLevelNone:
		return true
	}
	return false
}

func (e PiiLevel) String() string {
	return string(e)
}

func (e *PiiLevel) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = PiiLevel(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid PiiLevel", str)
	}
	return nil
}

func (e PiiLevel) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
