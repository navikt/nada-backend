package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/navikt/nada-backend/pkg/service"
	"google.golang.org/api/option"

	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
)

type GoogleGroupClient struct {
	service *admin.Service
}

func NewGoogleGroups(ctx context.Context, credentailFile, subject string) (*GoogleGroupClient, error) {
	jsonCredentials, err := os.ReadFile(credentailFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read service account key file %s", err)
	}

	config, err := google.JWTConfigFromJSON(jsonCredentials, admin.AdminDirectoryGroupMemberReadonlyScope, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse service account key file to config: %s", err)
	}

	config.Subject = subject

	s, err := admin.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Google Admin Client: %s", err)
	}

	if err != nil {
		return nil, err
	}

	return &GoogleGroupClient{
		service: s,
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
