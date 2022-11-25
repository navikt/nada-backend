package metabase

import (
	"context"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"google.golang.org/api/googleapi"
)

func (m *Metabase) revokeMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) {
	if subject == "group:all-users@nav.no" {
		m.softDeleteDatabase(ctx, dsID)
	} else if strings.HasPrefix(subject, "group:") {
		m.removeGroupAccess(ctx, dsID, subject)
	} else {
		m.removeMetabaseGroupMember(ctx, dsID, subject)
	}
}

func (m *Metabase) deleteDatabase(ctx context.Context, dsID uuid.UUID) {
	mbMeta, err := m.repo.GetMetabaseMetadata(ctx, dsID, true)
	if err != nil {
		m.log.WithError(err).Error("getting metabase metadata")
	}

	if isRestrictedDatabase(mbMeta) {
		m.deleteRestrictedDatabase(ctx, dsID)
		return
	}

	m.deleteAllUsersDatabase(ctx, dsID)
}

func (m *Metabase) revokeAccessesOnSoftDelete(ctx context.Context, dsID uuid.UUID) error {
	accesses, err := m.repo.ListActiveAccessToDataset(ctx, dsID)
	if err != nil {
		return err
	}

	for _, a := range accesses {
		if strings.HasPrefix(a.Subject, "group:") {
			m.removeGroupAccess(ctx, dsID, a.Subject)
		} else {
			m.removeMetabaseGroupMember(ctx, dsID, a.Subject)
		}
	}

	return nil
}

func (m *Metabase) removeGroupAccess(ctx context.Context, dsID uuid.UUID, subject string) {
	log := m.log.WithField("datasetID", dsID)

	mbMetadata, err := m.repo.GetMetabaseMetadata(ctx, dsID, false)
	if err != nil {
		log.WithError(err).Error("getting metabase metadata")
		return
	}

	if mbMetadata.AADPermissionGroupID == 0 {
		log.WithError(err).Errorf("permission group does not exist for dataset %v", dsID)
		return
	}

	s := strings.Split(subject, ":")
	if len(s) != 2 {
		log.WithError(err).Errorf("invalid subject format, should be type:email")
		return
	}

	if s[0] != "group" {
		log.Info("subject is not a group")
		return
	}

	groupID, err := m.client.GetAzureGroupID(ctx, s[1])
	if err != nil {
		log.WithError(err).Error("getting azure group id")
		return
	}

	if err := m.client.UpdateGroupMapping(ctx, groupID, mbMetadata.AADPermissionGroupID, GroupMappingOperationRemove); err != nil {
		log.WithError(err).Errorf("unable to remove metabase group mapping")
		return
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

func (m *Metabase) softDeleteDatabase(ctx context.Context, datasetID uuid.UUID) error {
	log := m.log.WithField("datasetID", datasetID)
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
		log.Error("Unable to revoke access")
		return err
	}

	if err := m.repo.SoftDeleteMetabaseMetadata(ctx, datasetID); err != nil {
		log.Error("Unable to soft delete metabase metadata")
		return err
	}

	log.Infof("Soft deleted Metabase database: %v", mbMeta.DatabaseID)
	return nil
}

func (m *Metabase) deleteAllUsersDatabase(ctx context.Context, datasetID uuid.UUID) {
	log := m.log.WithField("datasetID", datasetID)

	mbMeta, err := m.repo.GetMetabaseMetadata(ctx, datasetID, false)
	if err != nil {
		log.Error("Get metabase metadata")
		return
	}

	if err := m.client.deleteDatabase(ctx, mbMeta.DatabaseID); err != nil {
		log.Errorf("Unable to delete all-users database %v", mbMeta.DatabaseID)
		return
	}

	if err := m.repo.DeleteMetabaseMetadata(ctx, mbMeta.DatasetID); err != nil {
		log.Errorf("Unable to delete all-users metabase metadata for database %v", mbMeta.DatabaseID)
	}

	log.Info("Deleted all-users database")
}

func (m *Metabase) deleteRestrictedDatabase(ctx context.Context, datasetID uuid.UUID) {
	log := m.log.WithField("datasetID", datasetID)
	mbMeta, err := m.repo.GetMetabaseMetadata(ctx, datasetID, false)
	if err != nil {
		log.Error("Get metabase metadata")
		return
	}

	ds, err := m.repo.GetBigqueryDatasource(ctx, datasetID)
	if err != nil {
		log.Error("Get bigquery datasource")
		return
	}

	err = m.accessMgr.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMeta.SAEmail)
	if err != nil {
		log.Error("Unable to revoke access")
		return
	}

	if err := m.deleteServiceAccount(mbMeta.SAEmail); err != nil {
		log.Errorf("Unable to delete service account for restricted database %v", mbMeta.DatabaseID)
		return
	}

	if err := m.client.DeletePermissionGroup(ctx, mbMeta.PermissionGroupID); err != nil {
		log.Errorf("Unable to delete permission group %v", mbMeta.PermissionGroupID)
		return
	}

	if err := m.client.DeletePermissionGroup(ctx, mbMeta.AADPermissionGroupID); err != nil {
		log.Errorf("Unable to delete AAD permission group %v", mbMeta.AADPermissionGroupID)
		return
	}

	if err := m.client.ArchiveCollection(ctx, mbMeta.CollectionID); err != nil {
		log.Errorf("Unable to archive collection %v", mbMeta.CollectionID)
		return
	}

	if err := m.client.deleteDatabase(ctx, mbMeta.DatabaseID); err != nil {
		log.Errorf("Unable to delete restricted database %v", mbMeta.DatabaseID)
		return
	}

	if err := m.repo.DeleteRestrictedMetabaseMetadata(ctx, datasetID); err != nil {
		log.Error("Unable to delete metabase metadata")
		return
	}

	log.Infof("Deleted restricted Metabase database: %v", mbMeta.DatabaseID)
}

func (m *Metabase) deleteServiceAccount(saEmail string) error {
	_, err := m.iamService.Projects.ServiceAccounts.
		Delete("projects/" + os.Getenv("GCP_TEAM_PROJECT_ID") + "/serviceAccounts/" + saEmail).
		Do()
	if err != nil {
		apiError, ok := err.(*googleapi.Error)
		if ok {
			if apiError.Code == 404 {
				m.log.Infof("delete iam service account: service account %v does not exist", saEmail)
				return nil
			}
		}
		return err
	}
	return nil
}

func isRestrictedDatabase(mbMeta *models.MetabaseMetadata) bool {
	return mbMeta.CollectionID != 0
}
