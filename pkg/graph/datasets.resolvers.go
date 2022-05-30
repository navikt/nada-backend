package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"os"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
	log "github.com/sirupsen/logrus"
)

func (r *bigQueryResolver) Schema(ctx context.Context, obj *models.BigQuery) ([]*models.TableColumn, error) {
	return r.repo.GetDatasetMetadata(ctx, obj.DatasetID)
}

func (r *datasetResolver) Datasource(ctx context.Context, obj *models.Dataset) (models.Datasource, error) {
	return r.repo.GetBigqueryDatasource(ctx, obj.ID)
}

func (r *datasetResolver) Access(ctx context.Context, obj *models.Dataset) ([]*models.Access, error) {
	all, err := r.repo.ListAccessToDataset(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	dp, err := r.repo.GetDataproduct(ctx, obj.DataproductID)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)
	if user.Groups.Contains(dp.Owner.Group) {
		return all, nil
	}

	var ret []*models.Access
	for _, a := range all {
		if strings.EqualFold(a.Subject, "user:"+user.Email) {
			ret = append(ret, a)
		} else if strings.HasPrefix(a.Subject, "group:") && user.Groups.Contains(strings.TrimPrefix(a.Subject, "group:")) {
			ret = append(ret, a)
		}
	}

	return ret, nil
}

func (r *datasetResolver) Services(ctx context.Context, obj *models.Dataset) (*models.DatasetServices, error) {
	meta, err := r.repo.GetMetabaseMetadata(ctx, obj.ID, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.DatasetServices{}, nil
		}
		return nil, err
	}

	svc := &models.DatasetServices{}
	if meta.DatabaseID > 0 {
		base := "https://metabase.dev.intern.nav.no/browse/%v"
		if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
			base = "https://metabase.intern.nav.no/browse/%v"
		}
		url := fmt.Sprintf(base, meta.DatabaseID)
		svc.Metabase = &url
	}

	return svc, nil
}

func (r *datasetResolver) Mappings(ctx context.Context, obj *models.Dataset) ([]models.MappingService, error) {
	return r.repo.GetDataproductMappings(ctx, obj.ID)
}

func (r *datasetResolver) Requesters(ctx context.Context, obj *models.Dataset) ([]string, error) {
	allRequesters, err := r.repo.GetDatasetRequesters(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	dp, err := r.repo.GetDataproduct(ctx, obj.DataproductID)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)
	if user.Groups.Contains(dp.Owner.Group) {
		return allRequesters, nil
	}

	var ret []string
	for _, r := range allRequesters {
		if strings.EqualFold(r, user.Email) {
			ret = append(ret, r)
		} else if user.Groups.Contains(r) {
			ret = append(ret, r)
		}
	}

	return ret, nil
}

func (r *mutationResolver) CreateDataset(ctx context.Context, input models.NewDataset) (*models.Dataset, error) {
	dp, err := r.repo.GetDataproduct(ctx, input.DataproductID)
	if err != nil {
		return nil, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}

	if err := r.ensureUserHasAccessToGcpProject(ctx, input.BigQuery.ProjectID); err != nil {
		return nil, err
	}

	metadata, err := r.bigquery.TableMetadata(ctx, input.BigQuery.ProjectID, input.BigQuery.Dataset, input.BigQuery.Table)
	if err != nil {
		return nil, fmt.Errorf("trying to create table %v, but it does not exist in %v.%v",
			input.BigQuery.Table, input.BigQuery.ProjectID, input.BigQuery.Dataset)
	}

	switch metadata.TableType {
	case bigquery.RegularTable:
	case bigquery.ViewTable:
		if err := r.accessMgr.AddToAuthorizedViews(ctx, input.BigQuery.ProjectID, input.BigQuery.Dataset, input.BigQuery.Table); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported table type: %v", metadata.TableType)
	}

	input.Metadata = metadata
	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}
	ds, err := r.repo.CreateDataset(ctx, input)
	if err != nil {
		return nil, err
	}
	err = r.slack.NewDataproduct(dp)
	if err != nil {
		log.Errorf("failed to send slack notification: %v", err)
	}
	return ds, nil
}

func (r *mutationResolver) UpdateDataset(ctx context.Context, id uuid.UUID, input models.UpdateDataset) (*models.Dataset, error) {
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
	return r.repo.UpdateDataset(ctx, id, input)
}

func (r *mutationResolver) DeleteDataset(ctx context.Context, id uuid.UUID) (bool, error) {
	ds, err := r.repo.GetDataset(ctx, id)
	if err != nil {
		return false, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return false, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	return true, r.repo.DeleteDataproduct(ctx, dp.ID)
}

func (r *mutationResolver) MapDataset(ctx context.Context, datasetID uuid.UUID, services []models.MappingService) (bool, error) {
	ds, err := r.repo.GetDataset(ctx, datasetID)
	if err != nil {
		return false, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return false, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	err = r.repo.MapDataset(ctx, datasetID, services)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *queryResolver) Dataset(ctx context.Context, id uuid.UUID) (*models.Dataset, error) {
	return r.repo.GetDataset(ctx, id)
}

func (r *queryResolver) AccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]*models.AccessRequest, error) {
	dp, err := r.repo.GetDataproduct(ctx, datasetID)
	if err != nil {
		return nil, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}

	return r.repo.ListAccessRequestsForDataset(ctx, datasetID)
}

// BigQuery returns generated.BigQueryResolver implementation.
func (r *Resolver) BigQuery() generated.BigQueryResolver { return &bigQueryResolver{r} }

// Dataset returns generated.DatasetResolver implementation.
func (r *Resolver) Dataset() generated.DatasetResolver { return &datasetResolver{r} }

type bigQueryResolver struct{ *Resolver }
type datasetResolver struct{ *Resolver }
