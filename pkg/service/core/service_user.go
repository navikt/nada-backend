package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"strings"
)

var _ service.UserService = &userService{}

type userService struct {
	accessStorage         service.AccessStorage
	tokenStorage          service.TokenStorage
	storyStorage          service.StoryStorage
	dataProductStorage    service.DataProductsStorage
	insightProductStorage service.InsightProductStorage
	teamProjectsMapping   *auth.TeamProjectsMapping
}

func (s *userService) GetUserData(ctx context.Context) (*service.UserInfo, error) {
	user := auth.GetUser(ctx)
	if user == nil {
		return nil, fmt.Errorf("no user found in context")
	}

	userData := &service.UserInfo{
		Name:            user.Name,
		Email:           user.Email,
		GoogleGroups:    user.GoogleGroups,
		LoginExpiration: user.Expiry,
		AllGoogleGroups: user.AllGoogleGroups,
		AzureGroups:     user.AzureGroups,
	}

	teams := teamNamesFromGroups(userData.GoogleGroups)
	tokens, err := s.tokenStorage.GetNadaTokensForTeams(ctx, teams)
	if err != nil {
		return nil, fmt.Errorf("failed to get nada tokens for teams: %w", err)
	}

	for _, grp := range user.GoogleGroups {
		proj, ok := s.teamProjectsMapping.Get(auth.TrimNaisTeamPrefix(grp.Email))
		if !ok {
			continue
		}

		userData.GcpProjects = append(userData.GcpProjects, service.GCPProject{
			ID: proj,
			Group: &auth.Group{
				Name:  grp.Name,
				Email: grp.Email,
			},
		})
	}

	userData.NadaTokens = tokens

	dpwithds, dar, err := s.dataProductStorage.GetDataproductsWithDatasetsAndAccessRequests(ctx, []uuid.UUID{}, userData.GoogleGroups.Emails())
	if err != nil {
		return nil, fmt.Errorf("failed to get dataproducts by group from database: %w", err)
	}

	for _, dpds := range dpwithds {
		userData.Dataproducts = append(userData.Dataproducts, dpds.Dataproduct)
	}

	userData.AccessRequestsAsGranter = dar

	owned, granted, apiErr := s.dataProductStorage.GetAccessibleDatasets(ctx, userData.GoogleGroups.Emails(), "user:"+strings.ToLower(user.Email))
	if apiErr != nil {
		return nil, apiErr
	}

	userData.Accessable = service.AccessibleDatasets{
		Owned:   owned,
		Granted: granted,
	}

	dbStories, err := s.storyStorage.GetStoriesWithTeamkatalogenByGroups(ctx, user.GoogleGroups.Emails())
	if err != nil {
		return nil, fmt.Errorf("failed to get stories with teamkatalogen by groups: %w", err)
	}

	userData.Stories = dbStories

	dbProducts, err := s.insightProductStorage.GetInsightProductsByGroups(ctx, user.GoogleGroups.Emails())
	if err != nil {
		return nil, err
	}

	for _, p := range dbProducts {
		userData.InsightProducts = append(userData.InsightProducts, *p)
	}

	groups := []string{"user:" + strings.ToLower(user.Email)}
	for _, g := range user.GoogleGroups {
		groups = append(groups, "group:"+strings.ToLower(g.Email))
	}

	accessRequestSQLs, err := s.accessStorage.ListAccessRequestsForOwner(ctx, groups)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	} else if err == nil {
		for _, ar := range accessRequestSQLs {
			userData.AccessRequests = append(userData.AccessRequests, *ar)
		}
		if err != nil {
			return nil, err
		}
	}

	return userData, nil
}

func teamNamesFromGroups(groups auth.Groups) []string {
	teams := []string{}
	for _, g := range groups {
		teams = append(teams, auth.TrimNaisTeamPrefix(strings.Split(g.Email, "@")[0]))
	}

	return teams
}

func NewUserService() *userService {
	return &userService{}
}
