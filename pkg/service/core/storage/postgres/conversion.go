package postgres

import (
	"database/sql"
	"github.com/google/uuid"
	"net/url"
	"strings"
	"time"
)

type Converter[O any] interface {
	To() (O, error)
}

func From[I Converter[O], O any](i I) (O, error) {
	return i.To()
}

func uuidListToStringList(uuids []uuid.UUID) []string {
	strs := make([]string, len(uuids))

	for i, u := range uuids {
		strs[i] = u.String()
	}

	return strs
}

func uuidPtrToNullString(u *uuid.UUID) sql.NullString {
	if u == nil {
		return sql.NullString{}
	}

	return sql.NullString{
		String: u.String(),
		Valid:  true,
	}
}

func uuidToNullString(u uuid.UUID) sql.NullString {
	return sql.NullString{
		String: u.String(),
		Valid:  true,
	}
}

func nullStringToUUIDPtr(ns sql.NullString) *uuid.UUID {
	if !ns.Valid {
		return nil
	}

	v := uuid.MustParse(ns.String)

	return &v
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

func emailOfSubjectToLower(subjectWithType string) string {
	parts := strings.Split(subjectWithType, ":")
	parts[1] = strings.ToLower(parts[1])

	return strings.Join(parts, ":")
}
