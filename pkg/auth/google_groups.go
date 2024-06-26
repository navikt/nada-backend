package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
)

type GoogleGroupClient struct {
	service *admin.Service
	mock    bool
	log     *logrus.Entry
}

func NewGoogleGroups(ctx context.Context, credentailFile, subject string, log *logrus.Entry) (*GoogleGroupClient, error) {
	if credentailFile == "" && subject == "" {
		log.Warn("Credential file and subject empty, running GoogleGroups with mock data")
		return &GoogleGroupClient{
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

	if err != nil {
		return nil, err
	}

	return &GoogleGroupClient{
		service: service,
		log:     log,
	}, nil
}

func (g *GoogleGroupClient) Groups(ctx context.Context, email *string) (groups Groups, err error) {
	if g.mock {
		return Groups{
			Group{Name: "All users", Email: "all-users@nav.no"},
			Group{Name: "Dataplattform", Email: "nada@nav.no"},
			Group{Name: "nada", Email: "nada@nav.no"},
			Group{Name: "nais-team-nyteam", Email: "nais-team-nyteam@nav.no"},
		}, nil
	}

	groupListCall := g.service.Groups.List().Customer(`my_customer`)
	if email != nil {
		groupListCall = g.service.Groups.List().UserKey(*email)
	}
	err = groupListCall.Pages(ctx, func(g *admin.Groups) error {
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

func TrimNaisTeamPrefix(team string) string {
	return strings.TrimPrefix(team, "nais-team-")
}
