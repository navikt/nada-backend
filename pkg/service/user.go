package service

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type UserStorage interface {
}

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

func GetUserData(ctx context.Context) (*UserInfo, *APIError) {
	user := auth.GetUser(ctx)
	if user == nil {
		return nil, NewAPIError(http.StatusUnauthorized, errors.New("authentication error"), "getUserInfo(): no user session found")
	}

	userData := &UserInfo{
		Name:            user.Name,
		Email:           user.Email,
		GoogleGroups:    user.GoogleGroups,
		LoginExpiration: user.Expiry,
		AllGoogleGroups: user.AllGoogleGroups,
		AzureGroups:     user.AzureGroups,
	}

	teams := teamNamesFromGroups(userData.GoogleGroups)
	tokens, err := GetNadaTokensForTeams(ctx, teams)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "getUserInfo(): getting nada tokens for teams")
	}

	for _, grp := range user.GoogleGroups {
		proj, ok := teamProjectsMapping.Get(auth.TrimNaisTeamPrefix(grp.Email))
		if !ok {
			continue
		}

		userData.GcpProjects = append(userData.GcpProjects, GCPProject{
			ID: proj,
			Group: &auth.Group{
				Name:  grp.Name,
				Email: grp.Email,
			},
		})
	}

	userData.NadaTokens = tokens

	dpres, err := queries.GetDataproductsWithDatasetsAndAccessRequests(ctx, gensql.GetDataproductsWithDatasetsAndAccessRequestsParams{
		Ids:    []uuid.UUID{},
		Groups: userData.GoogleGroups.Emails(),
	})
	if err != nil && err != sql.ErrNoRows {
		return nil, DBErrorToAPIError(err, "getting dataproducts by group from database")
	} else {
		dpwithds, dar, e := dataproductsWithDatasetAndAccessRequestsForGranterFromSQL(dpres)

		if e != nil {
			return nil, NewAPIError(http.StatusInternalServerError, e, "getUserInfo(): converting access requests from database")
		}
		for _, dpds := range dpwithds {
			userData.Dataproducts = append(userData.Dataproducts, dpds.Dataproduct)
		}
		userData.AccessRequestsAsGranter = dar
	}

	owned, granted, apiErr := getAccessibleDatasets(ctx, userData.GoogleGroups.Emails(), "user:"+strings.ToLower(user.Email))
	if apiErr != nil {
		return nil, apiErr
	}

	userData.Accessable = AccessibleDatasets{
		Owned:   owned,
		Granted: granted,
	}

	dbStories, err := queries.GetStoriesWithTeamkatalogenByGroups(ctx, user.GoogleGroups.Emails())
	if err != nil {
		return nil, DBErrorToAPIError(err, "getUserInfo(): getting stories by group from database")
	}

	for _, s := range dbStories {
		userData.Stories = append(userData.Stories, *storyFromSQL(&s))
	}

	dbProducts, err := queries.GetInsightProductsByGroups(ctx, user.GoogleGroups.Emails())
	if err != nil {
		return nil, DBErrorToAPIError(err, "getUserInfo(): getting insight products by group from database")
	}

	for _, p := range dbProducts {
		userData.InsightProducts = append(userData.InsightProducts, *insightProductFromSQL(&p))
	}

	groups := []string{"user:" + strings.ToLower(user.Email)}
	for _, g := range user.GoogleGroups {
		groups = append(groups, "group:"+strings.ToLower(g.Email))
	}

	accessRequestSQLs, err := queries.ListAccessRequestsForOwner(ctx, groups)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, DBErrorToAPIError(err, "getUserInfo(): getting access requests by owner from database")
	} else if err == nil {
		userData.AccessRequests, err = AccessRequestsFromSQL(ctx, accessRequestSQLs)
		if err != nil {
			return nil, NewAPIError(http.StatusInternalServerError, err, "getUserInfo(): converting access requests from database")
		}
	}

	return userData, nil
}
