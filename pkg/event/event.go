package event

import (
	"sync"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

type DataproductListener func(*models.Dataproduct)

type Manager struct {
	lock                       sync.RWMutex
	dataproductCreateListeners []DataproductListener
	dataproductUpdateListeners []DataproductListener
	dataproductDeleteListeners []DataproductListener
}

func (m *Manager) ListenForDataproductCreate(fn DataproductListener) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductCreateListeners = append(m.dataproductCreateListeners, fn)
}

func (m *Manager) TriggerDataproductCreate(dp *models.Dataproduct) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductCreateListeners {
		fn(dp)
	}
}

func (m *Manager) ListenForDataproductUpdate(fn DataproductListener) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductUpdateListeners = append(m.dataproductUpdateListeners, fn)
}

func (m *Manager) TriggerDataproductUpdate(dp *models.Dataproduct) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductUpdateListeners {
		fn(dp)
	}
}

func (m *Manager) ListenForDataproductDelete(fn DataproductListener) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductDeleteListeners = append(m.dataproductDeleteListeners, fn)
}

func (m *Manager) TriggerDataproductDelete(dp *models.Dataproduct) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductDeleteListeners {
		fn(dp)
	}
}
