package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"html"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
	log "github.com/sirupsen/logrus"
)

// Description is the resolver for the description field.
func (r *dataproductResolver) Description(ctx context.Context, obj *models.Dataproduct, raw *bool) (string, error) {
	if obj.Description == nil {
		return "", nil
	}

	if raw != nil && *raw {
		return html.UnescapeString(*obj.Description), nil
	}

	return *obj.Description, nil
}

// Keywords is the resolver for the keywords field.
func (r *dataproductResolver) Keywords(ctx context.Context, obj *models.Dataproduct) ([]string, error) {
	datasets, err := r.repo.GetDatasetsInDataproduct(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	keywords := []string{}
	for _, ds := range datasets {
		keywords = append(keywords, ds.Keywords...)
	}

	return keywords, nil
}

// Datasets is the resolver for the datasets field.
func (r *dataproductResolver) Datasets(ctx context.Context, obj *models.Dataproduct) ([]*models.Dataset, error) {
	return r.repo.GetDatasetsInDataproduct(ctx, obj.ID)
}

// CreateDataproduct is the resolver for the createDataproduct field.
func (r *mutationResolver) CreateDataproduct(ctx context.Context, input models.NewDataproduct) (*models.Dataproduct, error) {
	if err := ensureUserInGroup(ctx, input.Group); err != nil {
		return nil, err
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	for i, ds := range input.Datasets {
		metadata, err := r.prepareBigQuery(ctx, ds.Bigquery, input.Group)
		if err != nil {
			return nil, err
		}

		input.Datasets[i].Metadata = metadata

		if input.Datasets[i].Description != nil && *input.Datasets[i].Description != "" {
			*input.Datasets[i].Description = html.EscapeString(*input.Datasets[i].Description)
		}
	}

	dp, err := r.repo.CreateDataproduct(ctx, input)
	if err != nil {
		return nil, err
	}

	err = r.slack.NewDataproduct(dp)
	if err != nil {
		log.Errorf("failed to send slack notification: %v", err)
	}
	return dp, nil
}

// UpdateDataproduct is the resolver for the updateDataproduct field.
func (r *mutationResolver) UpdateDataproduct(ctx context.Context, id uuid.UUID, input models.UpdateDataproduct) (*models.Dataproduct, error) {
	dp, err := r.repo.GetDataproduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}
	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}
	return r.repo.UpdateDataproduct(ctx, id, input)
}

// DeleteDataproduct is the resolver for the deleteDataproduct field.
func (r *mutationResolver) DeleteDataproduct(ctx context.Context, id uuid.UUID) (bool, error) {
	dp, err := r.repo.GetDataproduct(ctx, id)
	if err != nil {
		return false, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	return true, r.repo.DeleteDataproduct(ctx, dp.ID)
}

// Dataproduct is the resolver for the dataproduct field.
func (r *queryResolver) Dataproduct(ctx context.Context, id uuid.UUID) (*models.Dataproduct, error) {
	return r.repo.GetDataproduct(ctx, id)
}

// Dataproducts is the resolver for the dataproducts field.
func (r *queryResolver) Dataproducts(ctx context.Context, limit *int, offset *int, service *models.MappingService) ([]*models.Dataproduct, error) {
	l, o := pagination(limit, offset)
	return r.repo.GetDataproducts(ctx, l, o)
}

// GroupStats is the resolver for the groupStats field.
func (r *queryResolver) GroupStats(ctx context.Context, limit *int, offset *int) ([]*models.GroupStats, error) {
	l, o := pagination(limit, offset)
	return r.repo.DataproductGroupStats(ctx, l, o)
}

// Dataproduct returns generated.DataproductResolver implementation.
func (r *Resolver) Dataproduct() generated.DataproductResolver { return &dataproductResolver{r} }

type dataproductResolver struct{ *Resolver }
