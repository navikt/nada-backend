package service

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ServiceAccountAPI interface {
	// ListServiceAccounts returns a list of service accounts in the project
	// with their keys and role bindings.
	ListServiceAccounts(ctx context.Context, project string) ([]*ServiceAccount, error)

	// EnsureServiceAccountWithKeyAndBinding creates a service account in the project, and adds the
	// specified role binding to the service account at a project level.
	EnsureServiceAccountWithKeyAndBinding(ctx context.Context, sa *ServiceAccountRequest) (*ServiceAccountWithPrivateKey, error)

	// DeleteServiceAccountAndBindings deletes a service account and its role bindings
	// in the project. Deleting the service account will also delete all associated keys.
	DeleteServiceAccountAndBindings(ctx context.Context, project, email string) error
}

type Binding struct {
	Role    string
	Members []string
}

func (b Binding) Validate() error {
	return validation.ValidateStruct(&b,
		validation.Field(&b.Role, validation.Required),
		validation.Field(&b.Members, validation.Required),
	)
}

type ServiceAccountRequest struct {
	ProjectID   string
	AccountID   string
	DisplayName string
	Description string
	Binding     *Binding
}

func (s ServiceAccountRequest) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.ProjectID, validation.Required),
		validation.Field(&s.AccountID, validation.Required),
		validation.Field(&s.DisplayName, validation.Required),
		validation.Field(&s.Description, validation.Required),
		validation.Field(&s.Binding, validation.Required),
	)
}

type ServiceAccountMeta struct {
	Description string
	DisplayName string
	Email       string
	Name        string
	ProjectId   string
	UniqueId    string
}

type ServiceAccount struct {
	*ServiceAccountMeta
	Keys     []*ServiceAccountKey
	Bindings []*Binding
}

type ServiceAccountWithPrivateKey struct {
	*ServiceAccountMeta
	Key *ServiceAccountKeyWithPrivateKeyData
}

type ServiceAccountKey struct {
	Name         string
	KeyAlgorithm string
	KeyOrigin    string
	KeyType      string
}

type ServiceAccountKeyWithPrivateKeyData struct {
	*ServiceAccountKey
	PrivateKeyData []byte
}
