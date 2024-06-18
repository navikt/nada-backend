package gcp

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	"net/http"
	"strings"
)

var _ service.ServiceAccountAPI = &serviceAccountAPI{}

type serviceAccountAPI struct {
}

func (a *serviceAccountAPI) DeleteServiceAccount(ctx context.Context, gcpProject, saEmail string) error {
	const op errs.Op = "gcp.DeleteServiceAccount"

	iamService, err := iam.NewService(ctx)
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	_, err = iamService.Projects.ServiceAccounts.
		Delete("projects/" + gcpProject + "/serviceAccounts/" + saEmail).
		Do()
	if err != nil {
		var apiError *googleapi.Error

		ok := errors.As(err, &apiError)
		if ok {
			if apiError.Code == http.StatusNotFound {
				return nil
			}
		}

		return errs.E(errs.IO, op, err)
	}

	return nil
}

func (a *serviceAccountAPI) CreateServiceAccount(ctx context.Context, gcpProject string, ds *service.Dataset) ([]byte, string, error) {
	const op errs.Op = "gcp.CreateServiceAccount"

	request := &iam.CreateServiceAccountRequest{
		AccountId: "nada-" + MarshalUUID(ds.ID),
		ServiceAccount: &iam.ServiceAccount{
			Description: "Metabase service account for dataset " + ds.ID.String(),
			DisplayName: ds.Name,
		},
	}

	iamService, err := iam.NewService(ctx)
	if err != nil {
		return nil, "", errs.E(errs.IO, op, err)
	}

	account, err := iamService.Projects.ServiceAccounts.Create("projects/"+gcpProject, request).Do()
	if err != nil {
		return nil, "", errs.E(errs.IO, op, err)
	}

	crmService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, "", errs.E(errs.IO, op, err)
	}

	iamPolicyCall := crmService.Projects.GetIamPolicy(gcpProject, &cloudresourcemanager.GetIamPolicyRequest{})
	iamPolicies, err := iamPolicyCall.Do()
	if err != nil {
		return nil, "", errs.E(errs.IO, op, err)
	}

	iamPolicies.Bindings = append(iamPolicies.Bindings, &cloudresourcemanager.Binding{
		Members: []string{"serviceAccount:" + account.Email},
		Role:    "projects/" + gcpProject + "/roles/nada.metabase",
	})

	iamSetPolicyCall := crmService.Projects.SetIamPolicy(gcpProject, &cloudresourcemanager.SetIamPolicyRequest{
		Policy: iamPolicies,
	})

	_, err = iamSetPolicyCall.Do()
	if err != nil {
		return nil, "", errs.E(errs.IO, op, err)
	}

	keyRequest := &iam.CreateServiceAccountKeyRequest{}

	key, err := iamService.Projects.ServiceAccounts.Keys.Create("projects/-/serviceAccounts/"+account.UniqueId, keyRequest).Do()
	if err != nil {
		return nil, "", errs.E(errs.IO, op, err)
	}

	saJson, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return nil, "", errs.E(errs.IO, op, err)
	}

	return saJson, account.Email, nil
}

func MarshalUUID(id uuid.UUID) string {
	return strings.ToLower(base58.Encode(id[:]))
}

func NewServiceAccountAPI() *serviceAccountAPI {
	return &serviceAccountAPI{}
}
