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

func (r *mutationResolver) PublishStory(ctx context.Context, id uuid.UUID, group string, keywords []string) (*models.GraphStory, error) {
	user := auth.GetUser(ctx)
	if !user.Groups.Contains(group) {
		return nil, ErrUnauthorized
	}

	if keywords == nil {
		keywords = []string{}
	}

	s, err := r.repo.PublishStory(ctx, id, group, keywords)
	if err != nil {
		return nil, err
	}
	return storyFromDB(s)
}

func (r *mutationResolver) UpdateStory(ctx context.Context, id uuid.UUID, target uuid.UUID) (*models.GraphStory, error) {
	t, err := r.repo.GetStory(ctx, target)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)
	if !user.Groups.Contains(t.Group) {
		return nil, ErrUnauthorized
	}

	s, err := r.repo.UpdateStory(ctx, id, target)
	if err != nil {
		return nil, err
	}
	return storyFromDB(s)
}

func (r *mutationResolver) UpdateStoryMetadata(ctx context.Context, id uuid.UUID, name string, keywords []string) (*models.GraphStory, error) {
	existing, err := r.repo.GetStory(ctx, id)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)
	if !user.Groups.Contains(existing.Group) {
		return nil, ErrUnauthorized
	}

	story, err := r.repo.UpdateStoryMetadata(ctx, id, name, keywords)
	if err != nil {
		return nil, err
	}

	return storyFromDB(story)
}

func (r *mutationResolver) DeleteStory(ctx context.Context, id uuid.UUID) (bool, error) {
	s, err := r.repo.GetStory(ctx, id)
	if err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if !user.Groups.Contains(s.Group) {
		return false, ErrUnauthorized
	}

	if err := r.repo.DeleteStory(ctx, id); err != nil {
		return false, err
	}

	return true, nil
}

func (r *queryResolver) Stories(ctx context.Context, draft *bool) ([]*models.GraphStory, error) {
	var (
		stories []*models.DBStory
		err     error
	)

	if draft != nil && *draft {
		stories, err = r.repo.GetStoryDrafts(ctx)
	} else {
		stories, err = r.repo.GetStories(ctx)
	}
	if err != nil {
		return nil, err
	}

	ret := make([]*models.GraphStory, len(stories))
	for i, s := range stories {
		ret[i], err = storyFromDB(s)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (r *queryResolver) Story(ctx context.Context, id uuid.UUID, draft *bool) (*models.GraphStory, error) {
	var (
		story *models.DBStory
		err   error
	)
	if draft != nil && *draft {
		story, err = r.repo.GetStoryDraft(ctx, id)
	} else {
		story, err = r.repo.GetStory(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	return storyFromDB(story)
}

func (r *queryResolver) StoryView(ctx context.Context, id uuid.UUID, draft *bool) (models.GraphStoryView, error) {
	var (
		storyView *models.DBStoryView
		err       error
	)
	if draft != nil && *draft {
		storyView, err = r.repo.GetStoryViewDraft(ctx, id)
	} else {
		storyView, err = r.repo.GetStoryView(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	return storyViewFromDB(storyView)
}

func (r *queryResolver) StoryToken(ctx context.Context, id uuid.UUID) (*models.StoryToken, error) {
	story, err := r.repo.GetStory(ctx, id)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)
	if !user.Groups.Contains(story.Group) {
		return nil, ErrUnauthorized
	}

	return r.repo.GetStoryToken(ctx, id)
}

func (r *storyResolver) Owner(ctx context.Context, obj *models.GraphStory) (*models.Owner, error) {
	return &models.Owner{
		Group: obj.Group,
	}, nil
}

func (r *storyResolver) Views(ctx context.Context, obj *models.GraphStory) ([]models.GraphStoryView, error) {
	var (
		views []*models.DBStoryView
		err   error
	)
	if obj.Draft {
		views, err = r.repo.GetStoryViewDrafts(ctx, obj.ID)
	} else {
		views, err = r.repo.GetStoryViews(ctx, obj.ID)
	}
	if err != nil {
		return nil, err
	}

	ret := make([]models.GraphStoryView, len(views))
	for i, v := range views {
		ret[i], err = storyViewFromDB(v)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

// Story returns generated.StoryResolver implementation.
func (r *Resolver) Story() generated.StoryResolver { return &storyResolver{r} }

type storyResolver struct{ *Resolver }
