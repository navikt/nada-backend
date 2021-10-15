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

func NewGoogleGroups(ctx context.Context, credentailFile, subject string) (*GoogleGroups, error) {
	jsonCredentials, err := ioutil.ReadFile(credentailFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read service account key file %s", err)
	}

	config, err := google.JWTConfigFromJSON(jsonCredentials, admin.AdminDirectoryGroupMemberReadonlyScope, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse service account key file to config: %s", err)
	}

	config.Subject = subject
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

type Group struct {
	Name  string
	Email string
}

type Groups []Group

func (g Groups) Names() []string {
	names := []string{}
	for _, g := range g {
		names = append(names, g.Name)
	}
	return names
}

func (g *GoogleGroups) GroupsForUser(ctx context.Context, email string) (groups Groups, err error) {
	err = g.service.Groups.List().UserKey(email).Pages(ctx, func(g *admin.Groups) error {
		for _, grp := range g.Groups {
			groups = append(groups, Group{
				Name:  grp.Name,
				Email: grp.Email,
			})
		}
		return nil
	})

	return groups, err
}
