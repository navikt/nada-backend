package graph

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *mutationResolver) grantAllUsersOnCreation(ctx context.Context, bigquery models.NewBigQuery, grantAllUsers bool) error {
	if grantAllUsers {
		return r.accessMgr.Grant(ctx, bigquery.ProjectID, bigquery.Dataset, bigquery.Table, "group:all-users@nav.no")
	}

	return nil
}
