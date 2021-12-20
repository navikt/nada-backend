package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *queryResolver) Stories(ctx context.Context, draft *bool) ([]*models.Story, error) {
	if draft != nil && *draft {
		return r.repo.GetStoryDrafts(ctx)
	}
	panic("not implemented")
}

func (r *storyResolver) Owner(ctx context.Context, obj *models.Story) (*models.Owner, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *storyResolver) Views(ctx context.Context, obj *models.Story) ([]*models.StoryView, error) {
	if obj.Draft {
		return r.repo.GetStoryViewDraft(ctx, obj.ID)
	}
	panic(fmt.Errorf("not implemented"))
}

// Story returns generated.StoryResolver implementation.
func (r *Resolver) Story() generated.StoryResolver { return &storyResolver{r} }

type storyResolver struct{ *Resolver }