package access

import "context"

type Access interface {
	Grant(ctx context.Context, projectID, datasetID, tableID, member string) error
	Revoke(ctx context.Context, projectID, datasetID, tableID, member string) error
}
