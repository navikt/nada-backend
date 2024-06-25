package auth

import (
	"context"
	"fmt"
	"github.com/navikt/nada-backend/pkg/service"
	"os"
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
	jsonCredentials, err := os.ReadFile(credentailFile)
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

func (g *GoogleGroupClient) Groups(ctx context.Context, email *string) (service.Groups, error) {
	groupListCall := g.service.Groups.List().Customer(`my_customer`)
	if email != nil {
		groupListCall = g.service.Groups.List().UserKey(*email)
	}

	var groups service.Groups

	err := groupListCall.Pages(ctx, func(g *admin.Groups) error {
		for _, grp := range g.Groups {
			groups = append(groups, service.Group{
				Name:  grp.Name,
				Email: grp.Email,
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list groups: %s", err)
	}

	return groups, nil
}

func TrimNaisTeamPrefix(team string) string {
	return strings.TrimPrefix(team, "nais-team-")
}
