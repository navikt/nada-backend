package graph

import (
	"context"
	"fmt"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

var ErrUnauthorized = fmt.Errorf("unauthorized")

type GCP interface {
	GetTables(ctx context.Context, projectID, datasetID string) ([]*models.BigQueryTable, error)
	GetDatasets(ctx context.Context, projectID string) ([]string, error)
}

type Resolver struct {
	repo *database.Repo
	gcp  GCP
}

func New(repo *database.Repo, gcp GCP) *Resolver {
	return &Resolver{
		repo: repo,
		gcp:  gcp,
	}
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
