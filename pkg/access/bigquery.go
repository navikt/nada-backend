package access

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/iam"
)

type Bigquery struct{}

func New() *Bigquery {
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
