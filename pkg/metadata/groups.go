package metadata

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
)

type GoogleGroups struct {
	service *admin.Service
	mock    bool
}

func NewGoogleGroups(ctx context.Context, credentailFile, subject string) (*GoogleGroups, error) {
	if credentailFile == "" && subject == "" {
		logrus.Warn("Credential file and subject empty, running GoogleGroups with mock data")
		return &GoogleGroups{
			mock: true,
		}, nil
	}
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

func (g *GoogleGroups) GroupsForUser(ctx context.Context, email string) (groups Groups, err error) {
	if g.mock {
		return Groups{
			Group{Name: "All users", Email: "all-users@nav.no"},
			Group{Name: "Dataplattform", Email: "dataplattform@nav.no"},
			Group{Name: "nada", Email: "nada@nav.no"},
		}, nil
	}

	err = g.service.Groups.List().UserKey(email).Pages(ctx, func(g *admin.Groups) error {
		for _, grp := range g.Groups {
			groups = append(groups, Group{
				Name:  grp.Name,
				Email: grp.Email,
			})
		}
		return nil
	})

	fmt.Printf("%#v\n", groups)

	return groups, err
}
