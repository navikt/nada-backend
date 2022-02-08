package event

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type (
	DataproductListenerGrantAccess           func(ctx context.Context, dpID uuid.UUID, subject string)
	DataproductListenerRevokeAccess          func(ctx context.Context, dpID uuid.UUID, subject string)
	DataproductListenerAddMetabaseMapping    func(ctx context.Context, dpID uuid.UUID)
	DataproductListenerRemoveMetabaseMapping func(ctx context.Context, dpID uuid.UUID)
	DataproductListenerDelete                func(ctx context.Context, dpID uuid.UUID)
)

type Manager struct {
	lock                                              sync.RWMutex
	dataproductGrantAccessListeners                   []DataproductListenerGrantAccess
	dataproductRevokeAccessListeners                  []DataproductListenerRevokeAccess
	dataproductListenerAddMetabaseMappingListeners    []DataproductListenerAddMetabaseMapping
	dataproductListenerRemoveMetabaseMappingListeners []DataproductListenerRemoveMetabaseMapping
	dataproductDeleteListeners                        []DataproductListenerDelete
}

func (m *Manager) ListenForDataproductGrant(fn DataproductListenerGrantAccess) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductGrantAccessListeners = append(m.dataproductGrantAccessListeners, fn)
}

func (m *Manager) TriggerDataproductGrant(ctx context.Context, dpID uuid.UUID, subject string) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductGrantAccessListeners {
		fn(ctx, dpID, subject)
	}
}

func (m *Manager) ListenForDataproductRevoke(fn DataproductListenerRevokeAccess) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductRevokeAccessListeners = append(m.dataproductRevokeAccessListeners, fn)
}

func (m *Manager) TriggerDataproductRevoke(ctx context.Context, dpID uuid.UUID, subject string) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductRevokeAccessListeners {
		fn(ctx, dpID, subject)
	}
}

func (m *Manager) ListenForDataproductAddMetabaseMapping(fn DataproductListenerAddMetabaseMapping) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductListenerAddMetabaseMappingListeners = append(m.dataproductListenerAddMetabaseMappingListeners, fn)
}

func (m *Manager) TriggerDataproductAddMetabaseMapping(ctx context.Context, dpID uuid.UUID) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductListenerAddMetabaseMappingListeners {
		fn(ctx, dpID)
	}
}

func (m *Manager) ListenForDataproductRemoveMetabaseMapping(fn DataproductListenerRemoveMetabaseMapping) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductListenerRemoveMetabaseMappingListeners = append(m.dataproductListenerRemoveMetabaseMappingListeners, fn)
}

func (m *Manager) TriggerDataproductRemoveMetabaseMapping(ctx context.Context, dpID uuid.UUID) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductListenerRemoveMetabaseMappingListeners {
		fn(ctx, dpID)
	}
}

func (m *Manager) ListenForDataproductDelete(fn DataproductListenerDelete) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductDeleteListeners = append(m.dataproductDeleteListeners, fn)
}

func (m *Manager) TriggerDataproductDelete(ctx context.Context, dpID uuid.UUID) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductDeleteListeners {
		fn(ctx, dpID)
	}
}
