package slack

import (
	"github.com/sirupsen/logrus"
)

type MockSlackClient struct {
	log *logrus.Logger
}

func NewMockSlackClient(log *logrus.Logger) *MockSlackClient {
	return &MockSlackClient{
		log: log,
	}
}

func (s MockSlackClient) IsValidSlackChannel(name string) (bool, error) {
	return true, nil
}
