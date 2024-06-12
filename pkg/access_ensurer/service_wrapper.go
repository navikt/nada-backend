package access_ensurer

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
)

type ServiceWrapper struct{}

func (s ServiceWrapper) GetUnrevokedExpiredAccess(ctx context.Context) ([]*service.Access, error) {
	return service.GetUnrevokedExpiredAccess(ctx)
}

func (s ServiceWrapper) GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID, isReference bool) (*service.BigQuery, *service.APIError) {
	return service.GetBigqueryDatasource(ctx, dataproductID, isReference)
}

func (s ServiceWrapper) RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error {
	return service.RevokeAccessToDataset(ctx, id.String())
}

func (s ServiceWrapper) GetPseudoDatasourcesToDelete(ctx context.Context) ([]*service.BigQuery, error) {
	return service.GetPseudoDatasourcesToDelete(ctx)
}

func (s ServiceWrapper) SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error {
	return service.SetDatasourceDeleted(ctx, id)
}

func (s ServiceWrapper) GetJoinableViewsWithReference(ctx context.Context) ([]gensql.GetJoinableViewsWithReferenceRow, error) {
	return service.GetJoinableViewsWithReference(ctx)
}

func (s ServiceWrapper) GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error) {
	return service.GetOwnerGroupOfDataset(ctx, datasetID)
}

func (s ServiceWrapper) ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*service.Access, error) {
	return service.ListActiveAccessToDataset(ctx, datasetID)
}

func (s ServiceWrapper) GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]gensql.GetJoinableViewsToBeDeletedWithRefDatasourceRow, error) {
	return service.GetJoinableViewsToBeDeletedWithRefDatasource(ctx)
}

func (s ServiceWrapper) SetJoinableViewDeleted(ctx context.Context, joinableViewID uuid.UUID) error {
	return service.SetJoinableViewDeleted(ctx, joinableViewID)
}
