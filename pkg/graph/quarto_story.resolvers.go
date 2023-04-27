package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.30

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

// CreateQuartoStory is the resolver for the createQuartoStory field.
func (r *mutationResolver) CreateQuartoStory(ctx context.Context, files []*graphql.Upload, input models.NewQuartoStory) (*models.QuartoStory, error) {
	story, err := r.repo.CreateQuartoStory(ctx, auth.GetUser(ctx).Email, input)
	if err != nil {
		return nil, err
	}

	if err = WriteFilesToBucket(ctx, story.ID.String(), files); err != nil {
		return nil, err
	}

	//Create a new File object with the uploaded file's public URL
	return &story, nil
}

// UpdateQuartoStoryMetadata is the resolver for the updateQuartoStoryMetadata field.
func (r *mutationResolver) UpdateQuartoStoryMetadata(ctx context.Context, id uuid.UUID, name string, description string, keywords []string, teamkatalogenURL *string, productAreaID *string, teamID *string, group string) (*models.QuartoStory, error) {
	existing, err := r.repo.GetQuartoStory(ctx, id)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(existing.Group) {
		return nil, ErrUnauthorized
	}

	story, err := r.repo.UpdateQuartoStoryMetadata(ctx, id, name, description, keywords, teamkatalogenURL,
		productAreaID, teamID, group)
	if err != nil {
		return nil, err
	}

	return story, nil
}

// DeleteQuartoStory is the resolver for the deleteQuartoStory field.
func (r *mutationResolver) DeleteQuartoStory(ctx context.Context, id uuid.UUID) (bool, error) {
	s, err := r.repo.GetQuartoStory(ctx, id)
	if err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(s.Group) {
		return false, ErrUnauthorized
	}

	if err = r.repo.DeleteQuartoStory(ctx, id); err != nil {
		return false, err
	}

	if err := deleteQuartoStoryFolder(ctx, id.String()); err != nil {
		r.log.WithError(err).
			Errorf("Quarto story %v metadata deleted but failed to delete story files in GCP",
				id)
		return false, err
	}

	return true, nil
}

// QuartoStory is the resolver for the quartoStory field.
func (r *queryResolver) QuartoStory(ctx context.Context, id uuid.UUID) (*models.QuartoStory, error) {
	return r.repo.GetQuartoStory(ctx, id)
}
