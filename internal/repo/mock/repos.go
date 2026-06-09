// Package mock provides in-memory repository implementations for tests.
package mock

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"

	"publika-auction/internal/domain"
)

// BidRepo ----------------------------------------------------------------

type BidRepo struct {
	mu   sync.Mutex
	bids map[string]*domain.Bid
}

func NewBidRepo() *BidRepo { return &BidRepo{bids: make(map[string]*domain.Bid)} }

func (r *BidRepo) Insert(_ context.Context, b *domain.Bid) error {
	r.mu.Lock(); defer r.mu.Unlock()
	r.bids[b.ID] = b; return nil
}
func (r *BidRepo) GetByID(_ context.Context, id string) (*domain.Bid, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	b, ok := r.bids[id]
	if !ok { return nil, mongo.ErrNoDocuments }
	return b, nil
}
func (r *BidRepo) ListByLot(_ context.Context, lotID string) ([]*domain.Bid, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	var out []*domain.Bid
	for _, b := range r.bids {
		if b.LotID == lotID && !b.Cancelled { out = append(out, b) }
	}
	return out, nil
}
func (r *BidRepo) ListByPhone(_ context.Context, phone string) ([]*domain.Bid, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	var out []*domain.Bid
	for _, b := range r.bids {
		if b.Phone == phone && !b.Cancelled { out = append(out, b) }
	}
	return out, nil
}
func (r *BidRepo) MarkCancelled(_ context.Context, id string) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if b, ok := r.bids[id]; ok { b.Cancelled = true }
	return nil
}
func (r *BidRepo) Count() int {
	r.mu.Lock(); defer r.mu.Unlock(); return len(r.bids)
}

// LotRepo ----------------------------------------------------------------

type LotRepo struct {
	mu   sync.Mutex
	lots map[string]*domain.Lot
}

func NewLotRepo() *LotRepo { return &LotRepo{lots: make(map[string]*domain.Lot)} }

func (r *LotRepo) Create(_ context.Context, l *domain.Lot) error {
	r.mu.Lock(); defer r.mu.Unlock(); r.lots[l.ID] = l; return nil
}
func (r *LotRepo) GetByID(_ context.Context, id string) (*domain.Lot, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	l, ok := r.lots[id]
	if !ok { return nil, mongo.ErrNoDocuments }
	return l, nil
}
func (r *LotRepo) ListByAuction(_ context.Context, auctionID string) ([]*domain.Lot, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	var out []*domain.Lot
	for _, l := range r.lots { if l.AuctionID == auctionID { out = append(out, l) } }
	return out, nil
}
func (r *LotRepo) Update(_ context.Context, l *domain.Lot) error {
	r.mu.Lock(); defer r.mu.Unlock(); r.lots[l.ID] = l; return nil
}
func (r *LotRepo) MarkSold(_ context.Context, lotID, bidID string, amount int) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if l, ok := r.lots[lotID]; ok { l.Status = domain.LotSold; l.SoldFor = amount; l.SoldBidID = bidID }
	return nil
}
func (r *LotRepo) IncrViewCount(_ context.Context, lotID string) error { return nil }

// AuctionRepo ------------------------------------------------------------

type AuctionRepo struct {
	mu       sync.Mutex
	auctions map[string]*domain.Auction
}

func NewAuctionRepo() *AuctionRepo { return &AuctionRepo{auctions: make(map[string]*domain.Auction)} }

func (r *AuctionRepo) Create(_ context.Context, a *domain.Auction) error {
	r.mu.Lock(); defer r.mu.Unlock(); r.auctions[a.ID] = a; return nil
}
func (r *AuctionRepo) GetByID(_ context.Context, id string) (*domain.Auction, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	a, ok := r.auctions[id]
	if !ok { return nil, mongo.ErrNoDocuments }
	return a, nil
}
func (r *AuctionRepo) GetBySlug(_ context.Context, slug string) (*domain.Auction, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	for _, a := range r.auctions { if a.Slug == slug { return a, nil } }
	return nil, mongo.ErrNoDocuments
}
func (r *AuctionRepo) List(_ context.Context) ([]*domain.Auction, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	out := make([]*domain.Auction, 0, len(r.auctions))
	for _, a := range r.auctions { out = append(out, a) }
	return out, nil
}
func (r *AuctionRepo) UpdateStatus(_ context.Context, id string, status domain.AuctionStatus) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if a, ok := r.auctions[id]; ok { a.Status = status }
	return nil
}
func (r *AuctionRepo) Update(_ context.Context, a *domain.Auction) error {
	r.mu.Lock(); defer r.mu.Unlock(); r.auctions[a.ID] = a; return nil
}
