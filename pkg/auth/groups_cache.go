package auth

import (
	"sync"
	"time"
)

type groupsCacheValue struct {
	GoogleGroups  Groups
	AzureGroups   Groups
	GoogleExpires time.Time
	AzureExpires  time.Time
}

type groupsCacher struct {
	lock  sync.RWMutex
	cache map[string]groupsCacheValue
}

func (g *groupsCacher) GetGoogleGroups(email string) (Groups, bool) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	v, ok := g.cache[email]
	if !ok {
		return nil, false
	}
	if v.GoogleExpires.Before(time.Now()) {
		return nil, false
	}
	return v.GoogleGroups, true
}

func (g *groupsCacher) SetGoogleGroups(email string, groups Groups) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if userCache, ok := g.cache[email]; ok {
		userCache.GoogleGroups = groups
		userCache.GoogleExpires = time.Now().Add(1 * time.Hour)
		g.cache[email] = userCache

		return
	}

	// User not in cache
	g.cache[email] = groupsCacheValue{
		GoogleGroups:  groups,
		GoogleExpires: time.Now().Add(1 * time.Hour),
	}
}

func (g *groupsCacher) GetAzureGroups(email string) (Groups, bool) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	v, ok := g.cache[email]
	if !ok {
		return nil, false
	}
	if v.AzureExpires.Before(time.Now()) {
		return nil, false
	}
	return v.AzureGroups, true
}

func (g *groupsCacher) SetAzureGroups(email string, groups Groups) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if userCache, ok := g.cache[email]; ok {
		userCache.AzureGroups = groups
		userCache.AzureExpires = time.Now().Add(1 * time.Hour)
		g.cache[email] = userCache

		return
	}

	// User not in cache
	g.cache[email] = groupsCacheValue{
		AzureGroups:  groups,
		AzureExpires: time.Now().Add(1 * time.Hour),
	}
}
