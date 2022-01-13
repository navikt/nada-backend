package event

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type (
	DataproductListenerGrantAccessAllUsers   func(context.Context, uuid.UUID) error
	DataproductListenerRevokeAccessAllUsers  func(context.Context, uuid.UUID) error
	DataproductListenerGrantAccess           func(context.Context, uuid.UUID, string) error
	DataproductListenerRevokeAccess          func(context.Context, uuid.UUID, string) error
	DataproductListenerAddMetabaseMapping    func(context.Context, uuid.UUID) error
	DataproductListenerRemoveMetabaseMapping func(context.Context, uuid.UUID) error
)

type Manager struct {
	lock                                              sync.RWMutex
	dataproductGrantAllUsersAccessListeners           []DataproductListenerGrantAccessAllUsers
	dataproductRevokeAllUsersAccessListeners          []DataproductListenerRevokeAccessAllUsers
	dataproductGrantAccessListeners                   []DataproductListenerGrantAccess
	dataproductRevokeAccessListeners                  []DataproductListenerRevokeAccess
	dataproductListenerAddMetabaseMappingListeners    []DataproductListenerAddMetabaseMapping
	dataproductListenerRemoveMetabaseMappingListeners []DataproductListenerRemoveMetabaseMapping
}

func (m *Manager) ListenForDataproductGrantAllUsers(fn DataproductListenerGrantAccessAllUsers) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductGrantAllUsersAccessListeners = append(m.dataproductGrantAllUsersAccessListeners, fn)
}

func (m *Manager) TriggerDataproductGrantAllUsers(ctx context.Context, dpID uuid.UUID) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductGrantAllUsersAccessListeners {
		if err := fn(ctx, dpID); err != nil {
			fmt.Println(err)
		}
	}
}

func (m *Manager) ListenForDataproductRevokeAllUsers(fn DataproductListenerRevokeAccessAllUsers) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.dataproductRevokeAllUsersAccessListeners = append(m.dataproductRevokeAllUsersAccessListeners, fn)
}

func (m *Manager) TriggerDataproductRevokeAllUsers(ctx context.Context, dpID uuid.UUID) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, fn := range m.dataproductRevokeAllUsersAccessListeners {
		if err := fn(ctx, dpID); err != nil {
			fmt.Println(err)
		}
	}
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
		if err := fn(ctx, dpID, subject); err != nil {
			fmt.Println(err)
		}
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
		if err := fn(ctx, dpID, subject); err != nil {
			fmt.Println(err)
		}
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
		if err := fn(ctx, dpID); err != nil {
			fmt.Println(err)
		}
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
		if err := fn(ctx, dpID); err != nil {
			fmt.Println(err)
		}
	}
}
