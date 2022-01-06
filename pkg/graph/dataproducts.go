package graph

import (
	"context"
	"strings"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func isAllowedToGrantAccess(ctx context.Context, r *database.Repo, dp *models.Dataproduct, subject string, user *auth.User) error {
	if ensureUserInGroup(ctx, dp.Owner.Group) == nil {
		return nil
	}
	if !strings.EqualFold(subject, user.Email) {
		return ErrUnauthorized
	}
	requesters, err := r.GetDataproductRequesters(ctx, dp.ID)
	if err != nil {
		return err
	}

	for _, r := range requesters {
		if r == "all-users@nav.no" {
			return nil
		}
		if user.Groups.Contains(r) || strings.EqualFold(r, user.Email) {
			return nil
		}
	}

	return ErrUnauthorized
}
