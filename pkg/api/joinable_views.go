package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

func getJoinableViewsForReferenceAndUser(ctx context.Context, user string, pseudoDatasetID uuid.UUID) ([]gensql.GetJoinableViewsForReferenceAndUserRow, error) {
	joinableViews, err := queries.GetJoinableViewsForReferenceAndUser(ctx, gensql.GetJoinableViewsForReferenceAndUserParams{
		Owner:           user,
		PseudoDatasetID: pseudoDatasetID,
	})
	if err != nil {
		return nil, err
	}

	return joinableViews, nil
}
