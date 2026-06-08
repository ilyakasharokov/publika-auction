package domain

import "time"

type Bid struct {
	ID        string    `bson:"_id"`
	AuctionID string    `bson:"auction_id"`
	LotID     string    `bson:"lot_id"`
	LotNum    int       `bson:"lot_num"`
	ClientID  string    `bson:"client_id"`
	Phone     string    `bson:"phone"`
	TgUserID  int64     `bson:"tg_user_id"`
	Amount    int       `bson:"amount"`
	Cancelled bool      `bson:"cancelled"`
	PlacedAt  time.Time `bson:"placed_at"`
}
