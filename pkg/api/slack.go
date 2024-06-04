package api

import "net/http"

func isValidSlackChannel(name string) (bool, *APIError) {
	ok, err := slackClient.IsValidSlackChannel(name)
	if err != nil {
		return false, NewAPIError(http.StatusInternalServerError, err, "Failed to validate slack channel")
	}

	return ok, nil
}
