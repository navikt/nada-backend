package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}

	return &ns.String
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

func nullUUIDToUUIDPtr(nu uuid.NullUUID) *uuid.UUID {
	if !nu.Valid {
		return nil
	}
	return &nu.UUID
}

func apiGetWrapper(handlerDelegate func(r *http.Request) (interface{}, *APIError)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dto, apiErr := handlerDelegate(r)
		if apiErr != nil {
			apiErr.Log()
			http.Error(w, apiErr.Error(), apiErr.HttpStatus)
			return
		}
		err := json.NewEncoder(w).Encode(dto)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
