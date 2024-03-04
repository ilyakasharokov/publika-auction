package mng

import (
	"context"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"publika-auction/internal/app/models"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MngSrv struct {
	client *mongo.Client
	db     *mongo.Database
}

func New() (*MngSrv, error) {
	ms := &MngSrv{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, err
	}
	database := client.Database("auction")
	ms.client = client
	ms.db = database
	return ms, nil
}

func (ms *MngSrv) InsertBid(bid models.Bid) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	res, err := ms.db.Collection("bids").InsertOne(ctx, bid)
	if err != nil {
		log.Err(err).Interface("bid", bid).Msg("insert bid error")
		return
	}
	log.Info().Interface("res", res).Msg("insert bid success")
}

func (ms *MngSrv) GetBids() []models.Bid {
	filter := bson.D{}
	items, err := ms.db.Collection("bids").Find(context.Background(), filter)
	if err != nil {
		log.Err(err).Msg("GetBids error")
		return nil
	}
	bids := make([]models.Bid, 0)
	for items.Next(context.Background()) {
		var result models.Bid
		err := items.Decode(&result)
		if err != nil {
			log.Err(err).Msg("GetBids next error")
			return bids
		}
		bids = append(bids, result)
	}
	return bids
}

func (ms *MngSrv) GetBidsByPhone(phone string) []models.Bid {
	filter := bson.D{{"clientsphone", phone}}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{"date", -1}})
	items, err := ms.db.Collection("bids").Find(context.Background(), filter, findOptions)
	if err != nil {
		log.Err(err).Msg("GetBids by phone error")
		return nil
	}
	bids := make([]models.Bid, 0)
	for items.Next(context.Background()) {
		var result models.Bid
		err := items.Decode(&result)
		if err != nil {
			log.Err(err).Msg("GetBids by phone next error")
			return bids
		}
		bids = append(bids, result)
	}
	return bids
}
