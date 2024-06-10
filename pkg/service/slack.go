package service

import (
	"context"
	"net/http"
)

func IsValidSlackChannel(ctx context.Context, name string) (bool, *APIError) {
	ok, err := slackClient.IsValidSlackChannel(name)
	if err != nil {
		return false, NewAPIError(http.StatusInternalServerError, err, "Failed to validate slack channel")
	}

	return ok, nil
}
