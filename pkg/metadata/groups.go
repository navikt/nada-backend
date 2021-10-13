package metadata

import (
	"context"
	"fmt"
	"io/ioutil"

	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
)

type GoogleGroups struct {
	service *admin.Service
}

func NewGoogleGroups(ctx context.Context) (*GoogleGroups, error) {
	jsonCredentials, err := ioutil.ReadFile("../../test-sa.json")
	if err != nil {
		return nil, fmt.Errorf("unable to read service account key file %s", err)
	}

	config, err := google.JWTConfigFromJSON(jsonCredentials, admin.AdminDirectoryGroupMemberReadonlyScope, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse service account key file to config: %s", err)
	}

	config.Subject = "johnny.horvi@nav.no"
	client := config.Client(ctx)

	service, err := admin.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Google Admin Client: %s", err)
	}

	// service, err := admin.NewService(ctx, option.ImpersonateCredentials(target string, delegates ...string))
	if err != nil {
		return nil, err
	}

	return &GoogleGroups{
		service: service,
	}, nil
}

func (g *GoogleGroups) ForUser(ctx context.Context, email string) {
	groups := []*admin.Group{}
	err := g.service.Groups.List().UserKey(email).Pages(ctx, func(g *admin.Groups) error {
		groups = append(groups, g.Groups...)
		return nil
	})
	if err != nil {
		panic(err)
	}

	for _, g := range groups {
		fmt.Println(g.Name, g.Email)
	}
}
