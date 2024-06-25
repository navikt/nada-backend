package postgres

import (
	"database/sql"
	"strings"
	"time"
)

type Converter[O any] interface {
	To() (O, error)
}

func From[I Converter[O], O any](i I) (O, error) {
	return i.To()
}

func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: *s, Valid: true}
}

// FIXME: move all of these into a helpers.go file
func ptrToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}

	return sql.NullTime{Time: *t, Valid: true}
}

func emailOfSubjectToLower(subjectWithType string) string {
	parts := strings.Split(subjectWithType, ":")
	parts[1] = strings.ToLower(parts[1])

	return strings.Join(parts, ":")
}
