package core

import (
	"context"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.PollyService = &pollyService{}

type pollyService struct {
	pollyStorage service.PollyStorage
	pollyAPI     service.PollyAPI
}

func (p *pollyService) SearchPolly(ctx context.Context, q string) ([]*service.QueryPolly, error) {
	const op errs.Op = "pollyService.SearchPolly"

	res, err := p.pollyAPI.SearchPolly(ctx, q)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return res, nil
}

func NewPollyService(storage service.PollyStorage, api service.PollyAPI) *pollyService {
	return &pollyService{pollyStorage: storage, pollyAPI: api}
}
