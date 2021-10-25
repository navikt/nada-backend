package graph

import (
	"context"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func isAllowedToGrantAccess(ctx context.Context, r *database.Repo, dp *models.Dataproduct, subject string, user *auth.User) error {
	if ensureUserInGroup(ctx, dp.Owner.Group) == nil {
		return nil
	}
	if subject != user.Email {
		return ErrUnauthorized
	}
	requesters, err := r.GetDataproductRequesters(ctx, dp.ID)
	if err != nil {
		return err
	}

	for _, r := range requesters {
		if user.Groups.Contains(r) || r == user.Email {
			return nil
		}
	}

	return ErrUnauthorized
}
