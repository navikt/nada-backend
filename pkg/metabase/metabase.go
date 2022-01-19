package metabase

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/event"
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
	events     *event.Manager
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
	CollectionID    int
}

func New(repo *database.Repo, client *Client, accessMgr graph.AccessManager, eventMgr *event.Manager, serviceAccount, serviceAccountEmail string, errs *prometheus.CounterVec, iamService *iam.Service, crmService *cloudresourcemanager.Service, log *logrus.Entry) *Metabase {
	return &Metabase{
		repo:       repo,
		client:     client,
		accessMgr:  accessMgr,
		events:     eventMgr,
		sa:         serviceAccount,
		saEmail:    serviceAccountEmail,
		errs:       errs,
		iamService: iamService,
		crmService: crmService,
		log:        log,
	}
}

func (m *Metabase) Run(ctx context.Context, frequency time.Duration) {
	m.events.ListenForDataproductGrant(m.grantMetabaseAccess)
	m.events.ListenForDataproductRevoke(m.revokeMetabaseAccess)
	m.events.ListenForDataproductAddMetabaseMapping(m.addDataproductMapping)
	m.events.ListenForDataproductRemoveMetabaseMapping(m.removeDataproductMapping)

	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Metabase) grantMetabaseAccess(ctx context.Context, dpID uuid.UUID, subject string) {
	if subject == "group:all-users@nav.no" {
		m.addAllUsersDataproduct(ctx, dpID)
	} else {
		m.addMetabaseGroupMember(ctx, dpID, subject)
	}
}

func (m *Metabase) revokeMetabaseAccess(ctx context.Context, dpID uuid.UUID, subject string) {
	if subject == "group:all-users@nav.no" {
		m.deleteAllUsersDataproduct(ctx, dpID)
	} else {
		m.removeMetabaseGroupMember(ctx, dpID, subject)
	}
}

func (m *Metabase) addAllUsersDataproduct(ctx context.Context, dpID uuid.UUID) {
	log := m.log.WithField("dataproductID", dpID)
	databases, err := m.client.Databases(ctx)
	if err != nil {
		log.WithError(err).Error("get all databases")
		return
	}

	if dbExists(databases, dpID.String()) {
		mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dpID)
		if err != nil {
			log.WithError(err).Error("get metabase metadata")
			return
		}

		if mbMetadata.PermissionGroupID == 0 {
			// all-users access exists
			log.Info("all users database already exists in metabase")
			return
		}

		// todo(erikvatt) handle differently when restricted db exists in metabase?
		if err := m.delete(ctx, dpID.String(), databases); err != nil {
			log.WithError(err).Error("delete all users dataproduct")
			return
		}
	}

	dp, err := m.repo.GetDataproduct(ctx, dpID)
	if err != nil {
		log.WithError(err).Error("getting dataproduct")
		return
	}

	err = m.create(ctx, dpWrapper{
		Dataproduct: dp,
		Key:         m.sa,
		Email:       m.saEmail,
	})
	if err != nil {
		log.WithError(err).Error("creating metabase database")
		return
	}
}

func (m *Metabase) deleteAllUsersDataproduct(ctx context.Context, dpID uuid.UUID) {
	log := m.log.WithField("dataproductID", dpID)
	databases, err := m.client.Databases(ctx)
	if err != nil {
		log.WithError(err).Error("get all databases")
		return
	}

	if err := m.delete(ctx, dpID.String(), databases); err != nil {
		log.WithError(err).Error("delete all users database")
		return
	}
}

func (m *Metabase) addDataproductMapping(ctx context.Context, dpID uuid.UUID) {
	log := m.log.WithField("dataproductID", dpID)
	databases, err := m.client.Databases(ctx)
	if err != nil {
		log.WithError(err).Error("get all databases")
		return
	}

	if dbExists(databases, dpID.String()) {
		log.Info("database already exists in metabase")
		return
	}

	dp, err := m.repo.GetDataproduct(ctx, dpID)
	if err != nil {
		log.WithError(err).Error("getting dataproduct")
		return
	}

	if err := m.createRestricted(ctx, dp); err != nil {
		log.WithError(err).Error("create restricted database")
		return
	}
}

func (m *Metabase) removeDataproductMapping(ctx context.Context, dpID uuid.UUID) {
	log := m.log.WithField("dataproductID", dpID)
	databases, err := m.client.Databases(ctx)
	if err != nil {
		log.WithError(err).Error("getting all databases")
		return
	}

	// todo(erikvatt) handle if all-users access exists when removing metabase mapping
	if err := m.delete(ctx, dpID.String(), databases); err != nil {
		log.WithError(err).Error("delete restricted database")
		return
	}
}

func (m *Metabase) addMetabaseGroupMember(ctx context.Context, dpID uuid.UUID, subject string) {
	log := m.log.WithField("dataproductID", dpID)
	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dpID)
	if err != nil {
		log.WithError(err).Error("getting metabase metadata")
		return
	}

	s := strings.Split(subject, ":")
	if s[0] != "user" {
		log.Info("subject is not a user")
		return
	}

	mbGroupMembers, err := m.client.GetPermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		log.WithError(err).Error("getting permission group")
		return
	}

	exists, _ := memberExists(mbGroupMembers, s[1])
	if exists {
		log.Info("member already exists")
		return
	}

	if err := m.client.AddPermissionGroupMember(ctx, mbMetadata.PermissionGroupID, s[1]); err != nil {
		log.WithError(err).WithField("user", s).
			WithField("group", mbMetadata.PermissionGroupID).
			Warn("Unable to sync user")
	}
}

func (m *Metabase) removeMetabaseGroupMember(ctx context.Context, dpID uuid.UUID, subject string) {
	log := m.log.WithField("dataproductID", dpID)
	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dpID)
	if err != nil {
		log.WithError(err).Error("getting metabase metadata")
		return
	}

	s := strings.Split(subject, ":")
	if s[0] != "user" {
		log.Info("subject is not a user")
		return
	}

	mbGroupMembers, err := m.client.GetPermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		log.WithError(err).Error("getting permission group")
		return
	}

	exists, memberID := memberExists(mbGroupMembers, s[1])
	if !exists {
		log.Info("member does not exist")
		return
	}

	if err := m.client.RemovePermissionGroupMember(ctx, memberID); err != nil {
		log.WithError(err).Error("removing permission group member")
		return
	}
}

func memberExists(groupMembers []PermissionGroupMember, subject string) (bool, int) {
	for _, m := range groupMembers {
		if m.Email == subject {
			return true, m.ID
		}
	}
	return false, -1
}

func dbExists(databases []Database, dpID string) bool {
	for _, d := range databases {
		if d.NadaID == dpID {
			return true
		}
	}
	return false
}

func (m *Metabase) createRestricted(ctx context.Context, dp *models.Dataproduct) error {
	groupID, err := m.client.CreatePermissionGroup(ctx, dp.Name)
	if err != nil {
		return err
	}

	colID, err := m.client.CreateCollectionWithAccess(ctx, groupID, dp.Name)
	if err != nil {
		return err
	}

	key, email, err := m.createServiceAccount(dp)
	if err != nil {
		return err
	}

	err = m.create(ctx, dpWrapper{
		Dataproduct:     dp,
		Key:             string(key),
		Email:           email,
		MetabaseGroupID: groupID,
		CollectionID:    colID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *Metabase) create(ctx context.Context, dp dpWrapper) error {
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
		CollectionID:      dp.CollectionID,
		SAEmail:           dp.Email,
	})
	if err != nil {
		return err
	}

	m.waitForDatabase(ctx, dbID, datasource.Table)

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
	return nil
}

func (m *Metabase) delete(ctx context.Context, dataproductID string, databases []Database) error {
	var mdb Database
	for _, db := range databases {
		if db.NadaID == dataproductID {
			mdb = db
		}
	}

	if err := m.client.DeleteDatabase(ctx, mdb.ID); err != nil {
		m.log.WithError(err).Error("Deleting database in Metabase")
		m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
		return err
	}

	uid, err := uuid.Parse(dataproductID)
	if err != nil {
		m.log.WithError(err).Error("Parsing UUID")
		m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
		return err
	}

	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, uid)
	if err != nil {
		m.log.WithError(err).Error("Get metabase metadata on delete database")
		m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
		return err
	}

	if mbMetadata.PermissionGroupID > 0 {
		if err := m.client.DeletePermissionGroup(ctx, mbMetadata.PermissionGroupID); err != nil {
			m.log.WithError(err).Error("Deleting permission group in Metabase")
			m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
			return err
		}

		if err := m.client.ArchiveCollection(ctx, mbMetadata.CollectionID); err != nil {
			m.log.WithError(err).Error("Archiving collection in Metabase")
			m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
			return err
		}

		if err := m.deleteServiceAccount(mbMetadata.SAEmail); err != nil {
			m.log.WithError(err).Error("Deleting metabase permission group service account")
			m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
			return err
		}
	}

	ds, err := m.repo.GetBigqueryDatasource(ctx, uid)
	if err != nil {
		m.log.WithError(err).Error("Getting Bigquery datasource")
		m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
		return err
	}
	if err := m.accessMgr.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mdb.SAEmail); err != nil {
		m.log.WithError(err).Error("Revoking IAM access")
		m.errs.WithLabelValues("RemoveMetabaseDatabase").Inc()
		return err
	}

	if err := m.repo.DeleteMetabaseMetadata(ctx, uid); err != nil {
		return err
	}

	m.log.Infof("Deleted Metabase database with ID: %v", mdb.ID)

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

func (m *Metabase) waitForDatabase(ctx context.Context, dbID int, tableName string) {
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		tables, err := m.client.Tables(ctx, dbID)
		if err != nil || len(tables) == 0 {
			continue
		}
		for _, tab := range tables {
			if tab.Name == tableName && len(tab.Fields) > 0 {
				return
			}
		}
	}
}

func MarshalUUID(id uuid.UUID) string {
	return strings.ToLower(base58.Encode(id[:]))
}
