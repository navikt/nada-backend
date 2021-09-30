// Code generated by sqlc. DO NOT EDIT.

package gensql

import (
	"context"

	"github.com/google/uuid"
)

type Querier interface {
	CreateDataproduct(ctx context.Context, arg CreateDataproductParams) (Dataproduct, error)
	CreateDataset(ctx context.Context, arg CreateDatasetParams) (Dataset, error)
	DeleteDataproduct(ctx context.Context, id uuid.UUID) error
	DeleteDataset(ctx context.Context, id uuid.UUID) error
	GetDataproduct(ctx context.Context, id uuid.UUID) (Dataproduct, error)
	GetDataproducts(ctx context.Context) ([]Dataproduct, error)
	GetDataset(ctx context.Context, id uuid.UUID) (Dataset, error)
	UpdateDataproduct(ctx context.Context, arg UpdateDataproductParams) (Dataproduct, error)
	UpdateDataset(ctx context.Context, arg UpdateDatasetParams) (Dataset, error)
}

var _ Querier = (*Queries)(nil)
