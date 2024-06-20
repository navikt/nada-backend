package handlers

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"net/http"
)

type bigQueryHandler struct {
	service service.BigQueryService
}

func (h *bigQueryHandler) GetBigQueryColumns(ctx context.Context, r *http.Request, _ any) (*service.BQColumns, error) {
	projectID := r.URL.Query().Get("projectId")
	datasetID := r.URL.Query().Get("datasetId")
	tableID := r.URL.Query().Get("tableId")

	return h.service.GetBigQueryColumns(ctx, projectID, datasetID, tableID)
}

func (h *bigQueryHandler) GetBigQueryTables(ctx context.Context, r *http.Request, _ any) (*service.BQTables, error) {
	projectID := r.URL.Query().Get("projectId")
	datasetID := r.URL.Query().Get("datasetId")

	return h.service.GetBigQueryTables(ctx, projectID, datasetID)
}

func (h *bigQueryHandler) GetBigQueryDatasets(ctx context.Context, r *http.Request, _ any) (*service.BQDatasets, error) {
	projectID := r.URL.Query().Get("projectId")

	return h.service.GetBigQueryDatasets(ctx, projectID)
}

func (h *bigQueryHandler) SyncBigQueryTables(ctx context.Context, _ *http.Request, _ any) (*transport.Empty, error) {
	err := h.service.SyncBigQueryTables(ctx)
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func NewBigQueryHandler(service service.BigQueryService) *bigQueryHandler {
	return &bigQueryHandler{service: service}
}
