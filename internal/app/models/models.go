package models

import (
	"publika-auction/internal/app/clients-repo"
	"time"
)

var ISOLayout = "2006-01-02T15:04:05"

type DBTime struct {
	D time.Time
}

type Item struct {
	Id           int
	Bids         []Bid
	MaxConfirmed int
	MaxBid       Bid
	Photo        string
	Description  string
	SoldFor      int
	ViewCount    int
}

type Bid struct {
	Id           int
	ItemId       int
	ClientsPhone string
	Client       *clients_repo.Client
	Date         time.Time
	Summ         int
	Confirmed    bool
}

func (it Item) GetBidsCount() int {
	if it.Bids == nil {
		return 0
	}
	return len(it.Bids)
}
