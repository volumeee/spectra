package webhook

import (
	"context"
	"sync"

	"github.com/spectra-browser/spectra/internal/domain"
)

type MemoryStore struct {
	subs map[string]domain.WebhookSubscription
	mu   sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{subs: make(map[string]domain.WebhookSubscription)}
}

func (s *MemoryStore) CreateWebhook(_ context.Context, sub *domain.WebhookSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[sub.ID] = *sub
	return nil
}

func (s *MemoryStore) DeleteWebhook(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs, id)
	return nil
}

func (s *MemoryStore) ListWebhooks(_ context.Context) ([]domain.WebhookSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]domain.WebhookSubscription, 0, len(s.subs))
	for _, sub := range s.subs {
		list = append(list, sub)
	}
	return list, nil
}

func (s *MemoryStore) GetByEvent(_ context.Context, event domain.WebhookEvent) ([]domain.WebhookSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []domain.WebhookSubscription
	for _, sub := range s.subs {
		if sub.Event == event && sub.Active {
			list = append(list, sub)
		}
	}
	return list, nil
}
