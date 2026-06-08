package domain

import "time"

type AuctionStatus string

const (
	AuctionDraft  AuctionStatus = "draft"
	AuctionActive AuctionStatus = "active"
	AuctionEnded  AuctionStatus = "ended"
)

type Auction struct {
	ID          string        `bson:"_id"`
	Slug        string        `bson:"slug"`
	Title       string        `bson:"title"`
	Description string        `bson:"description"`
	Status      AuctionStatus `bson:"status"`
	BidStep     int           `bson:"bid_step"`
	StartAt     time.Time     `bson:"start_at"`
	EndAt       time.Time     `bson:"end_at"`
	CreatedAt   time.Time     `bson:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at"`
}
