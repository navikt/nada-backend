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

func (m MockSlackClient) NewDataProdukt(dp *models.Dataproduct) error {
	m.log.Info("NewDataProdukt")
	return nil
}
