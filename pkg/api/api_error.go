package api

import (
	"database/sql"
	"errors"
	"net/http"
)

type APIError struct {
	HttpStatus int
	Err        error
}

func (e *APIError) Error() string {
	return e.Err.Error()
}

func NewAPIError(status int, err error) *APIError {
	return &APIError{
		HttpStatus: status,
		Err:        err,
	}
}

func DBErrorToAPIError(err error) *APIError {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return NewAPIError(http.StatusNotFound, err)
	}

	return NewAPIError(http.StatusInternalServerError, err)
}
