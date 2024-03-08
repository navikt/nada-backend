package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.44

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

// Schema is the resolver for the schema field.
func (r *bigQueryResolver) Schema(ctx context.Context, obj *models.BigQuery) ([]*models.TableColumn, error) {
	return r.repo.GetDatasetMetadata(ctx, obj.DatasetID)
}

// Dataproduct is the resolver for the dataproduct field.
func (r *datasetResolver) Dataproduct(ctx context.Context, obj *models.Dataset) (*models.Dataproduct, error) {
	return r.repo.GetDataproduct(ctx, obj.DataproductID)
}

// Description is the resolver for the description field.
func (r *datasetResolver) Description(ctx context.Context, obj *models.Dataset, raw *bool) (string, error) {
	if obj.Description == nil {
		return "", nil
	}

	if raw != nil && *raw {
		return html.UnescapeString(*obj.Description), nil
	}

	return *obj.Description, nil
}

// Owner is the resolver for the owner field.
func (r *datasetResolver) Owner(ctx context.Context, obj *models.Dataset) (*models.Owner, error) {
	dp, err := r.repo.GetDataproduct(ctx, obj.DataproductID)
	if err != nil {
		return nil, err
	}
	return dp.Owner, nil
}

// Datasource is the resolver for the datasource field.
func (r *datasetResolver) Datasource(ctx context.Context, obj *models.Dataset) (models.Datasource, error) {
	return r.repo.GetBigqueryDatasource(ctx, obj.ID, false)
}

// Access is the resolver for the access field.
func (r *datasetResolver) Access(ctx context.Context, obj *models.Dataset) ([]*models.Access, error) {
	all, err := r.repo.ListAccessToDataset(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	dp, err := r.repo.GetDataproduct(ctx, obj.DataproductID)
	if err != nil {
		return nil, err
	}

	var ret []*models.Access

	user := auth.GetUser(ctx)
	if user == nil {
		return ret, nil
	}
	if user.GoogleGroups.Contains(dp.Owner.Group) {
		return all, nil
	}

	for _, a := range all {
		if strings.EqualFold(a.Subject, "user:"+user.Email) {
			ret = append(ret, a)
		} else if strings.HasPrefix(a.Subject, "group:") && user.GoogleGroups.Contains(strings.TrimPrefix(a.Subject, "group:")) {
			ret = append(ret, a)
		}
	}

	return ret, nil
}

// Services is the resolver for the services field.
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
		base := "https://metabase.intern.dev.nav.no/browse/%v"
		if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
			base = "https://metabase.intern.nav.no/browse/%v"
		}
		url := fmt.Sprintf(base, meta.DatabaseID)
		svc.Metabase = &url
	}

	return svc, nil
}

// Mappings is the resolver for the mappings field.
func (r *datasetResolver) Mappings(ctx context.Context, obj *models.Dataset) ([]models.MappingService, error) {
	return r.repo.GetDatasetMappings(ctx, obj.ID)
}

// CreateDataset is the resolver for the createDataset field.
func (r *mutationResolver) CreateDataset(ctx context.Context, input models.NewDataset) (*models.Dataset, error) {
	user := auth.GetUser(ctx)

	dp, err := r.repo.GetDataproduct(ctx, input.DataproductID)
	if err != nil {
		return nil, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}

	var referenceDatasource models.NewBigQuery
	var pseudoBigQuery *models.NewBigQuery
	if len(input.PseudoColumns) > 0 {
		projectID, datasetID, tableID, err := r.bigquery.CreatePseudonymisedView(ctx, input.BigQuery.ProjectID,
			input.BigQuery.Dataset, input.BigQuery.Table, input.PseudoColumns)
		if err != nil {
			return nil, err
		}

		referenceDatasource = input.BigQuery

		pseudoBigQuery = &models.NewBigQuery{
			ProjectID: projectID,
			Dataset:   datasetID,
			Table:     tableID,
			PiiTags:   input.BigQuery.PiiTags,
		}
	}

	metadata, err := r.prepareBigQuery(ctx, input.BigQuery, pseudoBigQuery, dp.Owner.Group)
	if err != nil {
		return nil, err
	}

	input.Metadata = metadata
	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	ds, err := r.repo.CreateDataset(ctx, input, &referenceDatasource, user)
	if err != nil {
		return nil, err
	}

	return ds, nil
}

// UpdateDataset is the resolver for the updateDataset field.
func (r *mutationResolver) UpdateDataset(ctx context.Context, id uuid.UUID, input models.UpdateDataset) (*models.Dataset, error) {
	ds, err := r.repo.GetDataset(ctx, id)
	if err != nil {
		return nil, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return nil, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}

	if input.DataproductID != nil && *input.DataproductID != ds.DataproductID {
		dp2, err := r.repo.GetDataproduct(ctx, *input.DataproductID)
		if err != nil {
			return nil, err
		}
		if err := ensureUserInGroup(ctx, dp2.Owner.Group); err != nil {
			return nil, err
		}
		if dp.Owner.Group != dp2.Owner.Group {
			return nil, errors.New("cannot move dataset to a dataproduct that is not owned by the same team")
		}
	} else {
		input.DataproductID = &ds.DataproductID
	}

	if len(input.PseudoColumns) > 0 {
		referenceDatasource, err := r.repo.GetBigqueryDatasource(ctx, id, true)
		if err != nil {
			return nil, err
		}
		_, _, _, err = r.bigquery.CreatePseudonymisedView(ctx, referenceDatasource.ProjectID,
			referenceDatasource.Dataset, referenceDatasource.Table, input.PseudoColumns)
		if err != nil {
			return nil, err
		}
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}
	return r.repo.UpdateDataset(ctx, id, input)
}

// DeleteDataset is the resolver for the deleteDataset field.
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

	return true, r.repo.DeleteDataset(ctx, ds.ID)
}

// MapDataset is the resolver for the mapDataset field.
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

// Dataset is the resolver for the dataset field.
func (r *queryResolver) Dataset(ctx context.Context, id uuid.UUID) (*models.Dataset, error) {
	return r.repo.GetDataset(ctx, id)
}

// AccessRequestsForDataset is the resolver for the accessRequestsForDataset field.
func (r *queryResolver) AccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]*models.AccessRequest, error) {
	ds, err := r.repo.GetDataset(ctx, datasetID)
	if err != nil {
		return nil, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return nil, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}

	return r.repo.ListAccessRequestsForDataset(ctx, datasetID)
}

// DatasetsInDataproduct is the resolver for the datasetsInDataproduct field.
func (r *queryResolver) DatasetsInDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]*models.Dataset, error) {
	return r.repo.GetDatasetsInDataproduct(ctx, dataproductID)
}

// AccessiblePseudoDatasets is the resolver for the accessiblePseudoDatasets field.
func (r *queryResolver) AccessiblePseudoDatasets(ctx context.Context) ([]*models.PseudoDataset, error) {
	user := auth.GetUser(ctx)
	subjectsAsOwner := []string{user.Email}
	subjectsAsOwner = append(subjectsAsOwner, user.GoogleGroups.Emails()...)
	subjectsAsAccesser := []string{"user:" + user.Email}
	for _, geml := range user.GoogleGroups.Emails() {
		subjectsAsAccesser = append(subjectsAsAccesser, "group:"+geml)
	}
	return r.repo.GetAccessiblePseudoDatasourcesByUser(ctx, subjectsAsOwner, subjectsAsAccesser)
}

// BigQuery returns generated.BigQueryResolver implementation.
func (r *Resolver) BigQuery() generated.BigQueryResolver { return &bigQueryResolver{r} }

// Dataset returns generated.DatasetResolver implementation.
func (r *Resolver) Dataset() generated.DatasetResolver { return &datasetResolver{r} }

type bigQueryResolver struct{ *Resolver }
type datasetResolver struct{ *Resolver }
