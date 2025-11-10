package message

import "context"

type MessageRepository interface {
	CheckMessageExists(ctx context.Context, id int) (bool, error)
	DeleteMessage(ctx context.Context, id int) error
	Create(ctx context.Context, chatID int, userID int, text string) (Message, error)
	EditMessage(ctx context.Context, messageID int, newText string) error
	ListByChat(ctx context.Context, chatID int64, limit int, offset int) ([]Message, error)
}
