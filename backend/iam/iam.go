package iam

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	BucketType   = "bucket"
	BigQueryType = "bigquery"
)

type Client struct {
	storageClient *storage.Client
}

func New(ctx context.Context) *Client {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Initializing storage client: %v", err)
	}

	return &Client{
		storageClient: storageClient,
	}
}

func (c *Client) CheckDatastoreAccess(ctx context.Context, datastore map[string]string, subject string) (bool, error) {
	datastoreType := datastore["type"]
	if len(datastoreType) == 0 {
		return false, fmt.Errorf("no type defined")
	}

	switch datastoreType {
	case BucketType:
		return c.checkAccessInBucket(ctx, datastore["bucket_id"], subject)
	case BigQueryType:
		return CheckAccessInBigQueryTable(ctx, datastore["project_id"], datastore["dataset_id"], datastore["resource_id"], subject)
	}

	return false, fmt.Errorf("unknown datastore type: %v", datastoreType)
}

func (c *Client) UpdateDatastoreAccess(ctx context.Context, datastore map[string]string, accessMap map[string]time.Time) error {
	datastoreType := datastore["type"]
	if len(datastoreType) == 0 {
		return fmt.Errorf("no type defined")
	}

	switch datastoreType {
	case BucketType:
		for subject, expiry := range accessMap {
			if err := c.UpdateBucketAccessControl(ctx, datastore["bucket_id"], subject, expiry); err != nil {
				return err
			}
			return nil
		}
	case BigQueryType:
		for subject := range accessMap {
			if err := UpdateBigqueryTableAccessControl(ctx, datastore["project_id"], datastore["dataset_id"], datastore["resource_id"], subject); err != nil {
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("unknown datastore type: %v", datastoreType)
}

func (c *Client) RemoveDatastoreAccess(ctx context.Context, datastore map[string]string, subject string) error {
	datastoreType := datastore["type"]
	if len(datastoreType) == 0 {
		return fmt.Errorf("no type defined")
	}

	switch datastoreType {
	case BucketType:
		return c.RemoveMemberFromBucket(ctx, datastore["bucket_id"], subject)
	case BigQueryType:
		return RemoveMemberFromBigQueryTable(ctx, datastore["project_id"], datastore["dataset_id"], datastore["resource_id"], subject)
	}

	return fmt.Errorf("unknown datastore type: %v", datastoreType)
}
