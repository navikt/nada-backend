package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

// Dataproducts is the resolver for the dataproducts field.
func (r *productAreaResolver) Dataproducts(ctx context.Context, obj *models.ProductArea) ([]*models.Dataproduct, error) {
	return r.repo.GetDataproductByProductArea(ctx, obj.ExternalID)
}

// Stories is the resolver for the stories field.
func (r *productAreaResolver) Stories(ctx context.Context, obj *models.ProductArea) ([]*models.GraphStory, error) {
	dbStories, err := r.repo.GetStoriesByProductArea(ctx, obj.ExternalID)
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

// ProductArea is the resolver for the productArea field.
func (r *queryResolver) ProductArea(ctx context.Context, id uuid.UUID) (*models.ProductArea, error) {
	return r.repo.GetProductArea(ctx, id)
}

// ProductAreas is the resolver for the productAreas field.
func (r *queryResolver) ProductAreas(ctx context.Context) ([]*models.ProductArea, error) {
	// temporarily return only one product area
	dashboardID := os.Getenv("DASHBOARD_PA_ID")
	if dashboardID == "" {
		return nil, nil
	}
	pas, err := r.repo.GetProductAreas(ctx)
	if err != nil {
		fmt.Println(err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	for _, pa := range pas {
		if pa.ExternalID == dashboardID {
			return []*models.ProductArea{pa}, nil
		}
	}

	return nil, nil
}

// ProductArea returns generated.ProductAreaResolver implementation.
func (r *Resolver) ProductArea() generated.ProductAreaResolver { return &productAreaResolver{r} }

type productAreaResolver struct{ *Resolver }
