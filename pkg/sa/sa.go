package sa

// import (
// 	"context"
//
// 	validation "github.com/go-ozzo/ozzo-validation/v4"
// )
//
// type Operations interface {
// 	CreateServiceAccount(ctx context.Context, sa *ServiceAccount) ([]byte, string, error)
// 	AddPolicyBinding(ctx context.Context, gcpProject, role, member string) error
// 	DeleteServiceAccount(ctx context.Context, gcpProject, saEmail string) error
// }
//
// type ServiceAccount struct {
// 	ProjectID   string
// 	AccountID   string
// 	DisplayName string
// 	Description string
// }
//
// func (s ServiceAccount) Validate() error {
// 	return validation.ValidateStruct(&s,
// 		validation.Field(&s.ProjectID, validation.Required),
// 		validation.Field(&s.AccountID, validation.Required),
// 		validation.Field(&s.DisplayName, validation.Required),
// 		validation.Field(&s.Description, validation.Required),
// 	)
// }
//
// type Client struct {
// 	endpoint    string
// 	disableAuth bool
// }
