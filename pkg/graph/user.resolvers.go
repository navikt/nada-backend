package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/generated"
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

func (r *userInfoResolver) GCPProjects(ctx context.Context, obj *models.UserInfo) ([]*models.GCPProject, error) {
	user := auth.GetUser(ctx)
	ret := []*models.GCPProject{}

	for _, grp := range user.Groups {
		proj, ok := r.gcpProjects.Get(grp.Email)
		if !ok {
			continue
		}
		for _, p := range proj {
			ret = append(ret, &models.GCPProject{
				ID: p,
				Group: &models.Group{
					Name:  grp.Name,
					Email: grp.Email,
				},
			})
		}
	}

	return ret, nil
}

// UserInfo returns generated.UserInfoResolver implementation.
func (r *Resolver) UserInfo() generated.UserInfoResolver { return &userInfoResolver{r} }

type userInfoResolver struct{ *Resolver }
