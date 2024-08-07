package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/navikt/nada-backend/pkg/syncers/metabase_mapper"

	"github.com/navikt/nada-backend/pkg/service/core/transport"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

type MetabaseHandler struct {
	service      service.MetabaseService
	mappingQueue chan metabase_mapper.Work
}

func (h *MetabaseHandler) MapDataset(ctx context.Context, _ *http.Request, in service.DatasetMap) (*transport.Accepted, error) {
	const op errs.Op = "MetabaseHandler.MapDataset"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	err = h.service.CreateMappingRequest(ctx, user, id, in.Services)
	if err != nil {
		return nil, errs.E(op, err)
	}

	h.mappingQueue <- metabase_mapper.Work{
		DatasetID: id,
		Services:  in.Services,
	}

	return &transport.Accepted{}, nil
}

func NewMetabaseHandler(service service.MetabaseService, mappingQueue chan metabase_mapper.Work) *MetabaseHandler {
	return &MetabaseHandler{
		service:      service,
		mappingQueue: mappingQueue,
	}
}
