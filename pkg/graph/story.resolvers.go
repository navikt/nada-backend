package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *mutationResolver) PublishStory(ctx context.Context, id uuid.UUID, group string) (*models.Story, error) {
	user := auth.GetUser(ctx)
	if !user.Groups.Contains(group) {
		return nil, ErrUnauthorized
	}

	return r.repo.PublishStory(ctx, id, group)
}

func (r *queryResolver) Stories(ctx context.Context, draft *bool) ([]*models.Story, error) {
	if draft != nil && *draft {
		return r.repo.GetStoryDrafts(ctx)
	}
	return r.repo.GetStories(ctx)
}

func (r *storyResolver) Owner(ctx context.Context, obj *models.Story) (*models.Owner, error) {
	return &models.Owner{
		Group: obj.Group,
	}, nil
}

func (r *storyResolver) Views(ctx context.Context, obj *models.Story) ([]*models.StoryView, error) {
	if obj.Draft {
		return r.repo.GetStoryViewDraft(ctx, obj.ID)
	}
	return r.repo.GetStoryViews(ctx, obj.ID)
}

// Story returns generated.StoryResolver implementation.
func (r *Resolver) Story() generated.StoryResolver { return &storyResolver{r} }

type storyResolver struct{ *Resolver }
