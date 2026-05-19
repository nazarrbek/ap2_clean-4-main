package idempotency

import (
	"context"
	"sync"
	"time"
)

type MemoryIdempotencyStore struct {
	mu    sync.RWMutex
	items map[string]time.Time
	ttl   time.Duration
}

func NewMemoryIdempotencyStore(ttl time.Duration) *MemoryIdempotencyStore {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	store := &MemoryIdempotencyStore{
		items: make(map[string]time.Time),
		ttl:   ttl,
	}

	go store.cleanup()

	return store
}

func (s *MemoryIdempotencyStore) IsProcessed(ctx context.Context, paymentID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	expiry, exists := s.items[paymentID]
	if !exists {
		return false, nil
	}

	if time.Now().After(expiry) {
		return false, nil
	}

	return true, nil
}

func (s *MemoryIdempotencyStore) MarkProcessed(ctx context.Context, paymentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[paymentID] = time.Now().Add(s.ttl)
	return nil
}

func (s *MemoryIdempotencyStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, expiry := range s.items {
			if now.After(expiry) {
				delete(s.items, key)
			}
		}
		s.mu.Unlock()
	}
}
