package api

import (
	"context"
	"errors"
	"net/http"
)

func RotateNadaToken(ctx context.Context, team string) *APIError {
	if team == "" {
		return NewAPIError(http.StatusBadRequest, errors.New("no team provided"))
	}
	if err := ensureUserInGroup(ctx, team+"@nav.no"); err != nil {
		return NewAPIError(http.StatusUnauthorized, err)
	}

	return DBErrorToAPIError(querier.RotateNadaToken(ctx, team))
}
