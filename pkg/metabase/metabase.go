package metabase

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph"
)

type Metabase struct {
	repo      *database.Repo
	client    *Client
	accessMgr graph.AccessManager
	sa        string
	saEmail   string
}

func New(repo *database.Repo, client *Client, accessMgr graph.AccessManager, serviceAccount, serviceAccountEmail string) *Metabase {
	return &Metabase{
		repo:      repo,
		client:    client,
		accessMgr: accessMgr,
		sa:        serviceAccount,
		saEmail:   serviceAccountEmail,
	}
}

func (m *Metabase) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		if err := m.run(ctx); err != nil {
			log.Println("failed to run metabase", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Metabase) run(ctx context.Context) error {
	dps, err := m.repo.GetDataproductsByUserAccess(ctx, "group:all-users@nav.no")
	if err != nil {
		return err
	}

	log.Printf("Work on %v dataproducts", len(dps))

	databases, err := m.client.Databases(ctx)
	if err != nil {
		return err
	}

	lookup := map[string]Database{}
	for _, d := range databases {
		lookup[d.NadaID] = d
	}

	for _, dp := range dps {
		if _, ok := lookup[dp.ID.String()]; ok {
			// It exists in metabase
			continue
		}

		datasource, err := m.repo.GetBigqueryDatasource(ctx, dp.ID)
		if err != nil {
			return err
		}

		err = m.accessMgr.Grant(ctx, datasource.ProjectID, datasource.Dataset, datasource.Table, "serviceAccount:"+m.saEmail)
		if err != nil {
			return err
		}

		log.Printf("Create database")
		id, err := m.client.CreateDatabase(ctx, dp.Name, m.sa, &datasource)
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
		if err := m.HideOtherTables(ctx, id, datasource.Table); err != nil {
			return err
		}

		if err := m.client.AutoMapSemanticTypes(ctx, id); err != nil {
			return err
		}
	}

	// Remove databases in Metabase that no longer exists or is not available to all users
	for _, mdb := range databases {
		found := false
		for _, dp := range dps {
			if mdb.NadaID == dp.ID.String() {
				found = true
			}
		}
		if !found {
			if err := m.client.DeleteDatabase(ctx, strconv.Itoa(mdb.ID)); err != nil {
				// log error
				// inc err metrics
				continue
			}
			uid, err := uuid.Parse(mdb.NadaID)
			if err != nil {
				// log error
				// inc err metrics
				continue
			}
			ds, err := m.repo.GetBigqueryDatasource(ctx, uid)
			if err != nil {
				return err
			}
			if err := m.accessMgr.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount"+m.saEmail); err != nil {
				fmt.Println(err)
				// log error
				// inc err metrics
				continue
			}
		}
	}

	return nil
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
