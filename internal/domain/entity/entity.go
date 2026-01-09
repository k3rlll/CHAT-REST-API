package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Chat struct {
	ID              int64     `json:"chat_id" `
	Title            string    `json:"title"`
	IsPrivate        bool      `json:"is_private"`
	CreatedAt        time.Time `json:"created_at"`
	MembersID        []int64   `json:"members"`
	MembersUsernames []string  `json:"members_usernames"`
	MembersCount     int       `json:"members_count"`
}

type ChatMember struct {
	ChatID int64 `json:"chat_id"`
	UserID int64 `json:"user_id"`
}

type Message struct {
	ID             primitive.ObjectID `json:"message_id" bson:"_id,omitempty"`
	Text           string             `json:"text" bson:"text"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	ChatID         int64              `json:"chat_id" bson:"chat_id"`
	SenderID       int64              `json:"sender_id" bson:"sender_id"`
	SenderUsername string             `json:"sender_username" bson:"sender_username"`
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshToken struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"refresh_token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
