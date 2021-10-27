package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen-contrib/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

var ErrUnauthorized = fmt.Errorf("unauthorized")

type GCP interface {
	GetTables(ctx context.Context, projectID, datasetID string) ([]*models.BigQueryTable, error)
	GetDatasets(ctx context.Context, projectID string) ([]string, error)
	TableExists(ctx context.Context, projectID string, datasetID string, tableID string) bool
}

type AccessManager interface {
	Grant(ctx context.Context, projectID, dataset, table, member string) error
	Revoke(ctx context.Context, projectID, dataset, table, member string) error
}

type SchemaUpdater interface {
	UpdateSchema(ctx context.Context, ds gensql.DatasourceBigquery) error
}

type Resolver struct {
	repo          *database.Repo
	gcp           GCP
	gcpProjects   *auth.TeamProjectsUpdater
	accessMgr     AccessManager
	schemaUpdater SchemaUpdater
	log           *logrus.Entry
}

func New(repo *database.Repo, gcp GCP, gcpProjects *auth.TeamProjectsUpdater, accessMgr AccessManager, schemaUpdater SchemaUpdater, log *logrus.Entry) *handler.Server {
	resolver := &Resolver{
		repo:          repo,
		gcp:           gcp,
		gcpProjects:   gcpProjects,
		accessMgr:     accessMgr,
		schemaUpdater: schemaUpdater,
		log:           log,
	}

	config := generated.Config{Resolvers: resolver}
	config.Directives.Authenticated = authenticate
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(config))
	srv.Use(prometheus.Tracer{})
	return srv
}

func (r *Resolver) ensureUserHasAccessToGcpProject(ctx context.Context, projectID string) error {
	user := auth.GetUser(ctx)

	for _, grp := range user.Groups {
		proj, ok := r.gcpProjects.Get(grp.Email)
		if !ok {
			continue
		}
		for _, p := range proj {
			if p == projectID {
				return nil
			}
		}
	}

	return ErrUnauthorized
}

func pagination(limit *int, offset *int) (int, int) {
	l := 15
	o := 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}
	return l, o
}

func ensureUserInGroup(ctx context.Context, group string) error {
	user := auth.GetUser(ctx)
	if user == nil || !user.Groups.Contains(group) {
		return ErrUnauthorized
	}
	return nil
}

func authenticate(ctx context.Context, obj interface{}, next graphql.Resolver, on *bool) (res interface{}, err error) {
	if auth.GetUser(ctx) == nil {
		// block calling the next resolver
		return nil, fmt.Errorf("access denied")
	}

	// or let it pass through
	return next(ctx)
}
