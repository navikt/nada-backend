package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
)

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}

	return &ns.String
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

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}

	return &nt.Time
}

func ptrToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}

	return sql.NullTime{Time: *t, Valid: true}
}

func nullUUIDToUUIDPtr(nu uuid.NullUUID) *uuid.UUID {
	if !nu.Valid {
		return nil
	}
	return &nu.UUID
}

func apiWrapper(handlerDelegate func(r *http.Request) (interface{}, *APIError)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dto, apiErr := handlerDelegate(r)
		if apiErr != nil {
			apiErr.Log()
			http.Error(w, apiErr.Error(), apiErr.HttpStatus)
			return
		}
		if dto != nil {
			err := json.NewEncoder(w).Encode(dto)
			if err != nil {
				log.WithError(err).Error("Failed to encode response")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func ptrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
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

func emailOfSubjectToLower(subectWithType string) string {
	parts := strings.Split(subectWithType, ":")
	parts[1] = strings.ToLower(parts[1])

	return strings.Join(parts, ":")
}

func ensureOwner(ctx context.Context, owner string) error {
	user := auth.GetUser(ctx)

	if user != nil && (user.GoogleGroups.Contains(owner) || owner == user.Email) {
		return nil
	}

	return ErrUnauthorized
}
