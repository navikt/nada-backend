package handlers

import (
	"context"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
)

type BigQueryHandler struct {
	service service.BigQueryService
}

type getBigQueryColumnsOpts struct {
	ProjectID string
	DatasetID string
	TableID   string
}

func (o getBigQueryColumnsOpts) Validate() error {
	return validation.ValidateStruct(&o,
		validation.Field(&o.ProjectID, validation.Required),
		validation.Field(&o.DatasetID, validation.Required),
		validation.Field(&o.TableID, validation.Required),
	)
}

func (h *BigQueryHandler) GetBigQueryColumns(ctx context.Context, r *http.Request, _ any) (*service.BQColumns, error) {
	const op errs.Op = "BigQueryHandler.GetBigQueryColumns"

	opts := getBigQueryColumnsOpts{
		ProjectID: r.URL.Query().Get("projectId"),
		DatasetID: r.URL.Query().Get("datasetId"),
		TableID:   r.URL.Query().Get("tableId"),
	}

	err := opts.Validate()
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	columns, err := h.service.GetBigQueryColumns(ctx, opts.ProjectID, opts.DatasetID, opts.TableID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return columns, nil
}

type getBigQueryTablesOpts struct {
	ProjectID string
	DatasetID string
}

func (o getBigQueryTablesOpts) Validate() error {
	return validation.ValidateStruct(&o,
		validation.Field(&o.ProjectID, validation.Required),
		validation.Field(&o.DatasetID, validation.Required),
	)
}

func (h *BigQueryHandler) GetBigQueryTables(ctx context.Context, r *http.Request, _ any) (*service.BQTables, error) {
	const op errs.Op = "BigQueryHandler.GetBigQueryTables"

	opts := getBigQueryTablesOpts{
		ProjectID: r.URL.Query().Get("projectId"),
		DatasetID: r.URL.Query().Get("datasetId"),
	}

	err := opts.Validate()
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	tables, err := h.service.GetBigQueryTables(ctx, opts.ProjectID, opts.DatasetID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return tables, nil
}

type getBigQueryDatasetsOpts struct {
	ProjectID string
}

func (o getBigQueryDatasetsOpts) Validate() error {
	return validation.ValidateStruct(&o,
		validation.Field(&o.ProjectID, validation.Required),
	)
}

func (h *BigQueryHandler) GetBigQueryDatasets(ctx context.Context, r *http.Request, _ any) (*service.BQDatasets, error) {
	const op errs.Op = "BigQueryHandler.GetBigQueryDatasets"

	opts := getBigQueryDatasetsOpts{
		ProjectID: r.URL.Query().Get("projectId"),
	}

	err := opts.Validate()
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	datasets, err := h.service.GetBigQueryDatasets(ctx, opts.ProjectID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return datasets, nil
}

func (h *BigQueryHandler) SyncBigQueryTables(ctx context.Context, _ *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "BigQueryHandler.SyncBigQueryTables"

	err := h.service.SyncBigQueryTables(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func NewBigQueryHandler(service service.BigQueryService) *BigQueryHandler {
	return &BigQueryHandler{service: service}
}
