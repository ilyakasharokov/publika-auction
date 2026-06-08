package models

import "time"

// Client представляет участника аукциона
type Client struct {
	TgID       int64     `bson:"tg_id" json:"tg_id"`
	TgUsername string    `bson:"tg_username" json:"tg_username"`
	FirstName  string    `bson:"first_name" json:"first_name"`
	LastName   string    `bson:"last_name" json:"last_name"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
}

// Bid представляет ставку на лот
type Bid struct {
	ID        string    `bson:"_id" json:"id"`
	LotID     int       `bson:"lot_id" json:"lot_id"`
	Client    *Client   `bson:"client" json:"client"`
	Amount    int64     `bson:"amount" json:"amount"`
	Status    string    `bson:"status" json:"status"` // confirmed, pending, rejected
	Date      time.Time `bson:"date" json:"date"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// Lot представляет лот аукциона
type Lot struct {
	ID           int       `bson:"_id" json:"id"`
	Title        string    `bson:"title" json:"title"`
	Description  string    `bson:"description" json:"description"`
	Bids         []*Bid    `bson:"bids" json:"bids"`
	MaxBid       *Bid      `bson:"max_bid" json:"max_bid"`
	MaxConfirmed int64     `bson:"max_confirmed" json:"max_confirmed"`
	SoldFor      int64     `bson:"sold_for" json:"sold_for"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
}

// GetBidsCount возвращает количество ставок на лот
func (l *Lot) GetBidsCount() int {
	if l.Bids == nil {
		return 0
	}
	return len(l.Bids)
}

// Chat представляет чат с участником
type Chat struct {
	ID        string    `bson:"_id" json:"id"`
	ClientID  int64     `bson:"client_id" json:"client_id"`
	Client    *Client   `bson:"client" json:"client"`
	Messages  []string  `bson:"messages" json:"messages"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// AuctionState представляет состояние аукциона
type AuctionState struct {
	ID        string     `bson:"_id" json:"id"`
	IsRunning bool       `bson:"is_running" json:"is_running"`
	StartedAt *time.Time `bson:"started_at" json:"started_at,omitempty"`
	EndedAt   *time.Time `bson:"ended_at" json:"ended_at,omitempty"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
}
