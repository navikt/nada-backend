package amplitude

import "context"

type MockAmplitudeClient struct{}

func NewMock() *MockAmplitudeClient {
	return &MockAmplitudeClient{}
}

func (am *MockAmplitudeClient) PublishEvent(ctx context.Context, title string) error {
	return nil
}
