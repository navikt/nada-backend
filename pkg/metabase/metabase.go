package metabase

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Metabase struct {
	repo      *database.Repo
	client    *Client
	accessMgr graph.AccessManager
	sa        string
	saEmail   string
	errs      *prometheus.CounterVec
	log       *logrus.Entry
}

func New(repo *database.Repo, client *Client, accessMgr graph.AccessManager, serviceAccount, serviceAccountEmail string, errs *prometheus.CounterVec, log *logrus.Entry) *Metabase {
	return &Metabase{
		repo:      repo,
		client:    client,
		accessMgr: accessMgr,
		sa:        serviceAccount,
		saEmail:   serviceAccountEmail,
		errs:      errs,
		log:       log,
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

		m.log.Infof("Created Metabase database: %v", dp.Name)
	}

	// Remove databases in Metabase that no longer exists or is not available to all users
	for _, mdb := range databases {
		found := false
		for _, dp := range dps {
			if mdb.NadaID == "" {
				continue
			} else if mdb.NadaID == dp.ID.String() {
				found = true
			}
		}
		if !found {
			if err := m.client.DeleteDatabase(ctx, strconv.Itoa(mdb.ID)); err != nil {
				m.log.WithError(err).Error("Deleting database in Metabase")
				m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
				continue
			}
			uid, err := uuid.Parse(mdb.NadaID)
			if err != nil {
				m.log.WithError(err).Error("Parsing UUID")
				m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
				continue
			}
			ds, err := m.repo.GetBigqueryDatasource(ctx, uid)
			if err != nil {
				m.log.WithError(err).Error("Getting Bigquery datasource")
				m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
				continue
			}
			if err := m.accessMgr.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+m.saEmail); err != nil {
				m.log.WithError(err).Error("Revoking IAM access")
				m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
				continue
			}
			m.log.Infof("Deleted Metabase database with ID: %v", mdb.ID)
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
