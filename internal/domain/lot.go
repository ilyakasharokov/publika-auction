package domain

import "time"

type LotStatus string

const (
	LotActive LotStatus = "active"
	LotSold   LotStatus = "sold"
	LotPulled LotStatus = "pulled"
)

type Lot struct {
	ID          string    `bson:"_id"`
	AuctionID   string    `bson:"auction_id"`
	Num         int       `bson:"num"`
	Title       string    `bson:"title"`
	Description string    `bson:"description"`
	PhotoURL    string    `bson:"photo_url"`
	StartPrice  int       `bson:"start_price"`
	Status      LotStatus `bson:"status"`
	SoldFor     int       `bson:"sold_for,omitempty"`
	SoldBidID   string    `bson:"sold_bid_id,omitempty"`
	ViewCount   int64     `bson:"view_count"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}
