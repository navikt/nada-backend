package database

import (
	"context"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateSession(ctx context.Context, session *models.Session) error {
	return r.querier.CreateSession(ctx, gensql.CreateSessionParams{
		Token:   session.Token,
		Email:   session.Email,
		Name:    session.Name,
		Expires: session.Expires,
	})
}

func (r *Repo) GetSession(ctx context.Context, token string) (*models.Session, error) {
	sess, err := r.querier.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}

	return &models.Session{
		Token:   sess.Token,
		Email:   sess.Email,
		Name:    sess.Name,
		Created: sess.Created,
		Expires: sess.Expires,
	}, nil
}
