package mock

import (
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/stretchr/testify/mock"
)

var _ database.Transacter = &MockTransacter{}

type MockTransacter struct {
	mock.Mock
}

func (m *MockTransacter) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTransacter) Rollback() error {
	args := m.Called()
	return args.Error(0)
}
