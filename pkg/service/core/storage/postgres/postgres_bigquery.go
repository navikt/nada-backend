package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/sqlc-dev/pqtype"
)

var _ service.BigQueryStorage = &bigQueryStorage{}

type bigQueryStorage struct {
	db *database.Repo
}

func (s *bigQueryStorage) UpdateBigqueryDatasource(ctx context.Context, input service.BigQueryDataSourceUpdate) error {
	const op errs.Op = "bigQueryStorage.UpdateBigqueryDatasource"

	err := s.db.Querier.UpdateBigqueryDatasource(ctx, gensql.UpdateBigqueryDatasourceParams{
		DatasetID: input.DatasetID,
		PiiTags: pqtype.NullRawMessage{
			RawMessage: json.RawMessage(ptrToString(input.PiiTags)),
			Valid:      len(ptrToString(input.PiiTags)) > 4,
		},
		PseudoColumns: input.PseudoColumns,
	})
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *bigQueryStorage) GetPseudoDatasourcesToDelete(ctx context.Context) ([]*service.BigQuery, error) {
	const op errs.Op = "bigQueryStorage.GetPseudoDatasourcesToDelete"

	rows, err := s.db.Querier.GetPseudoDatasourcesToDelete(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	var pseudoViews []*service.BigQuery
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
	const op errs.Op = "bigQueryStorage.UpdateBigqueryDatasourceMissing"

	err := s.db.Querier.UpdateBigqueryDatasourceMissing(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *bigQueryStorage) UpdateBigqueryDatasourceSchema(ctx context.Context, datasetID uuid.UUID, meta service.BigqueryMetadata) error {
	const op errs.Op = "bigQueryStorage.UpdateBigqueryDatasourceSchema"

	schemaJSON, err := json.Marshal(meta.Schema.Columns)
	if err != nil {
		return errs.E(errs.InvalidRequest, op, err)
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
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *bigQueryStorage) GetBigqueryDatasources(ctx context.Context) ([]*service.BigQuery, error) {
	const op errs.Op = "bigQueryStorage.GetBigqueryDatasources"

	bqs, err := s.db.Querier.GetBigqueryDatasources(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	ret := make([]*service.BigQuery, len(bqs))
	for i, bq := range bqs {
		piiTags := "{}"
		if bq.PiiTags.RawMessage != nil {
			piiTags = string(bq.PiiTags.RawMessage)
		}

		schema := service.BigquerySchema{}
		if bq.Schema.Valid {
			err := json.Unmarshal(bq.Schema.RawMessage, &schema.Columns)
			if err != nil {
				return nil, errs.E(errs.Internal, op, err)
			}
		}

		ret[i] = &service.BigQuery{
			ID:            bq.ID,
			DatasetID:     bq.DatasetID,
			ProjectID:     bq.ProjectID,
			Dataset:       bq.Dataset,
			Table:         bq.TableName,
			TableType:     service.BigQueryTableType(bq.TableType),
			LastModified:  bq.LastModified,
			Created:       bq.Created,
			Expires:       nullTimeToPtr(bq.Expires),
			Description:   bq.Description.String,
			PiiTags:       &piiTags,
			MissingSince:  &bq.MissingSince.Time,
			PseudoColumns: bq.PseudoColumns,
			Schema:        schema.Columns,
		}
	}

	return ret, nil
}

func (s *bigQueryStorage) GetBigqueryDatasource(ctx context.Context, datasetID uuid.UUID, isReference bool) (*service.BigQuery, error) {
	const op errs.Op = "bigQueryStorage.GetBigqueryDatasource"

	bq, err := s.db.Querier.GetBigqueryDatasource(ctx, gensql.GetBigqueryDatasourceParams{
		DatasetID:   datasetID,
		IsReference: isReference,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err)
		}

		return nil, errs.E(errs.Database, op, err)
	}

	piiTags := "{}"
	if bq.PiiTags.RawMessage != nil {
		piiTags = string(bq.PiiTags.RawMessage)
	}

	schema := service.BigquerySchema{}
	if bq.Schema.Valid {
		err := json.Unmarshal(bq.Schema.RawMessage, &schema.Columns)
		if err != nil {
			return nil, errs.E(errs.Internal, op, err)
		}
	}

	return &service.BigQuery{
		ID:            bq.ID,
		DatasetID:     bq.DatasetID,
		ProjectID:     bq.ProjectID,
		Dataset:       bq.Dataset,
		Table:         bq.TableName,
		TableType:     service.BigQueryTableType(bq.TableType),
		LastModified:  bq.LastModified,
		Created:       bq.Created,
		Expires:       nullTimeToPtr(bq.Expires),
		Description:   bq.Description.String,
		PiiTags:       &piiTags,
		MissingSince:  &bq.MissingSince.Time,
		PseudoColumns: bq.PseudoColumns,
		Schema:        schema.Columns,
	}, nil
}

func NewBigQueryStorage(db *database.Repo) *bigQueryStorage {
	return &bigQueryStorage{
		db: db,
	}
}
