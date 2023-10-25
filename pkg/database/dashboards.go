package database

import (
	"context"
	"database/sql"
	"errors"
)

func (r *Repo) GetDashboard(ctx context.Context, id string) (string, error) {
	dash, err := r.querier.GetDashboard(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return dash.Url, nil
}
