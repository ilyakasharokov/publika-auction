package mongorepo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"publika-auction/internal/domain"
)

type AuctionRepo struct {
	col *mongo.Collection
}

func NewAuctionRepo(db *mongo.Database) *AuctionRepo {
	r := &AuctionRepo{col: db.Collection("auctions")}
	r.ensureIndexes()
	return r
}

func (r *AuctionRepo) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
}

func (r *AuctionRepo) Create(ctx context.Context, a *domain.Auction) error {
	_, err := r.col.InsertOne(ctx, a)
	return err
}

func (r *AuctionRepo) GetByID(ctx context.Context, id string) (*domain.Auction, error) {
	var a domain.Auction
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&a)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AuctionRepo) GetBySlug(ctx context.Context, slug string) (*domain.Auction, error) {
	var a domain.Auction
	err := r.col.FindOne(ctx, bson.M{"slug": slug}).Decode(&a)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AuctionRepo) List(ctx context.Context) ([]*domain.Auction, error) {
	cursor, err := r.col.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var list []*domain.Auction
	for cursor.Next(ctx) {
		var a domain.Auction
		if err := cursor.Decode(&a); err != nil {
			return list, err
		}
		list = append(list, &a)
	}
	return list, cursor.Err()
}

func (r *AuctionRepo) UpdateStatus(ctx context.Context, id string, status domain.AuctionStatus) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"status":     status,
		"updated_at": time.Now(),
	}})
	return err
}

func (r *AuctionRepo) Update(ctx context.Context, a *domain.Auction) error {
	a.UpdatedAt = time.Now()
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": a.ID}, a)
	return err
}
