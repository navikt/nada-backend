package metabase

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/service"
	"google.golang.org/api/cloudresourcemanager/v1"
	iam "google.golang.org/api/iam/v1"
)

func (m *Metabase) grantMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) {
	log := m.log.WithField("datasetID", dsID)

	if subject == "group:all-users@nav.no" {
		m.addAllUsersDataset(ctx, dsID)
		return
	}

	email, sType, err := parseSubject(subject)
	if err != nil {
		log.WithError(err).Errorf("parsing subject %v", subject)
		return
	}

	switch sType {
	case "user":
		m.addMetabaseGroupMember(ctx, dsID, email)
	default:
		log.Infof("unsupported subject type %v for metabase access grant", sType)
	}
}

func (m *Metabase) addAllUsersDataset(ctx context.Context, dsID uuid.UUID) {
	log := m.log.WithField("datasetID", dsID)

	mbMetadata, err := service.GetMetabaseMetadata(ctx, dsID, true)
	if errors.Is(err, sql.ErrNoRows) {
		ds, apierr := service.GetDataset(ctx, dsID.String())
		if apierr != nil {
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

		if err := m.client.DeletePermissionGroup(ctx, mbMetadata.PermissionGroupID); err != nil {
			log.WithError(err).Errorf("removing old permission group %v when opening database", mbMetadata.PermissionGroupID)
		}

		if err := service.SetPermissionGroupMetabaseMetadata(ctx, mbMetadata.DatasetID, 0); err != nil {
			log.WithError(err).Error("setting permission group to all users")
			return
		}
	}
}

func (m *Metabase) addDatasetMapping(ctx context.Context, dsID uuid.UUID) {
	accesses, err := service.ListActiveAccessToDataset(ctx, dsID)
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

	mbMeta, err := service.GetMetabaseMetadata(ctx, dsID, true)
	if errors.Is(err, sql.ErrNoRows) {
		ds, err := service.GetDataset(ctx, dsID.String())
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
	accesses, err := service.ListActiveAccessToDataset(ctx, dsID)
	if err != nil {
		return err
	}

	for _, a := range accesses {
		email, sType, err := parseSubject(a.Subject)
		if err != nil {
			m.log.WithError(err).Errorf("parsing subject %v", a.Subject)
			return err
		}

		switch sType {
		case "user":
			m.addMetabaseGroupMember(ctx, dsID, email)
		default:
			m.log.Infof("unsupported subject type %v for metabase access grant", sType)
		}
	}

	return nil
}

func (m *Metabase) addMetabaseGroupMember(ctx context.Context, dsID uuid.UUID, email string) {
	log := m.log.WithField("datasetID", dsID)
	mbMetadata, err := service.GetMetabaseMetadata(ctx, dsID, false)
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

func (m *Metabase) createRestricted(ctx context.Context, ds *service.Dataset) error {
	groupID, err := m.client.CreatePermissionGroup(ctx, ds.Name)
	if err != nil {
		return err
	}

	colID, err := m.client.CreateCollectionWithAccess(ctx, []int{groupID}, ds.Name)
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
	log := m.log.WithField("Dataset", ds.Dataset.Name)
	log.Printf("Create metabase database for dataset %v", ds.Dataset.Name)

	datasource, apierr := service.GetBigqueryDatasource(ctx, ds.Dataset.ID, false)
	if apierr != nil {
		return apierr
	}

	err := m.accessMgr.Grant(ctx, datasource.ProjectID, datasource.Dataset, datasource.Table, "serviceAccount:"+ds.Email)
	if err != nil {
		return err
	}

	dp, apierr := service.GetDataproduct(ctx, ds.Dataset.DataproductID.String())
	if apierr != nil {
		return apierr
	}

	dbID, err := m.client.CreateDatabase(ctx, dp.Owner.Group, ds.Dataset.Name, ds.Key, ds.Email, datasource)
	if err != nil {
		return err
	}

	if err := m.waitForDatabase(ctx, dbID, datasource.Table); err != nil {
		if err := m.cleanupOnCreateDatabaseError(ctx, dbID, ds); err != nil {
			m.log.WithError(err).Error("cleaning up on metabase database creation timeout")
			return err
		}
		return err
	}

	mbMeta := service.MetabaseMetadata{
		DatasetID:         ds.Dataset.ID,
		DatabaseID:        dbID,
		PermissionGroupID: ds.MetabaseGroupID,
		CollectionID:      ds.CollectionID,
		SAEmail:           ds.Email,
	}
	err = service.CreateMetabaseMetadata(ctx, mbMeta)
	if err != nil {
		return err
	}

	if ds.MetabaseGroupID > 0 {
		err := m.client.RestrictAccessToDatabase(ctx, ds.MetabaseGroupID, dbID)
		if err != nil {
			return err
		}
	} else {
		err := m.client.OpenAccessToDatabase(ctx, dbID)
		if err != nil {
			return err
		}
	}

	if err := m.SyncTableVisibility(ctx, &mbMeta, *datasource); err != nil {
		return err
	}

	if err := m.client.AutoMapSemanticTypes(ctx, dbID); err != nil {
		return err
	}

	m.log.Infof("Created Metabase database: %v", ds.Dataset.Name)
	return nil
}

func (m *Metabase) createServiceAccount(ds *service.Dataset) ([]byte, string, error) {
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

func (m *Metabase) restore(ctx context.Context, datasetID uuid.UUID, mbMetadata *service.MetabaseMetadata) error {
	ds, apierr := service.GetBigqueryDatasource(ctx, datasetID, false)
	if apierr != nil {
		return apierr
	}

	err := m.accessMgr.Grant(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMetadata.SAEmail)
	if err != nil {
		m.log.Error("Unable to restore access")
		return err
	}

	if err := service.RestoreMetabaseMetadata(ctx, datasetID); err != nil {
		m.log.Error("Unable to soft create metabase metadata")
		return err
	}

	return nil
}

func (m *Metabase) waitForDatabase(ctx context.Context, dbID int, tableName string) error {
	for i := 0; i < 200; i++ {
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

func (m *Metabase) cleanupOnCreateDatabaseError(ctx context.Context, dbID int, ds dsWrapper) error {
	dataset, err := service.GetDataset(ctx, ds.Dataset.ID.String())
	if err != nil {
		return err
	}
	services := dataset.Mappings

	for idx, msvc := range services {
		if msvc == service.MappingServiceMetabase {
			services = append(services[:idx], services[idx+1:]...)
		}
	}

	if err := m.client.deleteDatabase(ctx, dbID); err != nil {
		return err
	}

	if ds.CollectionID != 0 {
		if err := m.client.DeletePermissionGroup(ctx, ds.MetabaseGroupID); err != nil {
			return err
		}

		if err := m.client.ArchiveCollection(ctx, ds.CollectionID); err != nil {
			return err
		}

		if err := m.deleteServiceAccount(ds.Email); err != nil {
			return err
		}
	}

	_, apierr := service.MapDataset(ctx, ds.Dataset.ID.String(), services)
	return apierr
}

func containsAllUsers(accesses []*service.Access) bool {
	for _, a := range accesses {
		if a.Subject == "group:all-users@nav.no" {
			return true
		}
	}

	return false
}
