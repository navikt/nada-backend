package api

import (
	"database/sql"
	"errors"
	"net/http"
)

type APIError struct {
	HttpStatus int
	Err        error
	Message    string
}

func (e *APIError) Error() string {
	return e.Err.Error()
}

func NewAPIError(status int, err error, message string) *APIError {
	return &APIError{
		HttpStatus: status,
		Err:        err,
		Message:    message,
	}
}

func NewInternalError(err error, message string) *APIError {
	return NewAPIError(http.StatusInternalServerError, err, message)
}

func DBErrorToAPIError(err error, message string) *APIError {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return NewAPIError(http.StatusNotFound, err, message)
	}

	return NewAPIError(http.StatusInternalServerError, err, message+":"+err.Error())
}

func (e *APIError) Log() {
	log.WithError(e.Err).Error(e.Message)
}
