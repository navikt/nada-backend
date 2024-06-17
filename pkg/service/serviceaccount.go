package service

import "context"

type ServiceAccountAPI interface {
	DeleteServiceAccount(ctx context.Context, gcpProject, saEmail string) error
	CreateServiceAccount(ctx context.Context, gcpProject string, ds *Dataset) ([]byte, string, error)
}
