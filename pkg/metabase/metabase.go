package metabase

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	iam "google.golang.org/api/iam/v1"
)

type Metabase struct {
	repo      *database.Repo
	client    *Client
	accessMgr graph.AccessManager
	sa        string
	saEmail   string
	errs      *prometheus.CounterVec
	service   *iam.Service
	log       *logrus.Entry
}

type dpWrapper struct {
	Dataproduct *models.Dataproduct
	Key         string
	Email       string
}

func New(repo *database.Repo, client *Client, accessMgr graph.AccessManager, serviceAccount, serviceAccountEmail string, errs *prometheus.CounterVec, service *iam.Service, log *logrus.Entry) *Metabase {
	return &Metabase{
		repo:      repo,
		client:    client,
		accessMgr: accessMgr,
		sa:        serviceAccount,
		saEmail:   serviceAccountEmail,
		errs:      errs,
		service:   service,
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
	openDps, err := m.repo.GetDataproductsByUserAccess(ctx, "group:all-users@nav.no")
	if err != nil {
		return err
	}

	restrictedDps, err := m.repo.GetDataproductsByMapping(ctx, models.MappingServiceMetabase)
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

	combinedDps := []dpWrapper{}
	combinedIDs := map[string]bool{}

	for _, dp := range openDps {
		if _, ok := lookup[dp.ID.String()]; ok {
			// It exists in metabase
			continue
		}

		combinedDps = append(combinedDps, dpWrapper{
			Dataproduct: dp,
			Key:         m.sa,
			Email:       m.saEmail,
		})
		combinedIDs[dp.ID.String()] = true
	}

	for _, dp := range restrictedDps {
		if _, ok := lookup[dp.ID.String()]; ok {
			// It exists in metabase
			continue
		} else if combinedIDs[dp.ID.String()] {
			continue
		}

		err := m.client.CreatePermissionGroup(ctx, dp.ID.String())
		if err != nil {
			return err
		}
		key, email, err := m.createServiceAccount(dp)
		if err != nil {
			return err
		}

		combinedDps = append(combinedDps, dpWrapper{
			Dataproduct: dp,
			Key:         string(key),
			Email:       email,
		})
	}

	err = m.create(ctx, combinedDps)
	if err != nil {
		return err
	}
	err = m.delete(ctx, combinedDps, databases)
	if err != nil {
		return err
	}

	return nil
}

func (m *Metabase) create(ctx context.Context, dps []dpWrapper) error {
	for _, dp := range dps {
		datasource, err := m.repo.GetBigqueryDatasource(ctx, dp.Dataproduct.ID)
		if err != nil {
			return err
		}

		err = m.accessMgr.Grant(ctx, datasource.ProjectID, datasource.Dataset, datasource.Table, "serviceAccount:"+dp.Email)
		if err != nil {
			return err
		}

		id, err := m.client.CreateDatabase(ctx, dp.Dataproduct.Name, dp.Key, dp.Email, &datasource)
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

		m.log.Infof("Created Metabase database: %v", dp.Dataproduct.Name)
	}
	return nil
}

func (m *Metabase) delete(ctx context.Context, dataproducts []dpWrapper, databases []Database) error {
	// Remove databases in Metabase that no longer exists or is not available to all users
	for _, mdb := range databases {
		if mdb.NadaID == "" {
			continue
		}
		found := false

		for _, dp := range dataproducts {
			if mdb.NadaID == dp.Dataproduct.ID.String() {
				found = true
			}
		}
		if found {
			continue
		}
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
		if err := m.accessMgr.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mdb.SAEmail); err != nil {
			m.log.WithError(err).Error("Revoking IAM access")
			m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
			continue
		}
		m.log.Infof("Deleted Metabase database with ID: %v", mdb.ID)
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

func (m *Metabase) createServiceAccount(dp *models.Dataproduct) ([]byte, string, error) {
	request := &iam.CreateServiceAccountRequest{
		AccountId: "nada-" + MarshalUUID(dp.ID),
		ServiceAccount: &iam.ServiceAccount{
			Description: "Metabase service account for dataproduct " + dp.ID.String(),
			DisplayName: dp.Name,
		},
	}

	account, err := m.service.Projects.ServiceAccounts.Create("projects/nada-prod-6977", request).Do()
	if err != nil {
		return nil, "", err
	}

	keyRequest := &iam.CreateServiceAccountKeyRequest{}

	key, err := m.service.Projects.ServiceAccounts.Keys.Create("projects/nada-prod-6977/serviceAccounts/"+account.UniqueId, keyRequest).Do()
	if err != nil {
		return nil, "", err
	}

	saJson, err := key.MarshalJSON()
	if err != nil {
		return nil, "", err
	}
	return saJson, account.Email, err
}

func containsAccessGroup(accessList []*models.Access, group string) bool {
	for _, a := range accessList {
		if a.Subject == group {
			return true
		}
	}
	return false
}

func MarshalUUID(id uuid.UUID) string {
	return base58.Encode(id[:])
}
