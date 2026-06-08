package cache

import (
	"sync"

	"publika-auction/internal/domain"
)

type ClientCache struct {
	mu      sync.RWMutex
	byPhone map[string]*domain.Client
	byTgID  map[int64]*domain.Client
}

func NewClientCache() *ClientCache {
	return &ClientCache{
		byPhone: make(map[string]*domain.Client),
		byTgID:  make(map[int64]*domain.Client),
	}
}

func (c *ClientCache) Set(cl *domain.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	copy := *cl
	c.byPhone[cl.Phone] = &copy
	if cl.TgUserID != 0 {
		c.byTgID[cl.TgUserID] = &copy
	}
}

func (c *ClientCache) GetByPhone(phone string) (*domain.Client, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cl, ok := c.byPhone[phone]
	if !ok {
		return nil, false
	}
	copy := *cl
	return &copy, true
}

func (c *ClientCache) GetByTgID(tgID int64) (*domain.Client, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cl, ok := c.byTgID[tgID]
	if !ok {
		return nil, false
	}
	copy := *cl
	return &copy, true
}

func (c *ClientCache) GetAllWithTgID() []*domain.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []*domain.Client
	for _, cl := range c.byPhone {
		if cl.TgUserID != 0 {
			copy := *cl
			out = append(out, &copy)
		}
	}
	return out
}

func (c *ClientCache) All() []*domain.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*domain.Client, 0, len(c.byPhone))
	for _, cl := range c.byPhone {
		copy := *cl
		out = append(out, &copy)
	}
	return out
}
