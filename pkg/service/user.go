package service

import (
	"context"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type UserService interface {
	GetUserData(ctx context.Context, user *User) (*UserInfo, error)
}

type User struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	AzureGroups     Groups
	GoogleGroups    Groups
	AllGoogleGroups Groups
	Expiry          time.Time `json:"expiry"`
}

func (u User) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Email, validation.Required, is.Email),
	)
}

type Group struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Groups []Group

func (g Groups) Names() []string {
	names := make([]string, len(g))

	for i, g := range g {
		names[i] = g.Name
	}

	return names
}

func (g Groups) Emails() []string {
	emails := make([]string, len(g))

	for i, g := range g {
		emails[i] = g.Email
	}

	return emails
}

func (g Groups) Get(email string) (Group, bool) {
	for _, grp := range g {
		if grp.Email == email {
			return grp, true
		}
	}

	return Group{}, false
}

func (g Groups) Contains(email string) bool {
	_, ok := g.Get(email)
	return ok
}

type UserInfo struct {
	// name of user
	Name string `json:"name"`

	// email of user.
	Email string `json:"email"`

	// googleGroups is the google groups the user is member of.
	GoogleGroups Groups `json:"googleGroups"`

	// allGoogleGroups is the all the known google groups of the user domains.
	AllGoogleGroups Groups `json:"allGoogleGroups"`

	// azureGroups is the azure groups the user is member of.
	AzureGroups Groups `json:"azureGroups"`

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
	Stories []*Story `json:"stories"`

	// insight products is the insight products owned by the user's group
	InsightProducts []InsightProduct `json:"insightProducts"`

	// accessRequests is a list of access requests where either the user or one of the users groups is owner.
	AccessRequests []AccessRequest `json:"accessRequests"`

	// accessRequestsAsGranter is a list of access requests where one of the users groups is obliged to handle.
	AccessRequestsAsGranter []AccessRequestForGranter `json:"accessRequestsAsGranter"`
}
