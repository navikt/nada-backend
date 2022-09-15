package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

// Dataproducts is the resolver for the dataproducts field.
func (r *productAreaResolver) Dataproducts(ctx context.Context, obj *models.ProductArea) ([]*models.Dataproduct, error) {
	return r.repo.GetDataproductByProductArea(ctx, obj.ID)
}

// Stories is the resolver for the stories field.
func (r *productAreaResolver) Stories(ctx context.Context, obj *models.ProductArea) ([]*models.GraphStory, error) {
	dbStories, err := r.repo.GetStoriesByProductArea(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	graphStories := make([]*models.GraphStory, len(dbStories))
	for idx, s := range dbStories {
		graphStory, err := storyFromDB(s)
		if err != nil {
			return nil, err
		}
		graphStories[idx] = graphStory
	}

	return graphStories, nil
}

// Teams is the resolver for the teams field.
func (r *productAreaResolver) Teams(ctx context.Context, obj *models.ProductArea) ([]*models.Team, error) {
	return r.teamkatalogen.GetTeamsInProductArea(ctx, obj.ID)
}

// ProductArea is the resolver for the productArea field.
func (r *queryResolver) ProductArea(ctx context.Context, id string) (*models.ProductArea, error) {
	return r.teamkatalogen.GetProductArea(ctx, id)
}

// ProductAreas is the resolver for the productAreas field.
func (r *queryResolver) ProductAreas(ctx context.Context) ([]*models.ProductArea, error) {
	return r.teamkatalogen.GetProductAreas(ctx)
}

// Team is the resolver for the team field.
func (r *queryResolver) Team(ctx context.Context, id string) (*models.Team, error) {
	panic(fmt.Errorf("not implemented: Team - team"))
}

// Dataproducts is the resolver for the dataproducts field.
func (r *teamResolver) Dataproducts(ctx context.Context, obj *models.Team) ([]*models.Dataproduct, error) {
	return r.repo.GetDataproductByTeam(ctx, obj.ID)
}

// Stories is the resolver for the stories field.
func (r *teamResolver) Stories(ctx context.Context, obj *models.Team) ([]*models.GraphStory, error) {
	dbStories, err := r.repo.GetStoriesByTeam(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	graphStories := make([]*models.GraphStory, len(dbStories))
	for idx, s := range dbStories {
		graphStory, err := storyFromDB(s)
		if err != nil {
			return nil, err
		}
		graphStories[idx] = graphStory
	}

	return graphStories, nil
}

// ProductArea returns generated.ProductAreaResolver implementation.
func (r *Resolver) ProductArea() generated.ProductAreaResolver { return &productAreaResolver{r} }

// Team returns generated.TeamResolver implementation.
func (r *Resolver) Team() generated.TeamResolver { return &teamResolver{r} }

type productAreaResolver struct{ *Resolver }
type teamResolver struct{ *Resolver }
