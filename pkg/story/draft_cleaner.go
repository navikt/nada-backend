package story

import (
	"context"
	"time"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/sirupsen/logrus"
)

type DraftCleaner struct {
	repo *database.Repo
	log  *logrus.Entry
}

func NewDraftCleaner(repo *database.Repo, log *logrus.Entry) *DraftCleaner {
	return &DraftCleaner{
		repo: repo,
		log:  log,
	}
}

func (d *DraftCleaner) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		d.run(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *DraftCleaner) run(ctx context.Context) {
	if err := d.repo.CleanupStoryDrafts(ctx); err != nil {
		d.log.WithError(err).Error("Unable to clean up story drafts in database")
	}
}
