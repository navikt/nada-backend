package metabase

import (
	"context"

	"github.com/navikt/nada-backend/pkg/database"
)

type Metabase struct {
	repo   *database.Repo
	client *Client
}

func New(repo *database.Repo, url, username, password string) *Metabase {
	return &Metabase{
		repo:   repo,
		client: NewClient(url, username, password),
	}
}

func (m *Metabase) Run(ctx context.Context) {
	// dbs, err := m.client.Databases(ctx)
}

func (m *Metabase) HideOtherTables(ctx context.Context, dbID, table string) error {
	tables, err := m.client.Tables(ctx, dbID)
	if err != nil {
		return err
	}
	other := []int{}
	for _, t := range tables {
		if t.Name != table {
			other = append(other, t.ID)
		}
	}

	if len(other) == 0 {
		return nil
	}
	return m.client.HideTables(ctx, other)
}
