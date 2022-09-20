package graph

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func isAllowedToGrantAccess(ctx context.Context, r *database.Repo, dp *models.Dataproduct, datasetID uuid.UUID, subject string, user *auth.User) error {
	if ensureUserInGroup(ctx, dp.Owner.Group) == nil {
		return nil
	}
	if !strings.EqualFold(subject, user.Email) {
		return ErrUnauthorized
	}
	requesters, err := r.GetDatasetRequesters(ctx, datasetID)
	if err != nil {
		return err
	}

	for _, r := range requesters {
		if user.GoogleGroups.Contains(r) || strings.EqualFold(r, user.Email) {
			return nil
		}
	}

	return ErrUnauthorized
}

func contains(keywords []string, keyword string) bool {
	for _, k := range keywords {
		if k == keyword {
			return true
		}
	}
	return false
}
