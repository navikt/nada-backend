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
	panic(fmt.Errorf("not implemented: Owner - owner"))
}

// Datasource is the resolver for the datasource field.
func (r *datasetResolver) Datasource(ctx context.Context, obj *models.Dataset) (models.Datasource, error) {
	panic(fmt.Errorf("not implemented: Datasource - datasource"))
}

// Access is the resolver for the access field.
func (r *datasetResolver) Access(ctx context.Context, obj *models.Dataset) ([]*models.Access, error) {
	panic(fmt.Errorf("not implemented: Access - access"))
}

// Services is the resolver for the services field.
func (r *datasetResolver) Services(ctx context.Context, obj *models.Dataset) (*models.DatasetServices, error) {
	panic(fmt.Errorf("not implemented: Services - services"))
}

// Mappings is the resolver for the mappings field.
func (r *datasetResolver) Mappings(ctx context.Context, obj *models.Dataset) ([]models.MappingService, error) {
	panic(fmt.Errorf("not implemented: Mappings - mappings"))
}

// Requesters is the resolver for the requesters field.
func (r *datasetResolver) Requesters(ctx context.Context, obj *models.Dataset) ([]string, error) {
	panic(fmt.Errorf("not implemented: Requesters - requesters"))
}

// CreateDataset is the resolver for the createDataset field.
func (r *mutationResolver) CreateDataset(ctx context.Context, input models.NewDataset) (*models.Dataset, error) {
	panic(fmt.Errorf("not implemented: CreateDataset - createDataset"))
}

// UpdateDataset is the resolver for the updateDataset field.
func (r *mutationResolver) UpdateDataset(ctx context.Context, id uuid.UUID, input models.UpdateDataset) (*models.Dataset, error) {
	panic(fmt.Errorf("not implemented: UpdateDataset - updateDataset"))
}

// DeleteDataset is the resolver for the deleteDataset field.
func (r *mutationResolver) DeleteDataset(ctx context.Context, id uuid.UUID) (bool, error) {
	panic(fmt.Errorf("not implemented: DeleteDataset - deleteDataset"))
}

// MapDataset is the resolver for the mapDataset field.
func (r *mutationResolver) MapDataset(ctx context.Context, datasetID uuid.UUID, services []models.MappingService) (bool, error) {
	panic(fmt.Errorf("not implemented: MapDataset - mapDataset"))
}

// Dataset is the resolver for the dataset field.
func (r *queryResolver) Dataset(ctx context.Context, id uuid.UUID) (*models.Dataset, error) {
	panic(fmt.Errorf("not implemented: Dataset - dataset"))
}

// AccessRequestsForDataset is the resolver for the accessRequestsForDataset field.
func (r *queryResolver) AccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]*models.AccessRequest, error) {
	panic(fmt.Errorf("not implemented: AccessRequestsForDataset - accessRequestsForDataset"))
}

// DatasetsInDataproduct is the resolver for the datasetsInDataproduct field.
func (r *queryResolver) DatasetsInDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]*models.Dataset, error) {
	panic(fmt.Errorf("not implemented: DatasetsInDataproduct - datasetsInDataproduct"))
}

// BigQuery returns generated.BigQueryResolver implementation.
func (r *Resolver) BigQuery() generated.BigQueryResolver { return &bigQueryResolver{r} }

// Dataset returns generated.DatasetResolver implementation.
func (r *Resolver) Dataset() generated.DatasetResolver { return &datasetResolver{r} }

type bigQueryResolver struct{ *Resolver }
type datasetResolver struct{ *Resolver }
