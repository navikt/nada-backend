package access

import "context"

type AccessMock struct{}

func NewMock() *AccessMock {
	return &AccessMock{}
}

func (a AccessMock) Grant(ctx context.Context, projectID, datasetID, tableID, member string) error {
	return nil
}

func (a AccessMock) Revoke(ctx context.Context, projectID, datasetID, tableID, member string) error {
	return nil
}
