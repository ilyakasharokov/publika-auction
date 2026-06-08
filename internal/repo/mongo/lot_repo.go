package mongorepo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"publika-auction/internal/domain"
)

type LotRepo struct {
	col *mongo.Collection
}

func NewLotRepo(db *mongo.Database) *LotRepo {
	r := &LotRepo{col: db.Collection("lots")}
	r.ensureIndexes()
	return r
}

func (r *LotRepo) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "auction_id", Value: 1}, {Key: "num", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
}

func (r *LotRepo) Create(ctx context.Context, lot *domain.Lot) error {
	_, err := r.col.InsertOne(ctx, lot)
	return err
}

func (r *LotRepo) GetByID(ctx context.Context, id string) (*domain.Lot, error) {
	var lot domain.Lot
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&lot)
	if err != nil {
		return nil, err
	}
	return &lot, nil
}

func (r *LotRepo) ListByAuction(ctx context.Context, auctionID string) ([]*domain.Lot, error) {
	cursor, err := r.col.Find(ctx, bson.M{"auction_id": auctionID}, options.Find().SetSort(bson.D{{Key: "num", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var list []*domain.Lot
	for cursor.Next(ctx) {
		var lot domain.Lot
		if err := cursor.Decode(&lot); err != nil {
			return list, err
		}
		list = append(list, &lot)
	}
	return list, cursor.Err()
}

func (r *LotRepo) Update(ctx context.Context, lot *domain.Lot) error {
	lot.UpdatedAt = time.Now()
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": lot.ID}, lot)
	return err
}

func (r *LotRepo) MarkSold(ctx context.Context, lotID, bidID string, amount int) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": lotID}, bson.M{"$set": bson.M{
		"status":      domain.LotSold,
		"sold_for":    amount,
		"sold_bid_id": bidID,
		"updated_at":  time.Now(),
	}})
	return err
}

func (r *LotRepo) IncrViewCount(ctx context.Context, lotID string) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": lotID}, bson.M{"$inc": bson.M{"view_count": 1}})
	return err
}
