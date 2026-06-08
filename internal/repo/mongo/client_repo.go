package mongorepo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"publika-auction/internal/domain"
)

type ClientRepo struct {
	col    *mongo.Collection
	msgCol *mongo.Collection
}

func NewClientRepo(db *mongo.Database) *ClientRepo {
	r := &ClientRepo{
		col:    db.Collection("clients"),
		msgCol: db.Collection("chat_messages"),
	}
	r.ensureIndexes()
	return r
}

func (r *ClientRepo) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "phone", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "tg_user_id", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
	})
	r.msgCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "tg_user_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
}

func (r *ClientRepo) Upsert(ctx context.Context, c *domain.Client) error {
	c.UpdatedAt = time.Now()
	filter := bson.M{"phone": c.Phone}
	_, err := r.col.ReplaceOne(ctx, filter, c, options.Replace().SetUpsert(true))
	return err
}

func (r *ClientRepo) GetByPhone(ctx context.Context, phone string) (*domain.Client, error) {
	var c domain.Client
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ClientRepo) GetByTgID(ctx context.Context, tgID int64) (*domain.Client, error) {
	var c domain.Client
	err := r.col.FindOne(ctx, bson.M{"tg_user_id": tgID}).Decode(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ClientRepo) List(ctx context.Context) ([]*domain.Client, error) {
	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var list []*domain.Client
	for cursor.Next(ctx) {
		var c domain.Client
		if err := cursor.Decode(&c); err != nil {
			return list, err
		}
		list = append(list, &c)
	}
	return list, cursor.Err()
}

func (r *ClientRepo) Block(ctx context.Context, phone string) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"phone": phone}, bson.M{"$set": bson.M{
		"is_blocked": true,
		"updated_at": time.Now(),
	}})
	return err
}

func (r *ClientRepo) InsertMessage(ctx context.Context, msg *domain.ChatMessage) error {
	_, err := r.msgCol.InsertOne(ctx, msg)
	return err
}

func (r *ClientRepo) ListMessages(ctx context.Context, tgID int64) ([]*domain.ChatMessage, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(200)
	cursor, err := r.msgCol.Find(ctx, bson.M{"tg_user_id": tgID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var list []*domain.ChatMessage
	for cursor.Next(ctx) {
		var m domain.ChatMessage
		if err := cursor.Decode(&m); err != nil {
			return list, err
		}
		list = append(list, &m)
	}
	return list, cursor.Err()
}
