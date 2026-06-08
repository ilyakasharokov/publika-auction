package repo

import (
	"context"

	"publika-auction/internal/domain"
)

type AuctionRepo interface {
	Create(ctx context.Context, a *domain.Auction) error
	GetByID(ctx context.Context, id string) (*domain.Auction, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Auction, error)
	List(ctx context.Context) ([]*domain.Auction, error)
	UpdateStatus(ctx context.Context, id string, status domain.AuctionStatus) error
	Update(ctx context.Context, a *domain.Auction) error
}

type LotRepo interface {
	Create(ctx context.Context, lot *domain.Lot) error
	GetByID(ctx context.Context, id string) (*domain.Lot, error)
	ListByAuction(ctx context.Context, auctionID string) ([]*domain.Lot, error)
	Update(ctx context.Context, lot *domain.Lot) error
	MarkSold(ctx context.Context, lotID, bidID string, amount int) error
	IncrViewCount(ctx context.Context, lotID string) error
}

type BidRepo interface {
	Insert(ctx context.Context, bid *domain.Bid) error
	GetByID(ctx context.Context, id string) (*domain.Bid, error)
	ListByLot(ctx context.Context, lotID string) ([]*domain.Bid, error)
	ListByPhone(ctx context.Context, phone string) ([]*domain.Bid, error)
	MarkCancelled(ctx context.Context, id string) error
}

type ClientRepo interface {
	Upsert(ctx context.Context, c *domain.Client) error
	GetByPhone(ctx context.Context, phone string) (*domain.Client, error)
	GetByTgID(ctx context.Context, tgID int64) (*domain.Client, error)
	List(ctx context.Context) ([]*domain.Client, error)
	Block(ctx context.Context, phone string) error
}

type MessageRepo interface {
	Insert(ctx context.Context, msg *domain.ChatMessage) error
	ListByTgID(ctx context.Context, tgID int64) ([]*domain.ChatMessage, error)
}
