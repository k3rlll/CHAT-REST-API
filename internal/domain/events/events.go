package events

import (
	"time"
)

type MessageCreated struct {
	MessageID string    `json:"message_id"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type MessageDeleted struct {
	MessageIDs []string `json:"message_ids"`
	ChatID     int64    `json:"chat_id"`
}
