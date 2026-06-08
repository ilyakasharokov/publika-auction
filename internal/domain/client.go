package domain

import "time"

type Client struct {
	ID          string    `bson:"_id"`
	Phone       string    `bson:"phone"`
	Name        string    `bson:"name"`
	Email       string    `bson:"email"`
	TgUserID    int64     `bson:"tg_user_id"`
	TgUsername  string    `bson:"tg_username"`
	TgFirstName string    `bson:"tg_first_name"`
	TgLastName  string    `bson:"tg_last_name"`
	IsBlocked   bool      `bson:"is_blocked"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

type ChatMessage struct {
	TgUserID  int64     `bson:"tg_user_id"`
	Author    string    `bson:"author"`
	Text      string    `bson:"text"`
	CreatedAt time.Time `bson:"created_at"`
}
