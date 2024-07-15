package metabase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/access"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/event"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/cloudresourcemanager/v1"
	iam "google.golang.org/api/iam/v1"
)

type Metabase struct {
	repo       *database.Repo
	client     *Client
	events     *event.Manager
	sa         string
	saEmail    string
	errs       *prometheus.CounterVec
	iamService *iam.Service
	crmService *cloudresourcemanager.Service
	log        *logrus.Entry
	accessMgr  access.AccessManager
}

type dsWrapper struct {
	Dataset         *service.Dataset
	Key             string
	Email           string
	MetabaseGroupID int
	CollectionID    int
}

func New(repo *database.Repo, client *Client, eventMgr *event.Manager, accessMgr access.AccessManager, serviceAccount, serviceAccountEmail string, errs *prometheus.CounterVec, iamService *iam.Service, crmService *cloudresourcemanager.Service, log *logrus.Entry) *Metabase {
	m := &Metabase{
		repo:       repo,
		client:     client,
		events:     eventMgr,
		sa:         serviceAccount,
		saEmail:    serviceAccountEmail,
		errs:       errs,
		iamService: iamService,
		crmService: crmService,
		log:        log,
		accessMgr:  accessMgr,
	}
	m.events.ListenForDatasetGrant(m.grantMetabaseAccess)
	m.events.ListenForDatasetRevoke(m.revokeMetabaseAccess)
	m.events.ListenForDatasetAddMetabaseMapping(m.addDatasetMapping)
	m.events.ListenForDatasetRemoveMetabaseMapping(m.deleteDatabase)
	m.events.ListenForDatasetDelete(m.deleteDatabase)
	return m
}

func (m *Metabase) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		m.run(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Metabase) run(ctx context.Context) {
	log := m.log.WithField("subsystem", "metabase synchronizer")

	mbMetas, err := service.GetAllMetabaseMetadata(ctx)
	if err != nil {
		log.WithError(err).Error("reading metabase metadata")
	}

	for _, db := range mbMetas {
		bq, err := service.GetBigqueryDatasource(ctx, db.DatasetID, false)
		if err != nil {
			log.WithError(err).Error("getting bigquery datasource for dataset")
		}
		if err := m.SyncTableVisibility(ctx, db, *bq); err != nil {
			log.WithError(err).Warningf("hiding other tables for database %v", db.DatabaseID)
		}
	}
}

func (m *Metabase) SyncTableVisibility(ctx context.Context, mbMeta *service.MetabaseMetadata, bq service.BigQuery) error {
	if err := m.client.ensureValidSession(ctx); err != nil {
		return err
	}

	var buf io.ReadWriter
	res, err := m.client.performRequest(ctx, http.MethodGet, fmt.Sprintf("/database/%v/metadata?include_hidden=true", mbMeta.DatabaseID), buf)
	if res.StatusCode == 404 {
		// suppress error when database does not exist
		return nil
	}
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var v struct {
		Tables []Table `json:"tables"`
	}
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		return err
	}

	includedTables := []string{bq.Table}
	if !isRestrictedDatabase(mbMeta) {
		includedTables, err = service.GetOpenMetabaseTablesInSameBigQueryDataset(ctx, bq.ProjectID, bq.Dataset)
		if err != nil {
			return err
		}
	}

	includedIDs := []int{}
	excludedIDs := []int{}
	for _, t := range v.Tables {
		if contains(includedTables, t.Name) {
			includedIDs = append(includedIDs, t.ID)
		} else {
			excludedIDs = append(excludedIDs, t.ID)
		}
	}

	if len(excludedIDs) != 0 {
		if err := m.client.HideTables(ctx, excludedIDs); err != nil {
			return err
		}
	}

	return m.client.ShowTables(ctx, includedIDs)
}

func MarshalUUID(id uuid.UUID) string {
	return strings.ToLower(base58.Encode(id[:]))
}

func memberExists(groupMembers []PermissionGroupMember, subject string) (bool, int) {
	for _, m := range groupMembers {
		if m.Email == subject {
			return true, m.ID
		}
	}
	return false, -1
}

func parseSubject(subject string) (string, string, error) {
	s := strings.Split(subject, ":")
	if len(s) != 2 {
		return "", "", fmt.Errorf("invalid subject format, should be type:email")
	}

	return s[1], s[0], nil
}

func contains(elems []string, elem string) bool {
	for _, e := range elems {
		if e == elem {
			return true
		}
	}

	return false
}
