package service

import "net/http"

func IsValidSlackChannel(name string) (bool, *APIError) {
	ok, err := slackClient.IsValidSlackChannel(name)
	if err != nil {
		return false, NewAPIError(http.StatusInternalServerError, err, "Failed to validate slack channel")
	}

	return ok, nil
}
