package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"html"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
	log "github.com/sirupsen/logrus"
)

func (r *dataproductResolver) Datasets(ctx context.Context, obj *models.Dataproduct) ([]*models.Dataset, error) {
	return r.repo.GetDatasetsInDataproduct(ctx, obj.ID)
}

func (r *mutationResolver) CreateDataproduct(ctx context.Context, input models.NewDataproduct) (*models.Dataproduct, error) {
	if err := ensureUserInGroup(ctx, input.Group); err != nil {
		return nil, err
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
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

func (r *queryResolver) Dataproduct(ctx context.Context, id uuid.UUID) (*models.Dataproduct, error) {
	return r.repo.GetDataproduct(ctx, id)
}

func (r *queryResolver) Dataproducts(ctx context.Context, limit *int, offset *int, service *models.MappingService) ([]*models.Dataproduct, error) {
	l, o := pagination(limit, offset)
	return r.repo.GetDataproducts(ctx, l, o)
}

func (r *queryResolver) GroupStats(ctx context.Context, limit *int, offset *int) ([]*models.GroupStats, error) {
	l, o := pagination(limit, offset)
	return r.repo.DataproductGroupStats(ctx, l, o)
}

func (r *newDataproductResolver) Pii(ctx context.Context, obj *models.NewDataproduct, data bool) error {
	panic(fmt.Errorf("not implemented"))
}

// Dataproduct returns generated.DataproductResolver implementation.
func (r *Resolver) Dataproduct() generated.DataproductResolver { return &dataproductResolver{r} }

// NewDataproduct returns generated.NewDataproductResolver implementation.
func (r *Resolver) NewDataproduct() generated.NewDataproductResolver {
	return &newDataproductResolver{r}
}

type dataproductResolver struct{ *Resolver }
type newDataproductResolver struct{ *Resolver }
