package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/rs/zerolog"

	"github.com/btcsuite/btcutil/base58"

	"github.com/gosimple/slug"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.MetabaseService = &metabaseService{}

type metabaseService struct {
	gcpProject          string
	serviceAccount      string
	serviceAccountEmail string
	groupAllUsers       string

	metabaseAPI       service.MetabaseAPI
	bigqueryAPI       service.BigQueryAPI
	serviceAccountAPI service.ServiceAccountAPI

	thirdPartyMappingStorage service.ThirdPartyMappingStorage
	metabaseStorage          service.MetabaseStorage
	bigqueryStorage          service.BigQueryStorage
	dataproductStorage       service.DataProductsStorage
	accessStorage            service.AccessStorage

	log zerolog.Logger
}

func (s *metabaseService) CreateMappingRequest(ctx context.Context, user *service.User, datasetID uuid.UUID, services []string) error {
	const op errs.Op = "metabaseService.CreateMappingRequest"

	ds, err := s.dataproductStorage.GetDataset(ctx, datasetID)
	if err != nil {
		return errs.E(op, err)
	}

	dp, err := s.dataproductStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return errs.E(op, err)
	}

	if err := ensureUserInGroup(user, dp.Owner.Group); err != nil {
		return errs.E(op, err)
	}

	err = s.thirdPartyMappingStorage.MapDataset(ctx, datasetID, services)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) MapDataset(ctx context.Context, datasetID uuid.UUID, services []string) error {
	const op errs.Op = "metabaseService.MapDataset"

	mapMetabase := false
	for _, svc := range services {
		if svc == service.MappingServiceMetabase {
			mapMetabase = true

			err := s.addDatasetMapping(ctx, datasetID)
			if err != nil {
				return errs.E(op, err)
			}

			break
		}
	}

	if !mapMetabase {
		err := s.DeleteDatabase(ctx, datasetID)
		if err != nil {
			return errs.E(op, err)
		}
	}

	return nil
}

func (s *metabaseService) addDatasetMapping(ctx context.Context, dsID uuid.UUID) error {
	const op errs.Op = "metabaseService.addDatasetMapping"

	accesses, err := s.accessStorage.ListActiveAccessToDataset(ctx, dsID)
	if err != nil {
		return errs.E(op, err)
	}

	if s.containsAllUsers(accesses) {
		err := s.addAllUsersDataset(ctx, dsID)
		if err != nil {
			return errs.E(op, err)
		}

		return nil
	}

	err = s.addRestrictedDatasetMapping(ctx, dsID)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) containsAllUsers(accesses []*service.Access) bool {
	for _, a := range accesses {
		if a.Subject == s.groupAllUsers {
			return true
		}
	}

	return false
}

func (s *metabaseService) addRestrictedDatasetMapping(ctx context.Context, dsID uuid.UUID) error {
	const op errs.Op = "metabaseService.addRestrictedDatasetMapping"

	mbMeta, err := s.metabaseStorage.GetMetadata(ctx, dsID, true)
	if errs.KindIs(errs.NotExist, err) {
		ds, err := s.dataproductStorage.GetDataset(ctx, dsID)
		if err != nil {
			return errs.E(op, err)
		}

		if err := s.createRestricted(ctx, ds); err != nil {
			return errs.E(op, err)
		}
	} else if err != nil {
		return errs.E(op, err)
	}

	if mbMeta != nil && mbMeta.PermissionGroupID == 0 {
		return errs.E(errs.InvalidRequest, op, fmt.Errorf("not allowed to expose a previously open database as a restricted"))
	}

	if mbMeta != nil && mbMeta.DeletedAt != nil {
		if err := s.restore(ctx, dsID, mbMeta); err != nil {
			return errs.E(op, err)
		}
	}

	if err := s.grantAccessesOnCreation(ctx, dsID); err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) grantAccessesOnCreation(ctx context.Context, dsID uuid.UUID) error {
	const op errs.Op = "metabaseService.grantAccessesOnCreation"

	accesses, err := s.accessStorage.ListActiveAccessToDataset(ctx, dsID)
	if err != nil {
		return errs.E(op, err)
	}

	for _, a := range accesses {
		email, sType, err := parseSubject(a.Subject)
		if err != nil {
			return errs.E(op, err)
		}

		switch sType {
		case "user":
			err := s.addMetabaseGroupMember(ctx, dsID, email)
			if err != nil {
				return errs.E(op, err)
			}
		default:
			s.log.Info().Msgf("Unsupported subject type %v for metabase access grant", sType)
		}
	}

	return nil
}

func (s *metabaseService) addMetabaseGroupMember(ctx context.Context, dsID uuid.UUID, email string) error {
	const op errs.Op = "metabaseService.addMetabaseGroupMember"

	mbMetadata, err := s.metabaseStorage.GetMetadata(ctx, dsID, false)
	if err != nil {
		// If we don't have metadata for the dataset, it means that the dataset is not mapped to Metabase
		// so no need to add the user to the group
		if errs.KindIs(errs.NotExist, err) {
			return nil
		}

		return errs.E(op, err)
	}

	mbGroupMembers, err := s.metabaseAPI.GetPermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		return errs.E(op, err)
	}

	exists, _ := memberExists(mbGroupMembers, email)
	if exists {
		return nil
	}

	if err := s.metabaseAPI.AddPermissionGroupMember(ctx, mbMetadata.PermissionGroupID, email); err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) restore(ctx context.Context, datasetID uuid.UUID, mbMetadata *service.MetabaseMetadata) error {
	const op errs.Op = "metabaseService.restore"

	ds, err := s.bigqueryStorage.GetBigqueryDatasource(ctx, datasetID, false)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.bigqueryAPI.Grant(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMetadata.SAEmail)
	if err != nil {
		return errs.E(op, err)
	}

	if err := s.metabaseStorage.RestoreMetadata(ctx, datasetID); err != nil {
		return errs.E(op, err)
	}

	return nil
}

func MarshalUUID(id uuid.UUID) string {
	return strings.ToLower(base58.Encode(id[:]))
}

func AccountIDFromDatasetID(id uuid.UUID) string {
	return fmt.Sprintf("nada-%s", MarshalUUID(id))
}

func (s *metabaseService) ConstantServiceAccountEmailFromDatasetID(id uuid.UUID) string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", AccountIDFromDatasetID(id), s.gcpProject)
}

func (s *metabaseService) getOrcreateServiceAccountWithKeyAndPolicy(ctx context.Context, ds *service.Dataset) (*service.ServiceAccountWithPrivateKey, error) {
	const op errs.Op = "metabaseService.getOrcreateServiceAccountWithKeyAndPolicy"

	accountID := AccountIDFromDatasetID(ds.ID)

	sa, err := s.serviceAccountAPI.EnsureServiceAccountWithKeyAndBinding(ctx, &service.ServiceAccountRequest{
		ProjectID:   s.gcpProject,
		AccountID:   accountID,
		DisplayName: ds.Name,
		Description: fmt.Sprintf("Metabase service account for dataset %s", ds.ID.String()),
		Binding: &service.Binding{
			Role: fmt.Sprintf("projects/%s/roles/nada.metabase", s.gcpProject),
			Members: []string{
				fmt.Sprintf("serviceAccount:%s", s.ConstantServiceAccountEmailFromDatasetID(ds.ID)),
			},
		},
	})
	if err != nil {
		return nil, errs.E(op, err)
	}

	return sa, nil
}

func (s *metabaseService) createRestricted(ctx context.Context, ds *service.Dataset) error {
	const op errs.Op = "metabaseService.createRestricted"

	sa, err := s.getOrcreateServiceAccountWithKeyAndPolicy(ctx, ds)
	if err != nil {
		return err
	}

	permissionGroupName := slug.Make(fmt.Sprintf("%s-%s", ds.Name, MarshalUUID(ds.ID)))

	groupID, err := s.metabaseAPI.GetOrCreatePermissionGroup(ctx, permissionGroupName)
	if err != nil {
		return errs.E(op, err)
	}

	colID, err := s.metabaseAPI.CreateCollectionWithAccess(ctx, groupID, fmt.Sprintf("%s %s", ds.Name, service.MetabaseRestrictedCollectionTag))
	if err != nil {
		return errs.E(op, err)
	}

	err = s.create(ctx, dsWrapper{
		Dataset:         ds,
		Key:             string(sa.Key.PrivateKeyData),
		Email:           sa.Email,
		MetabaseGroupID: groupID,
		CollectionID:    colID,
	})
	if err != nil {
		archiveErr := s.metabaseAPI.ArchiveCollection(ctx, colID)
		if archiveErr != nil {
			return errs.E(op, fmt.Errorf("creating metabase database: %w, cleaning up collection: %w", err, archiveErr))
		}

		return errs.E(op, err)
	}

	return nil
}

func ensureUserInGroup(user *service.User, group string) error {
	const op errs.Op = "ensureUserInGroup"

	if user == nil || !user.GoogleGroups.Contains(group) {
		return errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user not in group %v", group))
	}

	return nil
}

func (s *metabaseService) GrantMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject, subjectType string) error {
	const op errs.Op = "metabaseService.GrantMetabaseAccess"

	if fmt.Sprintf("%s:%s", subjectType, subject) == s.groupAllUsers {
		err := s.addAllUsersDataset(ctx, dsID)
		if err != nil {
			return errs.E(op, err)
		}

		return nil
	}

	switch subjectType {
	case "user":
		err := s.addMetabaseGroupMember(ctx, dsID, subject)
		if err != nil {
			return errs.E(op, err)
		}
	default:
		log.Info().Msgf("Unsupported subject type %v for metabase access grant", subjectType)
	}

	return nil
}

type dsWrapper struct {
	Dataset         *service.Dataset
	Key             string
	Email           string
	MetabaseGroupID int
	CollectionID    int
}

func (s *metabaseService) addAllUsersDataset(ctx context.Context, dsID uuid.UUID) error {
	const op errs.Op = "metabaseService.addAllUsersDataset"

	mbMetadata, err := s.metabaseStorage.GetMetadata(ctx, dsID, true)
	if err != nil {
		if errs.KindIs(errs.NotExist, err) {
			ds, err := s.dataproductStorage.GetDataset(ctx, dsID)
			if err != nil {
				return errs.E(op, err)
			}

			err = s.create(ctx, dsWrapper{
				Dataset: ds,
				Key:     s.serviceAccount,
				Email:   s.serviceAccountEmail,
			})
			if err != nil {
				return errs.E(op, err)
			}

			return nil
		}

		return errs.E(op, err)
	}

	if mbMetadata.DeletedAt != nil {
		err := s.restore(ctx, dsID, mbMetadata)
		if err != nil {
			return errs.E(op, err)
		}

		return nil
	}

	if mbMetadata.PermissionGroupID == 0 {
		// All users database already exists in metabase
		return nil
	}

	err = s.metabaseAPI.OpenAccessToDatabase(ctx, mbMetadata.DatabaseID)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.metabaseAPI.DeletePermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.metabaseStorage.SetPermissionGroupMetabaseMetadata(ctx, mbMetadata.DatasetID, 0)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) create(ctx context.Context, ds dsWrapper) error {
	const op errs.Op = "metabaseService.create"

	datasource, err := s.bigqueryStorage.GetBigqueryDatasource(ctx, ds.Dataset.ID, false)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.bigqueryAPI.Grant(ctx, datasource.ProjectID, datasource.Dataset, datasource.Table, "serviceAccount:"+ds.Email)
	if err != nil {
		return errs.E(op, err)
	}

	dp, err := s.dataproductStorage.GetDataproduct(ctx, ds.Dataset.DataproductID)
	if err != nil {
		return errs.E(op, err)
	}

	dbID, err := s.metabaseAPI.CreateDatabase(ctx, dp.Owner.Group, ds.Dataset.Name, ds.Key, ds.Email, datasource)
	if err != nil {
		return errs.E(op, err)
	}

	if err := s.waitForDatabase(ctx, dbID, datasource.Table); err != nil {
		if err := s.cleanupOnCreateDatabaseError(ctx, dbID, ds); err != nil {
			return errs.E(op, err)
		}

		return errs.E(op, err)
	}

	mbMeta := &service.MetabaseMetadata{
		DatasetID:         ds.Dataset.ID,
		DatabaseID:        dbID,
		PermissionGroupID: ds.MetabaseGroupID,
		CollectionID:      ds.CollectionID,
		SAEmail:           ds.Email,
	}
	err = s.metabaseStorage.CreateMetadata(ctx, mbMeta)
	if err != nil {
		return errs.E(op, err)
	}

	if ds.MetabaseGroupID > 0 {
		err := s.metabaseAPI.RestrictAccessToDatabase(ctx, ds.MetabaseGroupID, dbID)
		if err != nil {
			return errs.E(op, err)
		}
	} else {
		err := s.metabaseAPI.OpenAccessToDatabase(ctx, dbID)
		if err != nil {
			return errs.E(op, err)
		}
	}

	if err := s.SyncTableVisibility(ctx, mbMeta, *datasource); err != nil {
		return errs.E(op, err)
	}

	if err := s.metabaseAPI.AutoMapSemanticTypes(ctx, dbID); err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) waitForDatabase(ctx context.Context, dbID int, tableName string) error {
	const op errs.Op = "metabaseService.waitForDatabase"

	for i := 0; i < 200; i++ {
		time.Sleep(100 * time.Millisecond)
		tables, err := s.metabaseAPI.Tables(ctx, dbID)
		if err != nil || len(tables) == 0 {
			continue
		}
		for _, tab := range tables {
			if tab.Name == tableName && len(tab.Fields) > 0 {
				return nil
			}
		}
	}

	return errs.E(errs.Internal, op, fmt.Errorf("unable to create database %v", tableName))
}

func (s *metabaseService) cleanupOnCreateDatabaseError(ctx context.Context, dbID int, ds dsWrapper) error {
	const op errs.Op = "metabaseService.cleanupOnCreateDatabaseError"

	dataset, err := s.dataproductStorage.GetDataset(ctx, ds.Dataset.ID)
	if err != nil {
		return errs.E(op, err)
	}
	services := dataset.Mappings

	for idx, msvc := range services {
		if msvc == service.MappingServiceMetabase {
			services = append(services[:idx], services[idx+1:]...)
		}
	}

	if err := s.metabaseAPI.DeleteDatabase(ctx, dbID); err != nil {
		return errs.E(op, err)
	}

	if ds.CollectionID != 0 {
		if err := s.metabaseAPI.DeletePermissionGroup(ctx, ds.MetabaseGroupID); err != nil {
			return errs.E(op, err)
		}

		if err := s.metabaseAPI.ArchiveCollection(ctx, ds.CollectionID); err != nil {
			return errs.E(op, err)
		}

		if err := s.serviceAccountAPI.DeleteServiceAccountAndBindings(ctx, s.gcpProject, ds.Email); err != nil {
			return errs.E(op, err)
		}
	}

	err = s.MapDataset(ctx, ds.Dataset.ID, services)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) DeleteDatabase(ctx context.Context, dsID uuid.UUID) error {
	const op errs.Op = "metabaseService.DeleteDatabase"

	mbMeta, err := s.metabaseStorage.GetMetadata(ctx, dsID, true)
	if err != nil {
		if errs.KindIs(errs.NotExist, err) {
			return nil
		}

		return errs.E(op, err)
	}

	if isRestrictedDatabase(mbMeta) {
		err = s.deleteRestrictedDatabase(ctx, dsID, mbMeta)
		if err != nil {
			return errs.E(op, err)
		}

		return nil
	}

	err = s.deleteAllUsersDatabase(ctx, dsID, mbMeta)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) deleteAllUsersDatabase(ctx context.Context, datasetID uuid.UUID, mbMeta *service.MetabaseMetadata) error {
	const op errs.Op = "metabaseService.deleteAllUsersDatabase"

	err := s.metabaseAPI.DeleteDatabase(ctx, mbMeta.DatabaseID)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.metabaseStorage.DeleteMetadata(ctx, mbMeta.DatasetID)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) deleteRestrictedDatabase(ctx context.Context, datasetID uuid.UUID, mbMeta *service.MetabaseMetadata) error {
	const op errs.Op = "metabaseService.deleteRestrictedDatabase"

	ds, err := s.bigqueryStorage.GetBigqueryDatasource(ctx, datasetID, false)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.bigqueryAPI.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMeta.SAEmail)
	if err != nil {
		return errs.E(op, err)
	}

	if err := s.serviceAccountAPI.DeleteServiceAccountAndBindings(ctx, s.gcpProject, mbMeta.SAEmail); err != nil {
		return errs.E(op, err)
	}

	if err := s.metabaseAPI.DeletePermissionGroup(ctx, mbMeta.PermissionGroupID); err != nil {
		return errs.E(op, err)
	}

	if err := s.metabaseAPI.ArchiveCollection(ctx, mbMeta.CollectionID); err != nil {
		return errs.E(op, err)
	}

	if err := s.metabaseAPI.DeleteDatabase(ctx, mbMeta.DatabaseID); err != nil {
		return errs.E(op, err)
	}

	if err := s.metabaseStorage.DeleteRestrictedMetadata(ctx, datasetID); err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) RevokeMetabaseAccessFromAccessID(ctx context.Context, accessID uuid.UUID) error {
	const op errs.Op = "metabaseService.RevokeMetabaseAccessFromAccessID"

	access, err := s.accessStorage.GetAccessToDataset(ctx, accessID)
	if err != nil {
		return errs.E(op, err)
	}

	// FIXME: should we check if the user is the owner?

	err = s.RevokeMetabaseAccess(ctx, access.DatasetID, access.Subject)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) RevokeMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) error {
	const op errs.Op = "metabaseService.RevokeMetabaseAccess"

	_, err := s.metabaseStorage.GetMetadata(ctx, dsID, false)
	if err != nil {
		if errs.KindIs(errs.NotExist, err) {
			return nil
		}

		return errs.E(op, err)
	}

	if subject == s.groupAllUsers {
		err := s.softDeleteDatabase(ctx, dsID)
		if err != nil {
			return errs.E(op, err)
		}
	}

	email, sType, err := parseSubject(subject)
	if err != nil {
		return errs.E(op, err)
	}

	if sType == "user" {
		err = s.removeMetabaseGroupMember(ctx, dsID, email)
		if err != nil {
			return errs.E(op, err)
		}
	}

	// FIXME: Are we supposed to throw an error if the sType isn't user, before we just logged and returned

	return nil
}

func (s *metabaseService) softDeleteDatabase(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "metabaseService.softDeleteDatabase"

	mbMeta, err := s.metabaseStorage.GetMetadata(ctx, datasetID, false)
	if err != nil {
		return errs.E(op, err)
	}

	ds, err := s.bigqueryStorage.GetBigqueryDatasource(ctx, datasetID, false)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.bigqueryAPI.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMeta.SAEmail)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.metabaseStorage.SoftDeleteMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) removeMetabaseGroupMember(ctx context.Context, dsID uuid.UUID, email string) error {
	const op errs.Op = "metabaseService.removeMetabaseGroupMember"

	mbMetadata, err := s.metabaseStorage.GetMetadata(ctx, dsID, false)
	if err != nil {
		return errs.E(op, err)
	}

	mbGroupMembers, err := s.metabaseAPI.GetPermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		return errs.E(op, err)
	}

	exists, memberID := memberExists(mbGroupMembers, email)
	if !exists {
		return nil
	}

	err = s.metabaseAPI.RemovePermissionGroupMember(ctx, memberID)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *metabaseService) SyncAllTablesVisibility(ctx context.Context) error {
	const op errs.Op = "metabaseService.SyncAllTablesVisibility"

	mbMetas, err := s.metabaseStorage.GetAllMetadata(ctx)
	if err != nil {
		return errs.E(op, err)
	}

	for _, db := range mbMetas {
		bq, err := s.bigqueryStorage.GetBigqueryDatasource(ctx, db.DatasetID, false)
		if err != nil {
			return errs.E(op, err)
		}

		if err := s.SyncTableVisibility(ctx, db, *bq); err != nil {
			return errs.E(op, fmt.Errorf("syncing table visibility for database %v: %w", db.DatasetID, err))
		}
	}

	return nil
}

func (s *metabaseService) SyncTableVisibility(ctx context.Context, mbMeta *service.MetabaseMetadata, bq service.BigQuery) error {
	const op errs.Op = "metabaseService.SyncTableVisibility"

	err := s.metabaseAPI.EnsureValidSession(ctx)
	if err != nil {
		return errs.E(op, err)
	}

	var buf io.ReadWriter
	res, err := s.metabaseAPI.PerformRequest(ctx, http.MethodGet, fmt.Sprintf("/database/%v/metadata?include_hidden=true", mbMeta.DatabaseID), buf)
	// FIXME: dont return the error code, lets handle it in the caller
	if res.StatusCode == http.StatusNotFound {
		// suppress error when database does not exist
		return nil
	}
	if err != nil {
		return errs.E(op, err)
	}
	defer res.Body.Close()

	var v struct {
		Tables []service.MetabaseTable `json:"tables"`
	}
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		return errs.E(errs.IO, op, err)
	}

	includedTables := []string{bq.Table}
	if !isRestrictedDatabase(mbMeta) {
		includedTables, err = s.metabaseStorage.GetOpenTablesInSameBigQueryDataset(ctx, bq.ProjectID, bq.Dataset)
		if err != nil {
			return errs.E(op, err)
		}
	}

	var includedIDs, excludedIDs []int

	for _, t := range v.Tables {
		if contains(includedTables, t.Name) {
			includedIDs = append(includedIDs, t.ID)
		} else {
			excludedIDs = append(excludedIDs, t.ID)
		}
	}

	if len(excludedIDs) > 0 {
		if err := s.metabaseAPI.HideTables(ctx, excludedIDs); err != nil {
			return errs.E(op, err)
		}
	}

	if len(includedIDs) > 0 {
		err = s.metabaseAPI.ShowTables(ctx, includedIDs)
		if err != nil {
			return errs.E(op, err)
		}
	}

	return nil
}

func isRestrictedDatabase(mbMeta *service.MetabaseMetadata) bool {
	return mbMeta.CollectionID != 0
}

func contains(elems []string, elem string) bool {
	for _, e := range elems {
		if e == elem {
			return true
		}
	}

	return false
}

func parseSubject(subject string) (string, string, error) {
	const op errs.Op = "parseSubject"

	s := strings.Split(subject, ":")
	if len(s) != 2 {
		return "", "", errs.E(errs.InvalidRequest, op, errs.Parameter("subject"), fmt.Errorf("invalid subject format, got: %s, should be type:email", subject))
	}

	return s[1], s[0], nil
}

func memberExists(groupMembers []service.MetabasePermissionGroupMember, subject string) (bool, int) {
	for _, m := range groupMembers {
		if m.Email == subject {
			return true, m.ID
		}
	}

	return false, -1
}

func NewMetabaseService(
	gcpProject string,
	serviceAccount string,
	serviceAccountEmail string,
	groupAllUsers string,
	mbapi service.MetabaseAPI,
	bqapi service.BigQueryAPI,
	saapi service.ServiceAccountAPI,
	tpms service.ThirdPartyMappingStorage,
	mbs service.MetabaseStorage,
	bqs service.BigQueryStorage,
	dps service.DataProductsStorage,
	as service.AccessStorage,
	log zerolog.Logger,
) *metabaseService {
	return &metabaseService{
		gcpProject:               gcpProject,
		serviceAccount:           serviceAccount,
		serviceAccountEmail:      serviceAccountEmail,
		groupAllUsers:            groupAllUsers,
		metabaseAPI:              mbapi,
		bigqueryAPI:              bqapi,
		serviceAccountAPI:        saapi,
		thirdPartyMappingStorage: tpms,
		metabaseStorage:          mbs,
		bigqueryStorage:          bqs,
		dataproductStorage:       dps,
		accessStorage:            as,
		log:                      log,
	}
}
