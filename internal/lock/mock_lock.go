package lock

import (
	"context"
	"sync"
	"time"
)

// MutexLock is a per-key in-process lock for tests.
type MutexLock struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewMutexLock() *MutexLock {
	return &MutexLock{locks: make(map[string]*sync.Mutex)}
}

func (l *MutexLock) keyMu(key string) *sync.Mutex {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.locks[key]; !ok {
		l.locks[key] = &sync.Mutex{}
	}
	return l.locks[key]
}

func (l *MutexLock) Acquire(_ context.Context, key string, _ time.Duration) (string, bool, error) {
	l.keyMu(key).Lock()
	return "mock-token", true, nil
}

func (l *MutexLock) Release(_ context.Context, key, _ string) error {
	l.keyMu(key).Unlock()
	return nil
}
