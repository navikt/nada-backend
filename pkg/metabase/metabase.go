package metabase

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
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

type dsWrapper struct {
	Dataset         *models.Dataset
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
	m.events.ListenForDatasetGrant(m.grantMetabaseAccess)
	m.events.ListenForDatasetRevoke(m.revokeMetabaseAccess)
	m.events.ListenForDatasetAddMetabaseMapping(m.addDatasetMapping)
	m.events.ListenForDatasetRemoveMetabaseMapping(m.removeDatasetMapping)
	m.events.ListenForDatasetDelete(m.removeDatasetMapping)

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

func (m *Metabase) grantMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) {
	if subject == "group:all-users@nav.no" {
		m.addAllUsersDataset(ctx, dsID)
	} else if strings.HasPrefix(subject, "group:") {
		m.addGroupAccess(ctx, dsID, subject)
	} else {
		m.addMetabaseGroupMember(ctx, dsID, subject)
	}
}

func (m *Metabase) revokeMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) {
	if subject == "group:all-users@nav.no" {
		m.deleteAllUsersDataset(ctx, dsID)
	} else if strings.HasPrefix(subject, "group:") {
		m.removeGroupAccess(ctx, dsID, subject)
	} else {
		m.removeMetabaseGroupMember(ctx, dsID, subject)
	}
}

func (m *Metabase) addAllUsersDataset(ctx context.Context, dsID uuid.UUID) {
	log := m.log.WithField("datasetID", dsID)

	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dsID, true)
	if errors.Is(err, sql.ErrNoRows) {
		ds, err := m.repo.GetDataset(ctx, dsID)
		if err != nil {
			log.WithError(err).Error("getting dataset")
			return
		}

		err = m.create(ctx, dsWrapper{
			Dataset: ds,
			Key:     m.sa,
			Email:   m.saEmail,
		})
		if err != nil {
			log.WithError(err).Error("creating metabase database")
			return
		}
		return
	} else if err != nil {
		log.WithError(err).Error("get metabase metadata")
		return
	}

	if mbMetadata.DeletedAt != nil {
		log.Info("restoring db")
		if err := m.restore(ctx, dsID, mbMetadata); err != nil {
			log.WithError(err).Error("restoring db")
		}
		return
	}
	if mbMetadata.PermissionGroupID == 0 {
		// all-users access exists
		log.Info("all users database already exists in metabase")
		return
	} else {
		if err := m.client.OpenAccessToDatabase(ctx, mbMetadata.DatabaseID); err != nil {
			log.WithError(err).Error("open access to dataset")
			return
		}

		if err := m.repo.SetPermissionGroupMetabaseMetadata(ctx, mbMetadata.DatasetID, 0); err != nil {
			log.WithError(err).Error("setting permission group to all users")
			return
		}
	}
}

func (m *Metabase) deleteAllUsersDataset(ctx context.Context, dsID uuid.UUID) {
	log := m.log.WithField("datasetID", dsID)

	if err := m.softDelete(ctx, dsID); err != nil {
		log.WithError(err).Error("softDelete all users database")
		return
	}
}

func (m *Metabase) addDatasetMapping(ctx context.Context, dsID uuid.UUID) {
	log := m.log.WithField("datasetID", dsID)

	mbMeta, err := m.repo.GetMetabaseMetadata(ctx, dsID, true)
	if errors.Is(err, sql.ErrNoRows) {
		ds, err := m.repo.GetDataset(ctx, dsID)
		if err != nil {
			log.WithError(err).Error("getting dataproduct")
			return
		}

		if err := m.createRestricted(ctx, ds); err != nil {
			log.WithError(err).Error("create restricted database")
			return
		}
		return
	} else if err != nil {
		log.WithError(err).Error("get metabase metadata")
		return
	}

	if mbMeta.PermissionGroupID == 0 {
		log.Error("not allowed to expose a previously open database as a restricted")
		return
	}

	log.Info("database already exists in metabase")
	if mbMeta.DeletedAt != nil {
		log.Info("restoring db")
		if err := m.restore(ctx, dsID, mbMeta); err != nil {
			log.WithError(err).Error("restoring db")
		}
		return
	}
}

func (m *Metabase) removeDatasetMapping(ctx context.Context, dsID uuid.UUID) {
	log := m.log.WithField("datasetID", dsID)

	if err := m.softDelete(ctx, dsID); err != nil {
		log.WithError(err).Error("delete restricted database")
		return
	}
}

func (m *Metabase) addGroupAccess(ctx context.Context, dsID uuid.UUID, subject string) {
	log := m.log.WithField("datasetID", dsID)

	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dsID, false)
	if err != nil {
		log.WithError(err).Error("getting metabase metadata")
		return
	}

	if mbMetadata.PermissionGroupID == 0 {
		log.WithError(err).Errorf("permission group does not exist for dataset %v", dsID)
		return
	}

	s := strings.Split(subject, ":")
	if s[0] != "group" {
		log.Info("subject is not a group")
		return
	}

	groupID, err := m.client.GetAzureGroupID(ctx, s[1])
	if err != nil {
		log.WithError(err).Error("getting azure group id")
		return
	}

	if err := m.client.UpdateGroupMapping(ctx, groupID, mbMetadata.PermissionGroupID, GroupMappingOperationAdd); err != nil {
		log.WithError(err).Errorf("unable to add metabase group mapping")
		return
	}
}

func (m *Metabase) removeGroupAccess(ctx context.Context, dsID uuid.UUID, subject string) {
	log := m.log.WithField("datasetID", dsID)

	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dsID, false)
	if err != nil {
		log.WithError(err).Error("getting metabase metadata")
		return
	}

	if mbMetadata.PermissionGroupID == 0 {
		log.WithError(err).Errorf("permission group does not exist for dataset %v", dsID)
		return
	}

	s := strings.Split(subject, ":")
	if s[0] != "group" {
		log.Info("subject is not a group")
		return
	}

	groupID, err := m.client.GetAzureGroupID(ctx, s[1])
	if err != nil {
		log.WithError(err).Error("getting azure group id")
		return
	}

	if err := m.client.UpdateGroupMapping(ctx, groupID, mbMetadata.PermissionGroupID, GroupMappingOperationRemove); err != nil {
		log.WithError(err).Errorf("unable to add metabase group mapping")
		return
	}
}

func (m *Metabase) addMetabaseGroupMember(ctx context.Context, dsID uuid.UUID, subject string) {
	log := m.log.WithField("datasetID", dsID)
	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dsID, false)
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

func (m *Metabase) removeMetabaseGroupMember(ctx context.Context, dsID uuid.UUID, subject string) {
	log := m.log.WithField("datasetID", dsID)
	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dsID, false)
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

func (m *Metabase) createRestricted(ctx context.Context, ds *models.Dataset) error {
	groupID, err := m.client.CreatePermissionGroup(ctx, ds.Name)
	if err != nil {
		return err
	}

	colID, err := m.client.CreateCollectionWithAccess(ctx, groupID, ds.Name)
	if err != nil {
		return err
	}

	key, email, err := m.createServiceAccount(ds)
	if err != nil {
		return err
	}

	err = m.create(ctx, dsWrapper{
		Dataset:         ds,
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

func (m *Metabase) create(ctx context.Context, ds dsWrapper) error {
	datasource, err := m.repo.GetBigqueryDatasource(ctx, ds.Dataset.ID)
	if err != nil {
		return err
	}

	err = m.accessMgr.Grant(ctx, datasource.ProjectID, datasource.Dataset, datasource.Table, "serviceAccount:"+ds.Email)
	if err != nil {
		return err
	}

	dp, err := m.repo.GetDataproduct(ctx, ds.Dataset.DataproductID)
	if err != nil {
		return err
	}

	dbID, err := m.client.CreateDatabase(ctx, dp.Owner.Group, ds.Dataset.Name, ds.Key, ds.Email, &datasource)
	if err != nil {
		return err
	}

	err = m.repo.CreateMetabaseMetadata(ctx, models.MetabaseMetadata{
		DatasetID:         ds.Dataset.ID,
		DatabaseID:        dbID,
		PermissionGroupID: ds.MetabaseGroupID,
		CollectionID:      ds.CollectionID,
		SAEmail:           ds.Email,
	})
	if err != nil {
		return err
	}

	m.waitForDatabase(ctx, dbID, datasource.Table)

	if ds.MetabaseGroupID > 0 {
		err := m.client.RestrictAccessToDatabase(ctx, ds.MetabaseGroupID, dbID)
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

	m.log.Infof("Created Metabase database: %v", ds.Dataset.Name)
	return nil
}

func (m *Metabase) restore(ctx context.Context, datasetID uuid.UUID, mbMetadata *models.MetabaseMetadata) error {
	ds, err := m.repo.GetBigqueryDatasource(ctx, datasetID)
	if err != nil {
		return err
	}

	err = m.accessMgr.Grant(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMetadata.SAEmail)
	if err != nil {
		m.log.Error("Unable to revoke access")
		return err
	}

	if err := m.repo.RestoreMetabaseMetadata(ctx, datasetID); err != nil {
		m.log.Error("Unable to soft create metabase metadata")
		return err
	}

	return nil
}

func (m *Metabase) softDelete(ctx context.Context, datasetID uuid.UUID) error {
	mbMeta, er := m.repo.GetMetabaseMetadata(ctx, datasetID, false)
	if er != nil {
		return er
	}

	ds, err := m.repo.GetBigqueryDatasource(ctx, datasetID)
	if err != nil {
		return err
	}

	err = m.accessMgr.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMeta.SAEmail)
	if err != nil {
		m.log.Error("Unable to revoke access")
		return err
	}

	if err := m.repo.SoftDeleteMetabaseMetadata(ctx, datasetID); err != nil {
		m.log.Error("Unable to soft delete metabase metadata")
		return err
	}

	m.log.Infof("Soft deleted Metabase database: %v", mbMeta.DatabaseID)
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

func (m *Metabase) createServiceAccount(ds *models.Dataset) ([]byte, string, error) {
	projectResource := os.Getenv("GCP_TEAM_PROJECT_ID")
	request := &iam.CreateServiceAccountRequest{
		AccountId: "nada-" + MarshalUUID(ds.ID),
		ServiceAccount: &iam.ServiceAccount{
			Description: "Metabase service account for dataset " + ds.ID.String(),
			DisplayName: ds.Name,
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
