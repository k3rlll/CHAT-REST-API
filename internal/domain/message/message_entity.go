package message

import (
	"time"
)

type Message struct {
	Id        int64     `json:"message_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
}
