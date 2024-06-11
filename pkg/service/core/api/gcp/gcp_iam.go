package gcp

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/service"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	"strings"
)

type serviceAccountAPI struct {
	iams *iam.Service
	crms *cloudresourcemanager.Service
}

func (a *serviceAccountAPI) DeleteServiceAccount(gcpProject, saEmail string) error {
	_, err := a.iams.Projects.ServiceAccounts.
		Delete("projects/" + gcpProject + "/serviceAccounts/" + saEmail).
		Do()
	if err != nil {
		var apiError *googleapi.Error

		ok := errors.As(err, &apiError)
		if ok {
			if apiError.Code == 404 {
				return nil
			}
		}

		return fmt.Errorf("delete service account: %w", err)
	}

	return nil
}

func (a *serviceAccountAPI) CreateServiceAccount(gcpProject string, ds *service.Dataset) ([]byte, string, error) {
	request := &iam.CreateServiceAccountRequest{
		AccountId: "nada-" + MarshalUUID(ds.ID),
		ServiceAccount: &iam.ServiceAccount{
			Description: "Metabase service account for dataset " + ds.ID.String(),
			DisplayName: ds.Name,
		},
	}

	account, err := a.iams.Projects.ServiceAccounts.Create("projects/"+gcpProject, request).Do()
	if err != nil {
		return nil, "", fmt.Errorf("create service account: %w", err)
	}

	iamPolicyCall := a.crms.Projects.GetIamPolicy(gcpProject, &cloudresourcemanager.GetIamPolicyRequest{})
	iamPolicies, err := iamPolicyCall.Do()
	if err != nil {
		return nil, "", fmt.Errorf("get iam policies: %w", err)
	}

	iamPolicies.Bindings = append(iamPolicies.Bindings, &cloudresourcemanager.Binding{
		Members: []string{"serviceAccount:" + account.Email},
		Role:    "projects/" + gcpProject + "/roles/nada.metabase",
	})

	iamSetPolicyCall := a.crms.Projects.SetIamPolicy(gcpProject, &cloudresourcemanager.SetIamPolicyRequest{
		Policy: iamPolicies,
	})

	_, err = iamSetPolicyCall.Do()
	if err != nil {
		return nil, "", err
	}

	keyRequest := &iam.CreateServiceAccountKeyRequest{}

	key, err := a.iams.Projects.ServiceAccounts.Keys.Create("projects/-/serviceAccounts/"+account.UniqueId, keyRequest).Do()
	if err != nil {
		return nil, "", err
	}

	saJson, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return nil, "", err
	}

	return saJson, account.Email, err
}

func MarshalUUID(id uuid.UUID) string {
	return strings.ToLower(base58.Encode(id[:]))
}

func NewServiceAccountAPI(iams *iam.Service, crms *cloudresourcemanager.Service) *serviceAccountAPI {
	return &serviceAccountAPI{
		iams: iams,
		crms: crms,
	}
}
