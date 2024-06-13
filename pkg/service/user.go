package service

import (
	"context"
	"time"

	"github.com/navikt/nada-backend/pkg/auth"
)

type UserService interface {
	GetUserData(ctx context.Context) (*UserInfo, error)
}

type UserInfo struct {
	// name of user
	Name string `json:"name"`

	// email of user.
	Email string `json:"email"`

	// googleGroups is the google groups the user is member of.
	GoogleGroups auth.Groups `json:"googleGroups"`

	// allGoogleGroups is the all the known google groups of the user domains.
	AllGoogleGroups auth.Groups `json:"allGoogleGroups"`

	// azureGroups is the azure groups the user is member of.
	AzureGroups auth.Groups `json:"azureGroups"`

	// gcpProjects is GCP projects the user is a member of.
	GcpProjects []GCPProject `json:"gcpProjects"`

	// nadaTokens is a list of the nada tokens for each team the logged in user is a part of.
	NadaTokens []NadaToken `json:"nadaTokens"`

	// loginExpiration is when the token expires.
	LoginExpiration time.Time `json:"loginExpiration"`

	// dataproducts is a list of dataproducts with one of the users groups as owner.
	Dataproducts []Dataproduct `json:"dataproducts"`

	// accessable is a list of datasets which the user has either owns or has explicit access to.
	Accessable AccessibleDatasets `json:"accessable"`

	// stories is the stories owned by the user's group
	Stories []Story `json:"stories"`

	// insight products is the insight products owned by the user's group
	InsightProducts []InsightProduct `json:"insightProducts"`

	// accessRequests is a list of access requests where either the user or one of the users groups is owner.
	AccessRequests []AccessRequest `json:"accessRequests"`

	// accessRequestsAsGranter is a list of access requests where one of the users groups is obliged to handle.
	AccessRequestsAsGranter []AccessRequestForGranter `json:"accessRequestsAsGranter"`
}
