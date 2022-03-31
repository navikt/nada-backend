package dpextracter

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

type DPExtracter struct {
	project  string
	bucket   string
	bqClient *bigquery.Client
}

func New(ctx context.Context, project, bucket string) (*DPExtracter, error) {
	bqClient, err := bigquery.NewClient(ctx, bucket)
	if err != nil {
		return nil, err
	}
	bqClient.Location = "europe-north1"

	return &DPExtracter{
		project:  project,
		bucket:   bucket,
		bqClient: bqClient,
	}, nil
}

func (d *DPExtracter) CreateExtractJob(ctx context.Context, bq *models.BigQuery, email string) (string, error) {
	object := fmt.Sprintf("%v.csv", bq.Table)
	gcsURI := fmt.Sprintf("gs://%v/%v.csv", d.bucket, object)

	client, err := bigquery.NewClient(ctx, bq.ProjectID)
	if err != nil {
		return "", err
	}
	defer client.Close()

	gcsRef := bigquery.NewGCSReference(gcsURI)
	gcsRef.FieldDelimiter = ";"

	extractor := client.DatasetInProject(bq.ProjectID, bq.Dataset).Table(bq.Table).ExtractorTo(gcsRef)
	extractor.Location = "europe-north1"

	job, err := extractor.Run(ctx)
	if err != nil {
		return "", err
	}

	return job.ID(), nil
}

func (d *DPExtracter) CreateSignedURL(ctx context.Context, object string) (string, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(15 * time.Minute),
	}

	url, err := client.Bucket(d.bucket).SignedURL(object, opts)
	if err != nil {
		return "", err
	}

	return url, nil
}
