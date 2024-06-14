package core

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.PollyService = &pollyService{}

type pollyService struct {
	pollyStorage service.PollyStorage
	pollyAPI     service.PollyAPI
}

func (p *pollyService) SearchPolly(ctx context.Context, q string) ([]*service.QueryPolly, error) {
	return p.pollyAPI.SearchPolly(ctx, q)
}

func NewPollyService(storage service.PollyStorage, api service.PollyAPI) *pollyService {
	return &pollyService{pollyStorage: storage, pollyAPI: api}
}
