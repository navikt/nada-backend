package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"io"
	"net/http"
	"strings"
	"time"
)

type metabaseService struct {
	gcpProject          string
	serviceAccount      string
	serviceAccountEmail string

	metabaseAPI       service.MetabaseAPI
	bigqueryAPI       service.BigQueryAPI
	serviceAccountAPI service.ServiceAccountAPI

	thirdPartyMappingStorage service.ThirdPartyMappingStorage
	metabaseStorage          service.MetabaseStorage
	bigqueryStorage          service.BigQueryStorage
	dataproductStorage       service.DataProductStorage
	accessStorage            service.AccessStorage
}

var _ service.MetabaseService = &metabaseService{}

func (s *metabaseService) MapDataset(ctx context.Context, datasetID string, services []string) (*service.Dataset, error) {
	ds, err := s.dataproductStorage.GetDataset(ctx, datasetID)
	if err != nil {
		return nil, err
	}

	dp, err := s.dataproductStorage.GetDataproduct(ctx, ds.DataproductID.String())
	if err != nil {
		return nil, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		// FIXME: not sure if errunauthorized will work
		return nil, service.ErrUnauthorized
	}

	err = s.thirdPartyMappingStorage.MapDataset(ctx, datasetID, services)
	if err != nil {
		return nil, fmt.Errorf("mapping dataset: %w", err)
	}

	mapMetabase := false
	for _, svc := range services {
		if svc == service.MappingServiceMetabase {
			mapMetabase = true

			err := s.addDatasetMapping(ctx, uuid.MustParse(datasetID))
			if err != nil {
				return nil, fmt.Errorf("adding dataset mapping: %w", err)
			}

			break
		}
	}

	if !mapMetabase {
		err = s.DeleteDatabase(ctx, uuid.MustParse(datasetID))
		if err != nil {
			return nil, fmt.Errorf("delete database: %w", err)
		}
	}

	return ds, nil
}

func (s *metabaseService) addDatasetMapping(ctx context.Context, dsID uuid.UUID) error {
	accesses, err := s.accessStorage.ListActiveAccessToDataset(ctx, dsID)
	if err != nil {
		return fmt.Errorf("listing active access to dataset: %w", err)
	}

	if containsAllUsers(accesses) {
		err := s.addAllUsersDataset(ctx, dsID)
		if err != nil {
			return fmt.Errorf("adding all users dataset: %w", err)
		}

		return nil
	}

	err = s.addRestrictedDatasetMapping(ctx, dsID)
	if err != nil {
		return fmt.Errorf("adding restricted dataset mapping: %w", err)
	}

	return nil
}

func containsAllUsers(accesses []*service.Access) bool {
	for _, a := range accesses {
		if a.Subject == "group:all-users@nav.no" {
			return true
		}
	}

	return false
}

func (s *metabaseService) addRestrictedDatasetMapping(ctx context.Context, dsID uuid.UUID) error {
	mbMeta, err := s.metabaseStorage.GetMetadata(ctx, dsID, true)
	if errors.Is(err, sql.ErrNoRows) {
		ds, err := s.dataproductStorage.GetDataset(ctx, dsID.String())
		if err != nil {
			return fmt.Errorf("getting dataset: %w", err)
		}

		if err := s.createRestricted(ctx, ds); err != nil {
			return fmt.Errorf("create restricted database: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("get metabase metadata: %w", err)
	}

	if mbMeta != nil && mbMeta.PermissionGroupID == 0 {
		return fmt.Errorf("not allowed to expose a previously open database as a restricted")
	}

	if mbMeta != nil && mbMeta.DeletedAt != nil {
		if err := s.restore(ctx, dsID, mbMeta); err != nil {
			return fmt.Errorf("restoring db: %w", err)
		}
	}

	if err := s.grantAccessesOnCreation(ctx, dsID); err != nil {
		return fmt.Errorf("granting accesses on creation: %w", err)
	}

	return nil
}

func (s *metabaseService) grantAccessesOnCreation(ctx context.Context, dsID uuid.UUID) error {
	accesses, err := s.accessStorage.ListActiveAccessToDataset(ctx, dsID)
	if err != nil {
		return err
	}

	for _, a := range accesses {
		email, sType, err := parseSubject(a.Subject)
		if err != nil {
			return fmt.Errorf("parse subject %v: %w", a.Subject, err)
		}

		switch sType {
		case "user":
			err := s.addMetabaseGroupMember(ctx, dsID, email)
			if err != nil {
				return fmt.Errorf("adding metabase group member: %w", err)
			}
		default:
			return fmt.Errorf("unsupported subject type %v for metabase access grant", sType)
		}
	}

	return nil
}

func (s *metabaseService) addMetabaseGroupMember(ctx context.Context, dsID uuid.UUID, email string) error {
	mbMetadata, err := s.metabaseStorage.GetMetadata(ctx, dsID, false)
	if err != nil {
		return fmt.Errorf("get metabase metadata: %w", err)
	}

	mbGroupMembers, err := s.metabaseAPI.GetPermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		return fmt.Errorf("get permission group: %w", err)
	}

	exists, _ := memberExists(mbGroupMembers, email)
	if exists {
		return nil
	}

	if err := s.metabaseAPI.AddPermissionGroupMember(ctx, mbMetadata.PermissionGroupID, email); err != nil {
		return fmt.Errorf("add permission group member: %w", err)
	}

	return nil
}

func (s *metabaseService) restore(ctx context.Context, datasetID uuid.UUID, mbMetadata *service.MetabaseMetadata) error {
	ds, apierr := s.bigqueryStorage.GetBigqueryDatasource(ctx, datasetID, false)
	if apierr != nil {
		return apierr
	}

	err := s.bigqueryAPI.Grant(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMetadata.SAEmail)
	if err != nil {
		return fmt.Errorf("granting access: %w", err)
	}

	if err := s.metabaseStorage.RestoreMetadata(ctx, datasetID); err != nil {
		return fmt.Errorf("restoring metabase metadata: %w", err)
	}

	return nil
}

func (s *metabaseService) createRestricted(ctx context.Context, ds *service.Dataset) error {
	groupID, err := s.metabaseAPI.CreatePermissionGroup(ctx, ds.Name)
	if err != nil {
		return fmt.Errorf("creating permission group: %w", err)
	}

	colID, err := s.metabaseAPI.CreateCollectionWithAccess(ctx, []int{groupID}, ds.Name)
	if err != nil {
		return fmt.Errorf("creating collection with access: %w", err)
	}

	key, email, err := s.serviceAccountAPI.CreateServiceAccount(s.gcpProject, ds)
	if err != nil {
		return fmt.Errorf("creating service account: %w", err)
	}

	err = s.create(ctx, dsWrapper{
		Dataset:         ds,
		Key:             string(key),
		Email:           email,
		MetabaseGroupID: groupID,
		CollectionID:    colID,
	})
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}

	return nil
}

// FIXME: I will keep this for now, but if we need this information
// we should parse it out from the request and pass it down the line
// it shouldn't just magically appear
func ensureUserInGroup(ctx context.Context, group string) error {
	user := auth.GetUser(ctx)
	if user == nil || !user.GoogleGroups.Contains(group) {
		return service.ErrUnauthorized
	}
	return nil
}

func (s *metabaseService) GrantMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) error {
	if subject == "group:all-users@nav.no" {
		err := s.addAllUsersDataset(ctx, dsID)
		if err != nil {
			return fmt.Errorf("adding all users dataset: %w", err)
		}

		return nil
	}

	email, sType, err := parseSubject(subject)
	if err != nil {
		return fmt.Errorf("parsing subject %v: %w", subject, err)
	}

	switch sType {
	case "user":
		err := s.addMetabaseGroupMember(ctx, dsID, email)
		if err != nil {
			return fmt.Errorf("adding metabase group member: %w", err)
		}
	default:
		return fmt.Errorf("unsupported subject type %v for metabase access grant", sType)
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
	mbMetadata, err := s.metabaseStorage.GetMetadata(ctx, dsID, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ds, apierr := s.dataproductStorage.GetDataset(ctx, dsID.String())
			if apierr != nil {
				return fmt.Errorf("getting dataset: %w", apierr)
			}

			err = s.create(ctx, dsWrapper{
				Dataset: ds,
				Key:     s.serviceAccount,
				Email:   s.serviceAccountEmail,
			})
			if err != nil {
				return fmt.Errorf("create all users database: %w", err)
			}

			return nil
		}

		return fmt.Errorf("get metabase metadata: %w", err)
	}

	if mbMetadata.DeletedAt != nil {
		err := s.restore(ctx, dsID, mbMetadata)
		if err != nil {
			return fmt.Errorf("restoring db: %w", err)
		}

		return nil
	}

	if mbMetadata.PermissionGroupID == 0 {
		// All users database already exists in metabase
		return nil
	}

	err = s.metabaseAPI.OpenAccessToDatabase(ctx, mbMetadata.DatabaseID)
	if err != nil {
		return fmt.Errorf("opening access to database: %w", err)
	}

	err = s.metabaseAPI.DeletePermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		return fmt.Errorf("deleting permission group: %w", err)
	}

	err = s.metabaseStorage.SetPermissionGroupMetabaseMetadata(ctx, mbMetadata.DatasetID, 0)
	if err != nil {
		return fmt.Errorf("setting permission group to all users: %w", err)
	}

	return nil
}

func (s *metabaseService) create(ctx context.Context, ds dsWrapper) error {
	datasource, err := s.bigqueryStorage.GetBigqueryDatasource(ctx, ds.Dataset.ID, false)
	if err != nil {
		return fmt.Errorf("getting bigquery datasource: %w", err)
	}

	err = s.bigqueryAPI.Grant(ctx, datasource.ProjectID, datasource.Dataset, datasource.Table, "serviceAccount:"+ds.Email)
	if err != nil {
		return fmt.Errorf("granting access: %w", err)
	}

	dp, err := s.dataproductStorage.GetDataproduct(ctx, ds.Dataset.DataproductID.String())
	if err != nil {
		return fmt.Errorf("getting dataproduct: %w", err)
	}

	dbID, err := s.metabaseAPI.CreateDatabase(ctx, dp.Owner.Group, ds.Dataset.Name, ds.Key, ds.Email, datasource)
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}

	if err := s.waitForDatabase(ctx, dbID, datasource.Table); err != nil {
		if err := s.cleanupOnCreateDatabaseError(ctx, dbID, ds); err != nil {
			return fmt.Errorf("cleanup on create database error: %w", err)
		}

		return fmt.Errorf("waiting for database: %w", err)
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
		return fmt.Errorf("creating metabase metadata: %w", err)
	}

	if ds.MetabaseGroupID > 0 {
		err := s.metabaseAPI.RestrictAccessToDatabase(ctx, []int{ds.MetabaseGroupID}, dbID)
		if err != nil {
			return err
		}
	} else {
		err := s.metabaseAPI.OpenAccessToDatabase(ctx, dbID)
		if err != nil {
			return err
		}
	}

	if err := s.SyncTableVisibility(ctx, mbMeta, *datasource); err != nil {
		return err
	}

	if err := s.metabaseAPI.AutoMapSemanticTypes(ctx, dbID); err != nil {
		return err
	}

	return nil
}

func (s *metabaseService) waitForDatabase(ctx context.Context, dbID int, tableName string) error {
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

	return fmt.Errorf("unable to create database %v", tableName)
}

func (s *metabaseService) cleanupOnCreateDatabaseError(ctx context.Context, dbID int, ds dsWrapper) error {
	dataset, err := s.dataproductStorage.GetDataset(ctx, ds.Dataset.ID.String())
	if err != nil {
		return err
	}
	services := dataset.Mappings

	for idx, msvc := range services {
		if msvc == service.MappingServiceMetabase {
			services = append(services[:idx], services[idx+1:]...)
		}
	}

	if err := s.metabaseAPI.DeleteDatabase(ctx, dbID); err != nil {
		return err
	}

	if ds.CollectionID != 0 {
		if err := s.metabaseAPI.DeletePermissionGroup(ctx, ds.MetabaseGroupID); err != nil {
			return err
		}

		if err := s.metabaseAPI.ArchiveCollection(ctx, ds.CollectionID); err != nil {
			return err
		}

		if err := s.serviceAccountAPI.DeleteServiceAccount(s.gcpProject, ds.Email); err != nil {
			return err
		}
	}

	_, err = s.MapDataset(ctx, ds.Dataset.ID.String(), services)
	if err != nil {
		return fmt.Errorf("mapping dataset: %w", err)
	}

	return nil
}

func (s *metabaseService) DeleteDatabase(ctx context.Context, dsID uuid.UUID) error {
	mbMeta, err := s.metabaseStorage.GetMetadata(ctx, dsID, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("get metabase metadata: %w", err)
	}

	if isRestrictedDatabase(mbMeta) {
		err = s.deleteRestrictedDatabase(ctx, dsID, mbMeta)
		if err != nil {
			return fmt.Errorf("delete restricted database: %w", err)
		}

		return nil
	}

	err = s.deleteAllUsersDatabase(ctx, dsID, mbMeta)
	if err != nil {
		return fmt.Errorf("delete all-users database: %w", err)
	}

	return nil
}

func (s *metabaseService) deleteAllUsersDatabase(ctx context.Context, datasetID uuid.UUID, mbMeta *service.MetabaseMetadata) error {
	err := s.metabaseAPI.DeleteDatabase(ctx, mbMeta.DatabaseID)
	if err != nil {
		return fmt.Errorf("delete all-users database: %w", err)
	}

	err = s.metabaseStorage.DeleteMetadata(ctx, mbMeta.DatasetID)
	if err != nil {
		return fmt.Errorf("delete all-users metabase metadata: %w", err)
	}

	return nil
}

func (s *metabaseService) deleteRestrictedDatabase(ctx context.Context, datasetID uuid.UUID, mbMeta *service.MetabaseMetadata) error {
	ds, apierr := s.bigqueryStorage.GetBigqueryDatasource(ctx, datasetID, false)
	if apierr != nil {
		return fmt.Errorf("get bigquery datasource: %w", apierr)
	}

	err := s.bigqueryAPI.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMeta.SAEmail)
	if err != nil {
		return fmt.Errorf("revoke access: %w", err)
	}

	if err := s.serviceAccountAPI.DeleteServiceAccount(s.gcpProject, mbMeta.SAEmail); err != nil {
		return fmt.Errorf("delete service account: %w", err)
	}

	if err := s.metabaseAPI.DeletePermissionGroup(ctx, mbMeta.PermissionGroupID); err != nil {
		return fmt.Errorf("delete permission group: %w", err)
	}

	if err := s.metabaseAPI.ArchiveCollection(ctx, mbMeta.CollectionID); err != nil {
		return fmt.Errorf("archive collection: %w", err)
	}

	if err := s.metabaseAPI.DeleteDatabase(ctx, mbMeta.DatabaseID); err != nil {
		return fmt.Errorf("delete restricted database: %w", err)
	}

	if err := s.metabaseStorage.DeleteRestrictedMetadata(ctx, datasetID); err != nil {
		return fmt.Errorf("delete restricted metabase metadata: %w", err)
	}

	return nil
}

func (s *metabaseService) RevokeMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) error {
	if subject == "group:all-users@nav.no" {
		err := s.softDeleteDatabase(ctx, dsID)
		if err != nil {
			return fmt.Errorf("soft delete database: %v", err)
		}
	}

	email, sType, err := parseSubject(subject)
	if err != nil {
		return fmt.Errorf("parsing subject %v: %w", subject, err)
	}

	if sType == "user" {
		err = s.removeMetabaseGroupMember(ctx, dsID, email)
		if err != nil {
			return fmt.Errorf("remove metabase group member: %w", err)
		}
	}

	// FIXME: Are we supposed to throw an error if the sType isn't user, before we just logged and returned

	return nil
}

func (s *metabaseService) softDeleteDatabase(ctx context.Context, datasetID uuid.UUID) error {
	mbMeta, err := s.metabaseStorage.GetMetadata(ctx, datasetID, false)
	if err != nil {
		return fmt.Errorf("get metabase metadata: %w", err)
	}

	ds, apierr := s.bigqueryStorage.GetBigqueryDatasource(ctx, datasetID, false)
	if apierr != nil {
		return fmt.Errorf("get bigquery datasource: %w", apierr)
	}

	err = s.bigqueryAPI.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, "serviceAccount:"+mbMeta.SAEmail)
	if err != nil {
		return fmt.Errorf("revoke access: %w", err)
	}

	err = s.metabaseStorage.SoftDeleteMetadata(ctx, datasetID)
	if err != nil {
		return fmt.Errorf("soft delete metabase metadata: %w", err)
	}

	return nil
}

func (s *metabaseService) removeMetabaseGroupMember(ctx context.Context, dsID uuid.UUID, email string) error {
	mbMetadata, err := s.metabaseStorage.GetMetadata(ctx, dsID, false)
	if err != nil {
		return fmt.Errorf("get metabase metadata: %w", err)
	}

	mbGroupMembers, err := s.metabaseAPI.GetPermissionGroup(ctx, mbMetadata.PermissionGroupID)
	if err != nil {
		return fmt.Errorf("get permission group: %w", err)
	}

	exists, memberID := memberExists(mbGroupMembers, email)
	if !exists {
		return nil
	}

	err = s.metabaseAPI.RemovePermissionGroupMember(ctx, memberID)
	if err != nil {
		return fmt.Errorf("remove permission group member: %w", err)
	}

	return nil
}

func (s *metabaseService) SyncAllTablesVisibility(ctx context.Context) error {
	mbMetas, err := s.metabaseStorage.GetAllMetadata(ctx)
	if err != nil {
		return fmt.Errorf("getting all metabase metadata: %w", err)
	}

	for _, db := range mbMetas {
		bq, err := s.bigqueryStorage.GetBigqueryDatasource(ctx, db.DatasetID, false)
		if err != nil {
			return fmt.Errorf("getting bigquery datasource: %w", err)
		}

		if err := s.SyncTableVisibility(ctx, db, *bq); err != nil {
			return fmt.Errorf("syncing table visibility: %w", err)
		}
	}

	return nil
}

func (s *metabaseService) SyncTableVisibility(ctx context.Context, mbMeta *service.MetabaseMetadata, bq service.BigQuery) error {
	err := s.metabaseAPI.EnsureValidSession(ctx)
	if err != nil {
		return fmt.Errorf("ensuring valid session: %w", err)
	}

	var buf io.ReadWriter
	res, err := s.metabaseAPI.PerformRequest(ctx, http.MethodGet, fmt.Sprintf("/database/%v/metadata?include_hidden=true", mbMeta.DatabaseID), buf)
	// FIXME: dont return the error code, lets handle it in the caller
	if res.StatusCode == 404 {
		// suppress error when database does not exist
		return nil
	}
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer res.Body.Close()

	var v struct {
		Tables []service.MetabaseTable `json:"tables"`
	}
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	includedTables := []string{bq.Table}
	if !isRestrictedDatabase(mbMeta) {
		includedTables, err = s.metabaseStorage.GetOpenTablesInSameBigQueryDataset(ctx, bq.ProjectID, bq.Dataset)
		if err != nil {
			return fmt.Errorf("getting open metabase tables in same bigquery dataset: %w", err)
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

	if len(excludedIDs) != 0 {
		if err := s.metabaseAPI.HideTables(ctx, excludedIDs); err != nil {
			return fmt.Errorf("hiding tables: %w", err)
		}
	}

	err = s.metabaseAPI.ShowTables(ctx, includedIDs)
	if err != nil {
		return fmt.Errorf("showing tables: %w", err)
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
	s := strings.Split(subject, ":")
	if len(s) != 2 {
		return "", "", fmt.Errorf("invalid subject format, should be type:email")
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
	mbapi service.MetabaseAPI,
	bqapi service.BigQueryAPI,
	saapi service.ServiceAccountAPI,
	tpms service.ThirdPartyMappingStorage,
	mbs service.MetabaseStorage,
	bqs service.BigQueryStorage,
	dps service.DataProductStorage,
	as service.AccessStorage,
) *metabaseService {
	return &metabaseService{
		gcpProject:               gcpProject,
		serviceAccount:           serviceAccount,
		serviceAccountEmail:      serviceAccountEmail,
		metabaseAPI:              mbapi,
		bigqueryAPI:              bqapi,
		serviceAccountAPI:        saapi,
		thirdPartyMappingStorage: tpms,
		metabaseStorage:          mbs,
		bigqueryStorage:          bqs,
		dataproductStorage:       dps,
		accessStorage:            as,
	}
}
