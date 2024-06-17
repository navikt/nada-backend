package service

type ServiceAccountAPI interface {
	DeleteServiceAccount(gcpProject, saEmail string) error
	CreateServiceAccount(gcpProject string, ds *Dataset) ([]byte, string, error)
}
