package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/navikt/nada-backend/pkg/database/gensql"
)

var queries *gensql.Queries

type Session struct {
	Token       string
	AccessToken string
	Email       string `json:"preferred_username"`
	Name        string `json:"name"`
	Created     time.Time
	Expires     time.Time
}

func Init(db *sql.DB) {
	queries = gensql.New(db)
}

func CreateSession(ctx context.Context, session *Session) error {
	return queries.CreateSession(ctx, gensql.CreateSessionParams{
		Token:       session.Token,
		AccessToken: session.AccessToken,
		Email:       session.Email,
		Name:        session.Name,
		Expires:     session.Expires,
	})
}

func GetSession(ctx context.Context, token string) (*Session, error) {
	sess, err := queries.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}

	return &Session{
		Token:       sess.Token,
		AccessToken: sess.AccessToken,
		Email:       sess.Email,
		Name:        sess.Name,
		Created:     sess.Created,
		Expires:     sess.Expires,
	}, nil
}

func DeleteSession(ctx context.Context, token string) error {
	return queries.DeleteSession(ctx, token)
}
