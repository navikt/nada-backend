package event

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type (
	DatasetListenerGrantAccess           func(ctx context.Context, dpID uuid.UUID, subject string)
	DatasetListenerRevokeAccess          func(ctx context.Context, dpID uuid.UUID, subject string)
	DatasetListenerAddMetabaseMapping    func(ctx context.Context, dpID uuid.UUID)
	DatasetListenerRemoveMetabaseMapping func(ctx context.Context, dpID uuid.UUID)
	DatasetListenerDelete                func(ctx context.Context, dpID uuid.UUID)
)

type Manager struct {
	lock                                          sync.RWMutex
	datasetGrantAccessListeners                   []DatasetListenerGrantAccess
	datasetRevokeAccessListeners                  []DatasetListenerRevokeAccess
	datasetListenerAddMetabaseMappingListeners    []DatasetListenerAddMetabaseMapping
	datasetListenerRemoveMetabaseMappingListeners []DatasetListenerRemoveMetabaseMapping
	datasetDeleteListeners                        []DatasetListenerDelete
}

func (m *Manager) ListenForDatasetGrant(fn DatasetListenerGrantAccess) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.datasetGrantAccessListeners = append(m.datasetGrantAccessListeners, fn)
}

func (m *Manager) TriggerDatasetGrant(ctx context.Context, dpID uuid.UUID, subject string) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.datasetGrantAccessListeners {
		fn(ctx, dpID, subject)
	}
}

func (m *Manager) ListenForDatasetRevoke(fn DatasetListenerRevokeAccess) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.datasetRevokeAccessListeners = append(m.datasetRevokeAccessListeners, fn)
}

func (m *Manager) TriggerDatasetRevoke(ctx context.Context, dpID uuid.UUID, subject string) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.datasetRevokeAccessListeners {
		fn(ctx, dpID, subject)
	}
}

func (m *Manager) ListenForDatasetAddMetabaseMapping(fn DatasetListenerAddMetabaseMapping) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.datasetListenerAddMetabaseMappingListeners = append(m.datasetListenerAddMetabaseMappingListeners, fn)
}

func (m *Manager) TriggerDatasetAddMetabaseMapping(ctx context.Context, dpID uuid.UUID) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.datasetListenerAddMetabaseMappingListeners {
		fn(ctx, dpID)
	}
}

func (m *Manager) ListenForDatasetRemoveMetabaseMapping(fn DatasetListenerRemoveMetabaseMapping) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.datasetListenerRemoveMetabaseMappingListeners = append(m.datasetListenerRemoveMetabaseMappingListeners, fn)
}

func (m *Manager) TriggerDatasetRemoveMetabaseMapping(ctx context.Context, dpID uuid.UUID) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.datasetListenerRemoveMetabaseMappingListeners {
		fn(ctx, dpID)
	}
}

func (m *Manager) ListenForDatasetDelete(fn DatasetListenerDelete) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.datasetDeleteListeners = append(m.datasetDeleteListeners, fn)
}

func (m *Manager) TriggerDatasetDelete(ctx context.Context, dpID uuid.UUID) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.datasetDeleteListeners {
		fn(ctx, dpID)
	}
}
