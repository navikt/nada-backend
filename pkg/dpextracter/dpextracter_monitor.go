package dpextracter

import (
	"context"
	"time"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/sirupsen/logrus"
)

type DPExtractMonitor struct {
	repo        *database.Repo
	dpExtracter *DPExtracter
	log         *logrus.Entry
}

func NewMonitor(repo *database.Repo, dpExtracter *DPExtracter, log *logrus.Entry) *DPExtractMonitor {
	return &DPExtractMonitor{
		repo:        repo,
		dpExtracter: dpExtracter,
		log:         log,
	}
}

func (d *DPExtractMonitor) Run(ctx context.Context, frequency time.Duration) {
	// d.dpExtracter.eventMgr.ListenForDataproductExtract(d.createDataproductExtract)

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

func (d *DPExtractMonitor) run(ctx context.Context) {
	extractions, err := d.repo.GetUnreadyDataproductExtractions(ctx)
	if err != nil {
		return
	}

	for _, e := range extractions {
		if d.jobDone(ctx, e.JobID) {
			if err := d.repo.SetDataproductExtractReady(ctx, e.ID); err != nil {
				d.log.WithField("dataproductID", e.DataproductID).Errorf("set dataproduct ready", err)
			}

			if err := d.repo.SetDataproductExtractExpired(ctx, e.ID); err != nil {
				d.log.WithField("dataproductID", e.DataproductID).Errorf("set dataproduct expired", err)
			}
		}
	}
}

func (d *DPExtractMonitor) jobDone(ctx context.Context, jobID string) bool {
	job, err := d.dpExtracter.bqClient.JobFromID(ctx, jobID)
	if err != nil {
		return false
	}

	return job.LastStatus().Done()
}
