package metabase

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"google.golang.org/api/cloudresourcemanager/v1"
	iam "google.golang.org/api/iam/v1"
)

func (m *Metabase) grantMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) {
	log := m.log.WithField("datasetID", dsID)

	if subject == "group:all-users@nav.no" {
		m.addAllUsersDataset(ctx, dsID)
		return
	}

	email, isGroup, err := parseSubject(subject)
	if err != nil {
		log.WithError(err).Errorf("parsing subject %v", subject)
		return
	}
	if isGroup {
		m.addGroupAccess(ctx, dsID, email)
	} else {
		m.addMetabaseGroupMember(ctx, dsID, email)
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

		ownerAADGroupID, err := m.client.CreatePermissionGroup(ctx, ds.Name+" (owner aad)")
		if err != nil {
			log.WithError(err).Error("create owner group")
			return
		}

		err = m.create(ctx, dsWrapper{
			Dataset:                 ds,
			Key:                     m.sa,
			Email:                   m.saEmail,
			MetabaseOwnerAADGroupID: ownerAADGroupID,
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
		if err := m.client.OpenAccessToDatabase(ctx, 0, mbMetadata.DatabaseID); err != nil {
			log.WithError(err).Error("open access to dataset")
			return
		}

		if err := m.client.DeletePermissionGroup(ctx, mbMetadata.PermissionGroupID); err != nil {
			log.WithError(err).Errorf("removing old permission group %v when opening database", mbMetadata.PermissionGroupID)
		}

		if err := m.repo.SetPermissionGroupMetabaseMetadata(ctx, mbMetadata.DatasetID, 0); err != nil {
			log.WithError(err).Error("setting permission group to all users")
			return
		}
	}
}

func (m *Metabase) addDatasetMapping(ctx context.Context, dsID uuid.UUID) {
	accesses, err := m.repo.ListActiveAccessToDataset(ctx, dsID)
	if err != nil {
		return
	}

	if containsAllUsers(accesses) {
		m.addAllUsersDataset(ctx, dsID)
		return
	}

	m.addRestrictedDatasetMapping(ctx, dsID)
}

func (m *Metabase) addRestrictedDatasetMapping(ctx context.Context, dsID uuid.UUID) {
	log := m.log.WithField("datasetID", dsID)

	mbMeta, err := m.repo.GetMetabaseMetadata(ctx, dsID, true)
	if errors.Is(err, sql.ErrNoRows) {
		ds, err := m.repo.GetDataset(ctx, dsID)
		if err != nil {
			log.WithError(err).Error("getting dataset")
			return
		}

		if err := m.createRestricted(ctx, ds); err != nil {
			log.WithError(err).Error("create restricted database")
			return
		}
	} else if err != nil {
		log.WithError(err).Error("get metabase metadata")
		return
	}

	if mbMeta != nil && mbMeta.PermissionGroupID == 0 {
		log.Error("not allowed to expose a previously open database as a restricted")
		return
	}

	if mbMeta != nil && mbMeta.DeletedAt != nil {
		log.Info("restoring db")
		if err := m.restore(ctx, dsID, mbMeta); err != nil {
			log.WithError(err).Error("restoring db")
		}
	}

	if err := m.grantAccessesOnCreation(ctx, dsID); err != nil {
		log.WithError(err).Error("granting accesses after database creation")
		return
	}
}

func (m *Metabase) grantAccessesOnCreation(ctx context.Context, dsID uuid.UUID) error {
	accesses, err := m.repo.ListActiveAccessToDataset(ctx, dsID)
	if err != nil {
		return err
	}

	for _, a := range accesses {
		email, isGroup, err := parseSubject(a.Subject)
		if err != nil {
			return err
		}
		if isGroup {
			m.addGroupAccess(ctx, dsID, email)
		} else {
			m.addMetabaseGroupMember(ctx, dsID, email)
		}
	}

	return nil
}

func (m *Metabase) addGroupAccess(ctx context.Context, dsID uuid.UUID, email string) {
	log := m.log.WithField("datasetID", dsID)

	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dsID, false)
	if err != nil {
		log.WithError(err).Error("getting metabase metadata")
		return
	}

	if mbMetadata.AADPermissionGroupID == 0 {
		log.WithError(err).Errorf("aad permission group does not exist for dataset %v", dsID)
		return
	}

	groupID, err := m.client.GetAzureGroupID(ctx, email)
	if err != nil {
		log.WithError(err).Error("getting azure group id")
		return
	}

	if err := m.client.UpdateGroupMapping(ctx, groupID, mbMetadata.AADPermissionGroupID, GroupMappingOperationAdd); err != nil {
		log.WithError(err).Errorf("unable to add metabase group mapping")
		return
	}
}

func (m *Metabase) addMetabaseGroupMember(ctx context.Context, dsID uuid.UUID, email string) {
	log := m.log.WithField("datasetID", dsID)
	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dsID, false)
	if err != nil {
		log.WithError(err).Error("getting metabase metadata")
		return
	}

	mbGroupMembers, err := m.client.GetPermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		log.WithError(err).Error("getting permission group")
		return
	}

	exists, _ := memberExists(mbGroupMembers, email)
	if exists {
		log.Info("member already exists")
		return
	}

	if err := m.client.AddPermissionGroupMember(ctx, mbMetadata.PermissionGroupID, email); err != nil {
		log.WithError(err).WithField("user", email).
			WithField("group", mbMetadata.PermissionGroupID).
			Warn("Unable to sync user")
	}
}

func (m *Metabase) createRestricted(ctx context.Context, ds *models.Dataset) error {
	groupID, err := m.client.CreatePermissionGroup(ctx, ds.Name)
	if err != nil {
		return err
	}

	aadGroupID, err := m.client.CreatePermissionGroup(ctx, ds.Name+" (aad)")
	if err != nil {
		return err
	}

	ownerAADGroupID, err := m.client.CreatePermissionGroup(ctx, ds.Name+" (owner aad)")
	if err != nil {
		return err
	}

	log.Printf("Owner AAD group %v for %v, created", ownerAADGroupID, ds.Name)

	// Hack/workaround necessary due to how metabase has implemented saml group sync. When removing a saml group mapping to a
	// metabase permission group, the users in the saml group will remain members of the permission group.
	// Adding a dummy mapping ensures that users are evicted from the permission group when an actual group mapping is removed.
	// See https://github.com/metabase/metabase/issues/26079
	if err := m.client.UpdateGroupMapping(ctx, "non-existant-aad-group", aadGroupID, GroupMappingOperationAdd); err != nil {
		return err
	}

	colID, err := m.client.CreateCollectionWithAccess(ctx, []int{groupID, aadGroupID}, ds.Name)
	if err != nil {
		return err
	}

	key, email, err := m.createServiceAccount(ds)
	if err != nil {
		return err
	}

	err = m.create(ctx, dsWrapper{
		Dataset:                 ds,
		Key:                     string(key),
		Email:                   email,
		MetabaseGroupID:         groupID,
		MetabaseAADGroupID:      aadGroupID,
		MetabaseOwnerAADGroupID: ownerAADGroupID,
		CollectionID:            colID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *Metabase) create(ctx context.Context, ds dsWrapper) error {
	log := m.log.WithField("Dataset", ds.Dataset.Name)
	log.Printf("Create metabase database for dataset %v", ds.Dataset.Name)

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
		DatasetID:            ds.Dataset.ID,
		DatabaseID:           dbID,
		PermissionGroupID:    ds.MetabaseGroupID,
		AADPermissionGroupID: ds.MetabaseAADGroupID,
		OwnerAADGroupID:      ds.MetabaseOwnerAADGroupID,
		CollectionID:         ds.CollectionID,
		SAEmail:              ds.Email,
	})
	if err != nil {
		return err
	}

	if err := m.waitForDatabase(ctx, dbID, datasource.Table); err != nil {
		return err
	}

	if ds.MetabaseOwnerAADGroupID > 0 {
		log.Printf("Config owner aad group %v", ds.MetabaseOwnerAADGroupID)

		groupID, err := m.getOwnerAADGroupID(ctx, ds.Dataset.DataproductID)
		if err != nil {
			log.WithError(err).Errorf("Failed to get aad group of dataproduct %v", ds.Dataset.DataproductID)
		} else {
			log.Printf("Get aad group %v", groupID)
			if err := m.client.UpdateGroupMapping(ctx, groupID, ds.MetabaseOwnerAADGroupID, GroupMappingOperationAdd); err != nil {
				log.WithError(err).Error("Failed to update group mapping")
			}
		}
	}

	if ds.MetabaseGroupID > 0 || ds.MetabaseAADGroupID > 0 {
		err := m.client.RestrictAccessToDatabase(ctx, []int{ds.MetabaseGroupID, ds.MetabaseAADGroupID},
			ds.MetabaseOwnerAADGroupID, dbID)
		if err != nil {
			return err
		}
	} else if ds.MetabaseOwnerAADGroupID > 0 {
		err := m.client.grantAADGroupOwnerPermission(ctx, ds.MetabaseOwnerAADGroupID, dbID)
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

func (m *Metabase) restore(ctx context.Context, datasetID uuid.UUID, mbMetadata *models.MetabaseMetadata) error {
	ds, err := m.repo.GetBigqueryDatasource(ctx, datasetID)
	if err != nil {
		return err
	}

	err = m.accessMgr.Grant(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMetadata.SAEmail)
	if err != nil {
		m.log.Error("Unable to restore access")
		return err
	}

	if err := m.repo.RestoreMetabaseMetadata(ctx, datasetID); err != nil {
		m.log.Error("Unable to soft create metabase metadata")
		return err
	}

	return nil
}

func (m *Metabase) waitForDatabase(ctx context.Context, dbID int, tableName string) error {
	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		tables, err := m.client.Tables(ctx, dbID)
		if err != nil || len(tables) == 0 {
			continue
		}
		for _, tab := range tables {
			if tab.Name == tableName && len(tab.Fields) > 0 {
				return nil
			}
		}
	}

	return fmt.Errorf("unable to create database %v", tableName)
}

func containsAllUsers(accesses []*models.Access) bool {
	for _, a := range accesses {
		if a.Subject == "group:all-users@nav.no" {
			return true
		}
	}

	return false
}

func (m *Metabase) getOwnerAADGroupID(ctx context.Context, dpID uuid.UUID) (string, error) {
	dp, err := m.repo.GetDataproduct(ctx, dpID)
	if err != nil {
		return "", err
	}
	if dp.Owner.AADGroup == nil {
		return "", fmt.Errorf("Owner aad group for dataproduct %v not found, cannot create owner permisiion group in metabase", dpID)
	}

	return m.client.GetAzureGroupID(ctx, *dp.Owner.AADGroup)
}
