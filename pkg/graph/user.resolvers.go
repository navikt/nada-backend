package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *queryResolver) UserInfo(ctx context.Context) (*models.UserInfo, error) {
	user := auth.GetUser(ctx)
	groups := []*models.Group{}
	for _, g := range user.Groups {
		groups = append(groups, &models.Group{
			Name:  g.Name,
			Email: g.Email,
		})
	}

	return &models.UserInfo{
		Name:   user.Name,
		Email:  user.Email,
		Groups: groups,
	}, nil
}
