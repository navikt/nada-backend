package postgres

import (
	"database/sql"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Converter[O any] interface {
	To() (O, error)
}

func From[I Converter[O], O any](i I) (O, error) {
	return i.To()
}

func uuidToNullUUID(u uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{
		UUID:  u,
		Valid: true,
	}
}

func ptrToIntDefault(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

func matchAny(s string, targetSet []string) bool {
	for _, v := range targetSet {
		if s == v {
			return true
		}
	}
	return false
}

func nullInt32ToIntPtr(ni sql.NullInt32) *int {
	if !ni.Valid {
		return nil
	}

	i := int(ni.Int32)

	return &i
}

func nullStringToString(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}

	return ns.String
}

func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: *s, Valid: true}
}

func ptrToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}

	return sql.NullTime{Time: *t, Valid: true}
}

func slugify(maybeslug *string, fallback string) string {
	if maybeslug != nil {
		return *maybeslug
	}
	// TODO(thokra): Smartify this?
	return url.PathEscape(fallback)
}

func ptrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}

	return &ns.String
}

func nullUUIDToUUIDPtr(nu uuid.NullUUID) *uuid.UUID {
	if !nu.Valid {
		return nil
	}
	return &nu.UUID
}

func uuidPtrToNullUUID(u *uuid.UUID) uuid.NullUUID {
	if u == nil {
		return uuid.NullUUID{}
	}

	return uuid.NullUUID{
		UUID:  *u,
		Valid: true,
	}
}

func emailOfSubjectToLower(subjectWithType string) string {
	parts := strings.Split(subjectWithType, ":")
	parts[1] = strings.ToLower(parts[1])

	return strings.Join(parts, ":")
}
