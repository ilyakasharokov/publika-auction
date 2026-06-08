package mongorepo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"publika-auction/internal/domain"
)

type BidRepo struct {
	col *mongo.Collection
}

func NewBidRepo(db *mongo.Database) *BidRepo {
	r := &BidRepo{col: db.Collection("bids")}
	r.ensureIndexes()
	return r
}

func (r *BidRepo) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "lot_id", Value: 1}, {Key: "placed_at", Value: -1}}},
		{Keys: bson.D{{Key: "phone", Value: 1}}},
		{Keys: bson.D{{Key: "auction_id", Value: 1}}},
	})
}

func (r *BidRepo) Insert(ctx context.Context, bid *domain.Bid) error {
	_, err := r.col.InsertOne(ctx, bid)
	return err
}

func (r *BidRepo) GetByID(ctx context.Context, id string) (*domain.Bid, error) {
	var bid domain.Bid
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&bid)
	if err != nil {
		return nil, err
	}
	return &bid, nil
}

func (r *BidRepo) ListByLot(ctx context.Context, lotID string) ([]*domain.Bid, error) {
	opts := options.Find().SetSort(bson.D{{Key: "placed_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{"lot_id": lotID, "cancelled": false}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var list []*domain.Bid
	for cursor.Next(ctx) {
		var b domain.Bid
		if err := cursor.Decode(&b); err != nil {
			return list, err
		}
		list = append(list, &b)
	}
	return list, cursor.Err()
}

func (r *BidRepo) ListByPhone(ctx context.Context, phone string) ([]*domain.Bid, error) {
	opts := options.Find().SetSort(bson.D{{Key: "placed_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{"phone": phone, "cancelled": false}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var list []*domain.Bid
	for cursor.Next(ctx) {
		var b domain.Bid
		if err := cursor.Decode(&b); err != nil {
			return list, err
		}
		list = append(list, &b)
	}
	return list, cursor.Err()
}

func (r *BidRepo) MarkCancelled(ctx context.Context, id string) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"cancelled":  true,
		"updated_at": time.Now(),
	}})
	return err
}
