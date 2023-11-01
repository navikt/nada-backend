package access

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/iam"
)

type Bigquery struct{}

func NewBigquery() *Bigquery {
	return &Bigquery{}
}

func (b Bigquery) Grant(ctx context.Context, projectID, datasetID, tableID, member string) error {
	bqClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer bqClient.Close()

	policy, err := getPolicy(ctx, bqClient, datasetID, tableID)
	if err != nil {
		return fmt.Errorf("getting policy for %v.%v in %v: %v", datasetID, tableID, projectID, err)
	}

	var entityType bigquery.EntityType
	switch strings.Split(member, ":")[0] {
	case "user", "serviceAccount":
		entityType = bigquery.UserEmailEntity
	case "group":
		entityType = bigquery.GroupEmailEntity
	}
	newEntry := &bigquery.AccessEntry{
		EntityType: entityType,
		Entity:     strings.Split(member, ":")[1],
		Role:       bigquery.AccessRole("roles/bigquery.metadataViewer"),
	}
	ds := bqClient.Dataset(datasetID)
	m, err := ds.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("ds.Metadata: %w", err)
	}

	update := bigquery.DatasetMetadataToUpdate{
		Access: append(m.Access, newEntry),
	}
	if _, err := ds.Update(ctx, update, m.ETag); err != nil {
		return fmt.Errorf("ds.Update: %w", err)
	}

	// no support for V3 for BigQuery yet, and no support for conditions
	role := "roles/bigquery.dataViewer"
	policy.Add(member, iam.RoleName(role))

	bqTable := bqClient.Dataset(datasetID).Table(tableID)

	return bqTable.IAM().SetPolicy(ctx, policy)
}

func (b Bigquery) Revoke(ctx context.Context, projectID, datasetID, tableID, member string) error {
	bqClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer bqClient.Close()

	policy, err := getPolicy(ctx, bqClient, datasetID, tableID)
	if err != nil {
		return fmt.Errorf("getting policy for %v.%v in %v: %v", datasetID, tableID, projectID, err)
	}

	// no support for V3 for BigQuery yet, and no support for conditions
	role := "roles/bigquery.dataViewer"
	policy.Remove(member, iam.RoleName(role))

	bqTable := bqClient.Dataset(datasetID).Table(tableID)
	return bqTable.IAM().SetPolicy(ctx, policy)
}

func (b Bigquery) AddToAuthorizedViews(ctx context.Context, srcProjectID, srcDataset, sinkProjectID, sinkDataset, sinkTable string) error {
	bqClient, err := bigquery.NewClient(ctx, srcProjectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer bqClient.Close()
	ds := bqClient.Dataset(srcDataset)
	m, err := ds.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("ds.Metadata: %w", err)
	}

	if m.Access != nil {
		for _, e := range m.Access {
			if e != nil && e.EntityType == bigquery.ViewEntity && e.View != nil &&
				e.View.ProjectID == sinkProjectID && e.View.DatasetID == sinkDataset && e.View.TableID == sinkTable {
				return nil
			}
		}
	}

	newEntry := &bigquery.AccessEntry{
		EntityType: bigquery.ViewEntity,
		View: &bigquery.Table{
			ProjectID: sinkProjectID,
			DatasetID: sinkDataset,
			TableID:   sinkTable,
		},
	}

	update := bigquery.DatasetMetadataToUpdate{
		Access: append(m.Access, newEntry),
	}
	if _, err := ds.Update(ctx, update, m.ETag); err != nil {
		return err
	}

	return nil
}

func getPolicy(ctx context.Context, bqclient *bigquery.Client, datasetID, tableID string) (*iam.Policy, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	dataset := bqclient.Dataset(datasetID)
	table := dataset.Table(tableID)
	policy, err := table.IAM().Policy(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting policy for %v.%v: %v", datasetID, tableID, err)
	}

	return policy, nil
}

func (b Bigquery) HasAccess(ctx context.Context, projectID, datasetID, tableID, member string) (bool, error) {
	bqClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return false, fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer bqClient.Close()

	policy, err := getPolicy(ctx, bqClient, datasetID, tableID)
	if err != nil {
		return false, fmt.Errorf("getting policy for %v.%v in %v: %v", datasetID, tableID, projectID, err)
	}

	// no support for V3 for BigQuery yet, and no support for conditions
	role := "roles/bigquery.dataViewer"
	return policy.HasRole(member, iam.RoleName(role)), nil
}
