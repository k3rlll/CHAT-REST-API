package events

import "time"

type EventMessageCreated struct {
	MessageID string    `json:"message_id"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
	CreatedAt time.Time `json:"created_at"`
}
