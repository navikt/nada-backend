package core

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
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

func (s *userService) GetUserData(ctx context.Context, user *service.User) (*service.UserInfo, error) {
	const op = "userService.GetUserData"

	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, fmt.Errorf("no user found in context"))
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
		return nil, errs.E(op, errs.Parameter("team"), err)
	}

	for _, grp := range user.GoogleGroups {
		proj, ok := s.teamProjectsMapping.Get(auth.TrimNaisTeamPrefix(grp.Email))
		if !ok {
			continue
		}

		userData.GcpProjects = append(userData.GcpProjects, service.GCPProject{
			ID: proj,
			Group: &service.Group{
				Name:  grp.Name,
				Email: grp.Email,
			},
		})
	}

	userData.NadaTokens = tokens

	dpwithds, dar, err := s.dataProductStorage.GetDataproductsWithDatasetsAndAccessRequests(ctx, []uuid.UUID{}, userData.GoogleGroups.Emails())
	if err != nil {
		return nil, errs.E(op, err)
	}

	for _, dpds := range dpwithds {
		userData.Dataproducts = append(userData.Dataproducts, dpds.Dataproduct)
	}

	userData.AccessRequestsAsGranter = dar

	owned, granted, err := s.dataProductStorage.GetAccessibleDatasets(ctx, userData.GoogleGroups.Emails(), "user:"+strings.ToLower(user.Email))
	if err != nil {
		return nil, errs.E(op, err)
	}

	userData.Accessable = service.AccessibleDatasets{
		Owned:   owned,
		Granted: granted,
	}

	dbStories, err := s.storyStorage.GetStoriesWithTeamkatalogenByGroups(ctx, user.GoogleGroups.Emails())
	if err != nil {
		return nil, errs.E(op, err)
	}

	userData.Stories = dbStories

	dbProducts, err := s.insightProductStorage.GetInsightProductsByGroups(ctx, user.GoogleGroups.Emails())
	if err != nil {
		return nil, errs.E(op, err)
	}

	for _, p := range dbProducts {
		userData.InsightProducts = append(userData.InsightProducts, *p)
	}

	groups := []string{"user:" + strings.ToLower(user.Email)}
	for _, g := range user.GoogleGroups {
		groups = append(groups, "group:"+strings.ToLower(g.Email))
	}

	accessRequestSQLs, err := s.accessStorage.ListAccessRequestsForOwner(ctx, groups)
	if err != nil {
		if !errs.KindIs(errs.NotExist, err) {
			return nil, errs.E(op, err)
		}
	}

	for _, ar := range accessRequestSQLs {
		userData.AccessRequests = append(userData.AccessRequests, *ar)
	}

	return userData, nil
}

func teamNamesFromGroups(groups service.Groups) []string {
	var teams []string
	for _, g := range groups {
		teams = append(teams, auth.TrimNaisTeamPrefix(strings.Split(g.Email, "@")[0]))
	}

	return teams
}

func NewUserService(
	accessStorage service.AccessStorage,
	tokenStorage service.TokenStorage,
	storyStorage service.StoryStorage,
	dataProductStorage service.DataProductsStorage,
	insightProductStorage service.InsightProductStorage,
	teamProjectsMapping *auth.TeamProjectsMapping,
) *userService {
	return &userService{
		accessStorage:         accessStorage,
		tokenStorage:          tokenStorage,
		storyStorage:          storyStorage,
		dataProductStorage:    dataProductStorage,
		insightProductStorage: insightProductStorage,
		teamProjectsMapping:   teamProjectsMapping,
	}
}
