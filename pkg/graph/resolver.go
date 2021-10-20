package graph

import (
	"github.com/navikt/nada-backend/pkg/database"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	repo *database.Repo
}

func New(repo *database.Repo) *Resolver {
	return &Resolver{
		repo: repo,
	}
}
