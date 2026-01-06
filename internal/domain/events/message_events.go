package events

type EventMessageCreated struct {
	MessageID string `json:"message_id"`
	ChatID    string `json:"chat_id"`
	SenderID  string `json:"sender_id"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
}
