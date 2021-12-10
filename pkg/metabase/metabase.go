package metabase

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/cloudresourcemanager/v1"
	iam "google.golang.org/api/iam/v1"
)

type Metabase struct {
	repo       *database.Repo
	client     *Client
	accessMgr  graph.AccessManager
	sa         string
	saEmail    string
	errs       *prometheus.CounterVec
	iamService *iam.Service
	crmService *cloudresourcemanager.Service
	log        *logrus.Entry
}

type dpWrapper struct {
	Dataproduct     *models.Dataproduct
	Key             string
	Email           string
	MetabaseGroupID int
}

func New(repo *database.Repo, client *Client, accessMgr graph.AccessManager, serviceAccount, serviceAccountEmail string, errs *prometheus.CounterVec, iamService *iam.Service, crmService *cloudresourcemanager.Service, log *logrus.Entry) *Metabase {
	return &Metabase{
		repo:       repo,
		client:     client,
		accessMgr:  accessMgr,
		sa:         serviceAccount,
		saEmail:    serviceAccountEmail,
		errs:       errs,
		iamService: iamService,
		crmService: crmService,
		log:        log,
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

	createDps := []dpWrapper{}
	combinedIDs := map[string]bool{}

	for _, dp := range openDps {
		combinedIDs[dp.ID.String()] = true
		if _, ok := lookup[dp.ID.String()]; ok {
			// It exists in metabase
			continue
		}

		createDps = append(createDps, dpWrapper{
			Dataproduct: dp,
			Key:         m.sa,
			Email:       m.saEmail,
		})
	}

	for _, dp := range restrictedDps {
		if combinedIDs[dp.ID.String()] {
			continue
		}
		combinedIDs[dp.ID.String()] = true
		if _, ok := lookup[dp.ID.String()]; ok {
			// It exists in metabase
			continue
		}

		groupID, err := m.client.CreatePermissionGroup(ctx, dp.Name)
		if err != nil {
			return err
		}

		_, err = m.client.CreateCollectionWithAccess(ctx, groupID, dp.Name)
		if err != nil {
			return err
		}

		key, email, err := m.createServiceAccount(dp)
		if err != nil {
			return err
		}

		createDps = append(createDps, dpWrapper{
			Dataproduct:     dp,
			Key:             string(key),
			Email:           email,
			MetabaseGroupID: groupID,
		})
	}

	err = m.create(ctx, createDps)
	if err != nil {
		return err
	}
	err = m.delete(ctx, combinedIDs, databases)
	if err != nil {
		return err
	}

	err = m.syncPermissionGroupMembers(ctx, restrictedDps)
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

		dbID, err := m.client.CreateDatabase(ctx, dp.Dataproduct.Name, dp.Key, dp.Email, &datasource)
		if err != nil {
			return err
		}

		err = m.repo.CreateMetabaseMetadata(ctx, models.MetabaseMetadata{
			DataproductID:     dp.Dataproduct.ID,
			DatabaseID:        dbID,
			PermissionGroupID: dp.MetabaseGroupID,
			SAEmail:           dp.Email,
		})
		if err != nil {
			return err
		}

		// todo (erikvatt) find better solution for this, call metabase api instead of sleeping
		time.Sleep(2 * time.Second)

		if dp.MetabaseGroupID > 0 {
			err := m.client.RestrictAccessToDatabase(ctx, dp.MetabaseGroupID, dbID)
			if err != nil {
				return err
			}
		}

		if err := m.HideOtherTables(ctx, dbID, datasource.Table); err != nil {
			return err
		}

		if err := m.client.AutoMapSemanticTypes(ctx, dbID); err != nil {
			return err
		}

		m.log.Infof("Created Metabase database: %v", dp.Dataproduct.Name)
	}
	return nil
}

func (m *Metabase) delete(ctx context.Context, dataproducts map[string]bool, databases []Database) error {
	// Remove databases in Metabase that no longer exists or is not available to all users
	for _, mdb := range databases {
		if mdb.NadaID == "" || dataproducts[mdb.NadaID] {
			continue
		}

		if err := m.client.DeleteDatabase(ctx, mdb.ID); err != nil {
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

		mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, uid)
		if err != nil {
			m.log.WithError(err).Error("Get metabase metadata on delete database")
			m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
			continue
		}

		if mbMetadata.PermissionGroupID > 0 {
			if err := m.client.DeletePermissionGroup(ctx, mbMetadata.PermissionGroupID); err != nil {
				m.log.WithError(err).Error("Deleting permission group in Metabase")
				m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
				continue
			}

			if err := m.deleteServiceAccount(mbMetadata.SAEmail); err != nil {
				m.log.WithError(err).Error("Deleting metabase permission group service account")
				m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
				continue
			}
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

func (m *Metabase) syncPermissionGroupMembers(ctx context.Context, restrictedDps []*models.Dataproduct) error {
	for _, dp := range restrictedDps {
		subjects, err := m.repo.ListActiveAccessToDataproduct(ctx, dp.ID)
		if err != nil {
			return err
		}

		dpGrants := []string{}
		for _, s := range subjects {
			subject := strings.Split(s.Subject, ":")
			if subject[0] == "user" {
				dpGrants = append(dpGrants, subject[1])
			}
		}

		mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dp.ID)
		if err != nil {
			return err
		}

		mbGroupMembers, err := m.client.GetPermissionGroup(ctx, mbMetadata.PermissionGroupID)
		if err != nil {
			return err
		}

		err = m.removeDeletedMembersFromGroup(ctx, mbGroupMembers, dpGrants)
		if err != nil {
			return err
		}

		err = m.addNewMembersToGroup(ctx, mbMetadata.PermissionGroupID, mbGroupMembers, dpGrants)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Metabase) removeDeletedMembersFromGroup(ctx context.Context, groupMembers []PermissionGroupMember, dpGrants []string) error {
	for _, s := range groupMembers {
		if !contains(dpGrants, s.Email) {
			m.client.RemovePermissionGroupMember(ctx, s.ID)
		}
	}

	return nil
}

func (m *Metabase) addNewMembersToGroup(ctx context.Context, groupID int, groupMembers []PermissionGroupMember, dpGrants []string) error {
	for _, s := range dpGrants {
		if !groupContainsUser(groupMembers, s) {
			if err := m.client.AddPermissionGroupMember(ctx, groupID, s); err != nil {
				m.log.WithError(err).WithField("user", s).
					WithField("group", groupID).
					Warn("Unable to sync user")
			}
		}
	}

	return nil
}

func (m *Metabase) HideOtherTables(ctx context.Context, dbID int, table string) error {
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
	projectResource := os.Getenv("GCP_TEAM_PROJECT_ID")
	request := &iam.CreateServiceAccountRequest{
		AccountId: "nada-" + MarshalUUID(dp.ID),
		ServiceAccount: &iam.ServiceAccount{
			Description: "Metabase service account for dataproduct " + dp.ID.String(),
			DisplayName: dp.Name,
		},
	}

	account, err := m.iamService.Projects.ServiceAccounts.Create("projects/"+projectResource, request).Do()
	if err != nil {
		return nil, "", err
	}

	iamPolicyCall := m.crmService.Projects.GetIamPolicy(projectResource, &cloudresourcemanager.GetIamPolicyRequest{})
	iamPolicies, err := iamPolicyCall.Do()
	if err != nil {
		return nil, "", err
	}

	iamPolicies.Bindings = append(iamPolicies.Bindings, &cloudresourcemanager.Binding{
		Members: []string{"serviceAccount:" + account.Email},
		Role:    "projects/" + projectResource + "/roles/nada.metabase",
	})

	iamSetPolicyCall := m.crmService.Projects.SetIamPolicy(projectResource, &cloudresourcemanager.SetIamPolicyRequest{
		Policy: iamPolicies,
	})

	_, err = iamSetPolicyCall.Do()
	if err != nil {
		return nil, "", err
	}

	keyRequest := &iam.CreateServiceAccountKeyRequest{}

	key, err := m.iamService.Projects.ServiceAccounts.Keys.Create("projects/-/serviceAccounts/"+account.UniqueId, keyRequest).Do()
	if err != nil {
		return nil, "", err
	}

	saJson, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return nil, "", err
	}

	return saJson, account.Email, err
}

func (m *Metabase) deleteServiceAccount(saEmail string) error {
	_, err := m.iamService.Projects.ServiceAccounts.
		Delete("projects/" + os.Getenv("GCP_TEAM_PROJECT_ID") + "/serviceAccounts/" + saEmail).
		Do()
	return err
}

func groupContainsUser(mbGroups []PermissionGroupMember, email string) bool {
	for _, a := range mbGroups {
		if a.Email == email {
			return true
		}
	}
	return false
}

func contains(list []string, value string) bool {
	for _, g := range list {
		if value == g {
			return true
		}
	}
	return false
}

func MarshalUUID(id uuid.UUID) string {
	return strings.ToLower(base58.Encode(id[:]))
}
