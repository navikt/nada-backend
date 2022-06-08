package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"html"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
	log "github.com/sirupsen/logrus"
)

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

	for i, ds := range input.Datasets {
		if err := r.ensureUserHasAccessToGcpProject(ctx, ds.Bigquery.ProjectID); err != nil {
			return nil, err
		}

		metadata, err := r.bigquery.TableMetadata(ctx, ds.Bigquery.ProjectID, ds.Bigquery.Dataset, ds.Bigquery.Table)
		if err != nil {
			return nil, fmt.Errorf("trying to create table %v, but it does not exist in %v.%v",
				ds.Bigquery.Table, ds.Bigquery.ProjectID, ds.Bigquery.Dataset)
		}

		switch metadata.TableType {
		case bigquery.RegularTable:
		case bigquery.ViewTable:
			if err := r.accessMgr.AddToAuthorizedViews(ctx, ds.Bigquery.ProjectID, ds.Bigquery.Dataset, ds.Bigquery.Table); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported table type: %v", metadata.TableType)
		}

		input.Datasets[i].Metadata = metadata
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

// Dataproduct returns generated.DataproductResolver implementation.
func (r *Resolver) Dataproduct() generated.DataproductResolver { return &dataproductResolver{r} }

type dataproductResolver struct{ *Resolver }
