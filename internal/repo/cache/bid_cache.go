package cache

import (
	"fmt"
	"sync"
)

type LotState struct {
	MaxAmount   int
	MaxBidID    string
	MaxClientID string
	MaxTgID     int64
	BidCount    int
	StartPrice  int
	BidStep     int
	LotID       string
	LotNum      int
	AuctionSlug string
}

type BidCache struct {
	mu     sync.RWMutex
	states map[string]LotState
}

func NewBidCache() *BidCache {
	return &BidCache{states: make(map[string]LotState)}
}

func (c *BidCache) Key(auctionID, lotID string) string {
	return fmt.Sprintf("%s:%s", auctionID, lotID)
}

func (c *BidCache) Get(auctionID, lotID string) (LotState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.states[c.Key(auctionID, lotID)]
	return s, ok
}

func (c *BidCache) Set(auctionID, lotID string, s LotState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.states[c.Key(auctionID, lotID)] = s
}

func (c *BidCache) GetAll(auctionID string) []LotState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []LotState
	prefix := auctionID + ":"
	for k, v := range c.states {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, v)
		}
	}
	return out
}

func (c *BidCache) Delete(auctionID, lotID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.states, c.Key(auctionID, lotID))
}
