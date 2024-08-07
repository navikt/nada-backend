package service

import "context"

type ServiceAccountAPI interface {
	ListServiceAccounts(ctx context.Context, gcpProject string) ([]string, error)
	DeleteServiceAccount(ctx context.Context, gcpProject, saEmail string) error
	CreateServiceAccount(ctx context.Context, gcpProject string, ds *Dataset) ([]byte, string, error)
}
