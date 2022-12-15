package slack

import (
	"github.com/navikt/nada-backend/pkg/graph/models"
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

func (m MockSlackClient) NewAccessRequest(contact string, dp *models.Dataproduct, ds *models.Dataset, ar *models.AccessRequest) error {
	m.log.Info("New AccessRequest send to " + contact)
	return nil
}

func (s MockSlackClient) IsValidSlackChannel(name string) (bool, error) {
	return true, nil
}
