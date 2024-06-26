package service

import (
	"context"

	"github.com/navikt/nada-backend/pkg/teamkatalogen"
)

func SearchTeamKatalogen(ctx context.Context, gcpGroups []string) ([]teamkatalogen.TeamkatalogenResult, *APIError) {
	tr, err := tkClient.Search(ctx, gcpGroups)
	if err != nil {
		return nil, NewInternalError(err, "Failed to search Team Katalogen")
	}
	return tr, nil
}
