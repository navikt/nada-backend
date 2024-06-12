package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/sqlc-dev/pqtype"
	"strings"
)

var _ service.BigQueryStorage = &bigQueryStorage{}

type bigQueryStorage struct {
	db *database.Repo
}

func (s *bigQueryStorage) UpdateBigqueryDatasource(ctx context.Context, input service.BigQueryDataSourceUpdate) error {
	err := s.db.Querier.UpdateBigqueryDatasource(ctx, gensql.UpdateBigqueryDatasourceParams{
		DatasetID: input.DatasetID,
		PiiTags: pqtype.NullRawMessage{
			RawMessage: json.RawMessage(ptrToString(input.PiiTags)),
			Valid:      len(ptrToString(input.PiiTags)) > 4,
		},
		PseudoColumns: input.PseudoColumns,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *bigQueryStorage) GetPseudoDatasourcesToDelete(ctx context.Context) ([]*service.BigQuery, error) {
	rows, err := s.db.Querier.GetPseudoDatasourcesToDelete(ctx)
	if err != nil {
		return nil, err
	}

	pseudoViews := []*service.BigQuery{}
	for _, d := range rows {
		pseudoViews = append(pseudoViews, &service.BigQuery{
			ID:            d.ID,
			Dataset:       d.Dataset,
			ProjectID:     d.ProjectID,
			Table:         d.TableName,
			PseudoColumns: d.PseudoColumns,
		})
	}

	return pseudoViews, nil
}

func (s *bigQueryStorage) UpdateBigqueryDatasourceMissing(ctx context.Context, datasetID uuid.UUID) error {
	return s.db.Querier.UpdateBigqueryDatasourceMissing(ctx, datasetID)
}

func (s *bigQueryStorage) UpdateBigqueryDatasourceSchema(ctx context.Context, datasetID uuid.UUID, meta service.BigqueryMetadata) error {
	schemaJSON, err := json.Marshal(meta.Schema.Columns)
	if err != nil {
		return fmt.Errorf("marshalling schema: %w", err)
	}

	err = s.db.Querier.UpdateBigqueryDatasourceSchema(ctx, gensql.UpdateBigqueryDatasourceSchemaParams{
		Schema: pqtype.NullRawMessage{
			RawMessage: schemaJSON,
			Valid:      true,
		},
		LastModified:  meta.LastModified,
		Expires:       sql.NullTime{Time: meta.Expires, Valid: !meta.Expires.IsZero()},
		Description:   sql.NullString{String: meta.Description, Valid: true},
		PseudoColumns: nil,
		DatasetID:     datasetID,
	})
	if err != nil {
		return fmt.Errorf("writing metadata to database: %w", err)
	}

	return nil
}

func (s *bigQueryStorage) GetBigqueryDatasources(ctx context.Context) ([]*service.BigQuery, error) {
	bqs, err := s.db.Querier.GetBigqueryDatasources(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("getting bigquery datasources: %w", err)
	}

	ret := make([]*service.BigQuery, len(bqs))
	for i, bq := range bqs {
		piiTags := "{}"
		if bq.PiiTags.RawMessage != nil {
			piiTags = string(bq.PiiTags.RawMessage)
		}

		ret[i] = &service.BigQuery{
			ID:            bq.ID,
			DatasetID:     bq.DatasetID,
			ProjectID:     bq.ProjectID,
			Dataset:       bq.Dataset,
			Table:         bq.TableName,
			TableType:     service.BigQueryType(strings.ToLower(bq.TableType)),
			LastModified:  bq.LastModified,
			Created:       bq.Created,
			Expires:       nullTimeToPtr(bq.Expires),
			Description:   bq.Description.String,
			PiiTags:       &piiTags,
			MissingSince:  &bq.MissingSince.Time,
			PseudoColumns: bq.PseudoColumns,
		}
	}

	return ret, nil
}

func (s *bigQueryStorage) GetBigqueryDatasource(ctx context.Context, datasetID uuid.UUID, isReference bool) (*service.BigQuery, error) {
	bq, err := s.db.Querier.GetBigqueryDatasource(ctx, gensql.GetBigqueryDatasourceParams{
		DatasetID:   datasetID,
		IsReference: isReference,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("getting bigquery datasource: %w", err)
		}

		return nil, fmt.Errorf("getting bigquery datasource: %w", err)
	}

	piiTags := "{}"
	if bq.PiiTags.RawMessage != nil {
		piiTags = string(bq.PiiTags.RawMessage)
	}

	return &service.BigQuery{
		ID:            bq.ID,
		DatasetID:     bq.DatasetID,
		ProjectID:     bq.ProjectID,
		Dataset:       bq.Dataset,
		Table:         bq.TableName,
		TableType:     service.BigQueryType(strings.ToLower(bq.TableType)),
		LastModified:  bq.LastModified,
		Created:       bq.Created,
		Expires:       nullTimeToPtr(bq.Expires),
		Description:   bq.Description.String,
		PiiTags:       &piiTags,
		MissingSince:  &bq.MissingSince.Time,
		PseudoColumns: bq.PseudoColumns,
	}, nil
}

func NewBigQueryStorage(db *database.Repo) *bigQueryStorage {
	return &bigQueryStorage{
		db: db,
	}
}
